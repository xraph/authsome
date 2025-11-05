package mfa

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/repository"
	"github.com/xraph/authsome/schema"
)

// Service provides MFA orchestration and management
type Service struct {
	repo            *repository.MFARepository
	adapterRegistry *FactorAdapterRegistry
	riskEngine      *RiskEngine
	rateLimiter     *RateLimiter
	config          *Config
}

// NewService creates a new MFA service
func NewService(
	repo *repository.MFARepository,
	adapterRegistry *FactorAdapterRegistry,
	config *Config,
) *Service {
	// Validate config
	_ = config.Validate()

	return &Service{
		repo:            repo,
		adapterRegistry: adapterRegistry,
		riskEngine:      NewRiskEngine(&config.AdaptiveMFA, repo),
		rateLimiter:     NewRateLimiter(&config.RateLimit, repo),
		config:          config,
	}
}

// ==================== Factor Enrollment ====================

// EnrollFactor initiates factor enrollment for a user
func (s *Service) EnrollFactor(ctx context.Context, userID xid.ID, req *FactorEnrollmentRequest) (*FactorEnrollmentResponse, error) {
	// Check if factor type is allowed
	if !s.config.IsFactorAllowed(req.Type) {
		return nil, fmt.Errorf("factor type %s not allowed", req.Type)
	}

	// Get adapter for this factor type
	adapter, err := s.adapterRegistry.Get(req.Type)
	if err != nil {
		return nil, fmt.Errorf("factor type not supported: %w", err)
	}

	// Check if adapter is available
	if !adapter.IsAvailable() {
		return nil, fmt.Errorf("factor type %s not available", req.Type)
	}

	// Use adapter to initiate enrollment
	resp, err := adapter.Enroll(ctx, userID, req.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to enroll factor: %w", err)
	}

	// Set defaults
	if req.Priority == "" {
		req.Priority = FactorPriorityPrimary
	}
	if req.Name == "" {
		req.Name = fmt.Sprintf("%s Factor", req.Type)
	}

	// Create factor record
	factor := &schema.MFAFactor{
		ID:       resp.FactorID,
		UserID:   userID,
		Type:     string(req.Type),
		Status:   string(resp.Status),
		Priority: string(req.Priority),
		Name:     req.Name,
		Metadata: req.Metadata,
	}

	// Store secret if provided (encrypted)
	if secret, ok := resp.ProvisioningData["secret"].(string); ok {
		// Encrypt secret before storing
		encrypted, err := s.encryptSecret(secret)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt secret: %w", err)
		}
		factor.Secret = encrypted
	}

	// Set audit fields
	factor.AuditableModel.CreatedBy = userID
	factor.AuditableModel.UpdatedBy = userID

	// Save to database
	if err := s.repo.CreateFactor(ctx, factor); err != nil {
		return nil, fmt.Errorf("failed to save factor: %w", err)
	}

	return resp, nil
}

// VerifyEnrollment completes factor enrollment verification
func (s *Service) VerifyEnrollment(ctx context.Context, factorID xid.ID, proof string) error {
	// Get the factor
	factor, err := s.repo.GetFactor(ctx, factorID)
	if err != nil {
		return fmt.Errorf("failed to get factor: %w", err)
	}
	if factor == nil {
		return fmt.Errorf("factor not found")
	}

	// Check if already verified
	if factor.Status == string(FactorStatusActive) {
		return fmt.Errorf("factor already verified")
	}

	// Get adapter
	adapter, err := s.adapterRegistry.Get(FactorType(factor.Type))
	if err != nil {
		return err
	}

	// Verify enrollment
	if err := adapter.VerifyEnrollment(ctx, factorID, proof); err != nil {
		return fmt.Errorf("verification failed: %w", err)
	}

	// Update factor status
	now := time.Now()
	factor.Status = string(FactorStatusActive)
	factor.VerifiedAt = &now
	factor.AuditableModel.UpdatedBy = factor.UserID

	return s.repo.UpdateFactor(ctx, factor)
}

// ListFactors lists all factors for a user
func (s *Service) ListFactors(ctx context.Context, userID xid.ID, activeOnly bool) ([]*Factor, error) {
	var statusFilter []string
	if activeOnly {
		statusFilter = []string{string(FactorStatusActive)}
	}

	schemaFactors, err := s.repo.ListUserFactors(ctx, userID, statusFilter...)
	if err != nil {
		return nil, err
	}

	// Convert to response format
	factors := make([]*Factor, len(schemaFactors))
	for i, sf := range schemaFactors {
		factors[i] = s.convertSchemaFactor(sf)
	}

	return factors, nil
}

// GetFactor retrieves a specific factor
func (s *Service) GetFactor(ctx context.Context, factorID xid.ID) (*Factor, error) {
	schemaFactor, err := s.repo.GetFactor(ctx, factorID)
	if err != nil {
		return nil, err
	}
	if schemaFactor == nil {
		return nil, fmt.Errorf("factor not found")
	}

	return s.convertSchemaFactor(schemaFactor), nil
}

// UpdateFactor updates factor settings
func (s *Service) UpdateFactor(ctx context.Context, factorID xid.ID, updates map[string]interface{}) error {
	factor, err := s.repo.GetFactor(ctx, factorID)
	if err != nil {
		return err
	}
	if factor == nil {
		return fmt.Errorf("factor not found")
	}

	// Apply updates
	if name, ok := updates["name"].(string); ok {
		factor.Name = name
	}
	if priority, ok := updates["priority"].(string); ok {
		factor.Priority = priority
	}
	if metadata, ok := updates["metadata"].(map[string]any); ok {
		factor.Metadata = metadata
	}

	factor.AuditableModel.UpdatedBy = factor.UserID
	return s.repo.UpdateFactor(ctx, factor)
}

// DeleteFactor removes a factor
func (s *Service) DeleteFactor(ctx context.Context, factorID xid.ID) error {
	// Get factor to check it exists
	factor, err := s.repo.GetFactor(ctx, factorID)
	if err != nil {
		return err
	}
	if factor == nil {
		return fmt.Errorf("factor not found")
	}

	// Check if this is the last active factor
	activeFactors, err := s.repo.ListUserFactors(ctx, factor.UserID, string(FactorStatusActive))
	if err != nil {
		return err
	}

	if len(activeFactors) <= 1 {
		return fmt.Errorf("cannot delete last active factor")
	}

	return s.repo.DeleteFactor(ctx, factorID)
}

// ==================== Challenge & Verification ====================

// InitiateChallenge starts an MFA verification challenge
func (s *Service) InitiateChallenge(ctx context.Context, req *ChallengeRequest) (*ChallengeResponse, error) {
	// Check rate limits
	limitResult, err := s.rateLimiter.CheckUserLimit(ctx, req.UserID)
	if err != nil {
		return nil, err
	}
	if !limitResult.Allowed {
		return nil, fmt.Errorf("rate limit exceeded, try again in %v", limitResult.RetryAfter)
	}

	// Perform risk assessment
	riskCtx := &RiskContext{
		UserID:    req.UserID,
		IPAddress: getString(req.Metadata, "ip_address"),
		UserAgent: getString(req.Metadata, "user_agent"),
		DeviceID:  getString(req.Metadata, "device_id"),
		Timestamp: time.Now(),
	}

	riskAssessment, err := s.riskEngine.AssessRisk(ctx, riskCtx)
	if err != nil {
		return nil, fmt.Errorf("risk assessment failed: %w", err)
	}

	// Save risk assessment
	riskRecord := &schema.MFARiskAssessment{
		ID:          xid.New(),
		UserID:      req.UserID,
		RiskLevel:   string(riskAssessment.Level),
		RiskScore:   riskAssessment.Score,
		Factors:     riskAssessment.Factors,
		Recommended: convertFactorTypes(riskAssessment.Recommended),
		IPAddress:   riskCtx.IPAddress,
		UserAgent:   riskCtx.UserAgent,
		Metadata:    riskAssessment.Metadata,
	}
	riskRecord.AuditableModel.CreatedBy = req.UserID
	riskRecord.AuditableModel.UpdatedBy = req.UserID
	_ = s.repo.CreateRiskAssessment(ctx, riskRecord)

	// Determine required factor count based on risk
	factorsRequired := s.config.RequiredFactorCount
	if s.config.AdaptiveMFA.Enabled {
		riskBasedCount := s.riskEngine.GetRequiredFactorCount(riskAssessment.Level)
		if riskBasedCount > factorsRequired {
			factorsRequired = riskBasedCount
		}
	}

	// Get user's active factors
	factors, err := s.ListFactors(ctx, req.UserID, true)
	if err != nil {
		return nil, err
	}

	if len(factors) == 0 {
		return nil, fmt.Errorf("no factors enrolled")
	}

	// Filter by requested factor types if specified
	var availableFactors []FactorInfo
	for _, f := range factors {
		// Check if factor type is in requested types (if specified)
		if len(req.FactorTypes) > 0 {
			found := false
			for _, ft := range req.FactorTypes {
				if f.Type == ft {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		availableFactors = append(availableFactors, FactorInfo{
			FactorID: f.ID,
			Type:     f.Type,
			Name:     f.Name,
			Metadata: s.maskSensitiveData(f.Metadata),
		})
	}

	if len(availableFactors) == 0 {
		return nil, fmt.Errorf("no suitable factors available")
	}

	// Create MFA session
	sessionToken, err := s.generateSessionToken()
	if err != nil {
		return nil, err
	}

	session := &schema.MFASession{
		ID:              xid.New(),
		UserID:          req.UserID,
		SessionToken:    sessionToken,
		FactorsRequired: factorsRequired,
		FactorsVerified: 0,
		RiskLevel:       string(riskAssessment.Level),
		RiskScore:       riskAssessment.Score,
		Context:         req.Context,
		IPAddress:       riskCtx.IPAddress,
		UserAgent:       riskCtx.UserAgent,
		Metadata:        req.Metadata,
		ExpiresAt:       time.Now().Add(time.Duration(s.config.SessionExpiryMinutes) * time.Minute),
	}
	session.AuditableModel.CreatedBy = req.UserID
	session.AuditableModel.UpdatedBy = req.UserID

	if err := s.repo.CreateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return &ChallengeResponse{
		ChallengeID:      xid.New(), // This will be created when user selects a factor
		SessionID:        session.ID,
		FactorsRequired:  factorsRequired,
		AvailableFactors: availableFactors,
		ExpiresAt:        session.ExpiresAt,
	}, nil
}

// VerifyChallenge verifies a challenge response
func (s *Service) VerifyChallenge(ctx context.Context, req *VerificationRequest) (*VerificationResponse, error) {
	// Get the session
	session, err := s.repo.GetSession(ctx, req.ChallengeID) // Using challenge ID as session lookup for now
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, fmt.Errorf("invalid session")
	}

	// Check if session expired
	if time.Now().After(session.ExpiresAt) {
		return nil, fmt.Errorf("session expired")
	}

	// Check rate limits
	limitResult, err := s.rateLimiter.CheckFactorLimit(ctx, session.UserID, FactorType(session.Context))
	if err != nil {
		return nil, err
	}
	if !limitResult.Allowed {
		return nil, fmt.Errorf("rate limit exceeded")
	}

	// Get the factor
	factor, err := s.repo.GetFactor(ctx, req.FactorID)
	if err != nil {
		return nil, err
	}
	if factor == nil {
		return nil, fmt.Errorf("factor not found")
	}

	// Verify factor belongs to user
	if factor.UserID != session.UserID {
		return nil, fmt.Errorf("factor does not belong to user")
	}

	// Get adapter
	adapter, err := s.adapterRegistry.Get(FactorType(factor.Type))
	if err != nil {
		return nil, err
	}

	// Create challenge for this factor
	challenge := &Challenge{
		ID:       xid.New(),
		UserID:   session.UserID,
		FactorID: req.FactorID,
		Type:     FactorType(factor.Type),
		Status:   ChallengeStatusPending,
	}

	// Verify using adapter
	valid, err := adapter.Verify(ctx, challenge, req.Code, req.Data)
	if err != nil {
		// Record failed attempt
		_ = s.rateLimiter.RecordAttempt(ctx, session.UserID, &req.FactorID, FactorType(factor.Type), false, map[string]string{
			"failure_reason": err.Error(),
		})
		return nil, fmt.Errorf("verification failed: %w", err)
	}

	if !valid {
		// Record failed attempt
		_ = s.rateLimiter.RecordAttempt(ctx, session.UserID, &req.FactorID, FactorType(factor.Type), false, map[string]string{
			"failure_reason": "invalid code",
		})
		return &VerificationResponse{
			Success:          false,
			SessionComplete:  false,
			FactorsRemaining: session.FactorsRequired - session.FactorsVerified,
		}, nil
	}

	// Record successful attempt
	_ = s.rateLimiter.RecordAttempt(ctx, session.UserID, &req.FactorID, FactorType(factor.Type), true, nil)

	// Update factor last used
	_ = s.repo.UpdateFactorLastUsed(ctx, req.FactorID)

	// Update session
	session.FactorsVerified++
	session.VerifiedFactors = append(session.VerifiedFactors, req.FactorID.String())

	sessionComplete := session.FactorsVerified >= session.FactorsRequired
	if sessionComplete {
		now := time.Now()
		session.CompletedAt = &now
	}

	if err := s.repo.UpdateSession(ctx, session); err != nil {
		return nil, err
	}

	// Handle trusted device if requested
	if req.RememberDevice && req.DeviceInfo != nil && sessionComplete {
		_ = s.TrustDevice(ctx, session.UserID, req.DeviceInfo)
	}

	resp := &VerificationResponse{
		Success:          true,
		SessionComplete:  sessionComplete,
		FactorsRemaining: session.FactorsRequired - session.FactorsVerified,
	}

	if sessionComplete {
		resp.Token = session.SessionToken
		expiresAt := session.ExpiresAt
		resp.ExpiresAt = &expiresAt
	}

	return resp, nil
}

// ==================== Trusted Devices ====================

// TrustDevice marks a device as trusted
func (s *Service) TrustDevice(ctx context.Context, userID xid.ID, deviceInfo *DeviceInfo) error {
	if !s.config.TrustedDevices.Enabled {
		return fmt.Errorf("trusted devices not enabled")
	}

	// Check if device already trusted
	existing, err := s.repo.GetTrustedDevice(ctx, userID, deviceInfo.DeviceID)
	if err != nil {
		return err
	}

	if existing != nil {
		// Update existing
		existing.LastUsedAt = ptrTime(time.Now())
		return s.repo.UpdateTrustedDevice(ctx, existing)
	}

	// Create new trusted device
	device := &schema.MFATrustedDevice{
		ID:        xid.New(),
		UserID:    userID,
		DeviceID:  deviceInfo.DeviceID,
		Name:      deviceInfo.Name,
		Metadata:  deviceInfo.Metadata,
		ExpiresAt: time.Now().Add(time.Duration(s.config.TrustedDevices.DefaultExpiryDays) * 24 * time.Hour),
	}
	device.AuditableModel.CreatedBy = userID
	device.AuditableModel.UpdatedBy = userID

	return s.repo.CreateTrustedDevice(ctx, device)
}

// IsTrustedDevice checks if a device is trusted
func (s *Service) IsTrustedDevice(ctx context.Context, userID xid.ID, deviceID string) (bool, error) {
	if !s.config.TrustedDevices.Enabled {
		return false, nil
	}

	device, err := s.repo.GetTrustedDevice(ctx, userID, deviceID)
	if err != nil {
		return false, err
	}

	if device != nil {
		// Update last used
		_ = s.repo.UpdateDeviceLastUsed(ctx, device.ID)
		return true, nil
	}

	return false, nil
}

// ListTrustedDevices lists all trusted devices for a user
func (s *Service) ListTrustedDevices(ctx context.Context, userID xid.ID) ([]*TrustedDevice, error) {
	schemaDevices, err := s.repo.ListTrustedDevices(ctx, userID)
	if err != nil {
		return nil, err
	}

	devices := make([]*TrustedDevice, len(schemaDevices))
	for i, sd := range schemaDevices {
		devices[i] = &TrustedDevice{
			ID:         sd.ID,
			UserID:     sd.UserID,
			DeviceID:   sd.DeviceID,
			Name:       sd.Name,
			Metadata:   sd.Metadata,
			LastUsedAt: sd.LastUsedAt,
			CreatedAt:  sd.CreatedAt,
			ExpiresAt:  sd.ExpiresAt,
		}
	}

	return devices, nil
}

// RevokeTrustedDevice removes trust from a device
func (s *Service) RevokeTrustedDevice(ctx context.Context, deviceID xid.ID) error {
	return s.repo.DeleteTrustedDevice(ctx, deviceID)
}

// ==================== Status & Policy ====================

// GetMFAStatus returns the MFA status for a user
func (s *Service) GetMFAStatus(ctx context.Context, userID xid.ID, deviceID string) (*MFAStatus, error) {
	factors, err := s.ListFactors(ctx, userID, true)
	if err != nil {
		return nil, err
	}

	factorInfos := make([]FactorInfo, len(factors))
	for i, f := range factors {
		factorInfos[i] = FactorInfo{
			FactorID: f.ID,
			Type:     f.Type,
			Name:     f.Name,
			Metadata: s.maskSensitiveData(f.Metadata),
		}
	}

	trustedDevice := false
	if deviceID != "" {
		trustedDevice, _ = s.IsTrustedDevice(ctx, userID, deviceID)
	}

	return &MFAStatus{
		Enabled:         len(factors) > 0,
		EnrolledFactors: factorInfos,
		RequiredCount:   s.config.RequiredFactorCount,
		PolicyActive:    s.config.Enabled,
		TrustedDevice:   trustedDevice,
	}, nil
}

// ==================== Helper Methods ====================

func (s *Service) convertSchemaFactor(sf *schema.MFAFactor) *Factor {
	return &Factor{
		ID:         sf.ID,
		UserID:     sf.UserID,
		Type:       FactorType(sf.Type),
		Status:     FactorStatus(sf.Status),
		Priority:   FactorPriority(sf.Priority),
		Name:       sf.Name,
		Metadata:   sf.Metadata,
		LastUsedAt: sf.LastUsedAt,
		VerifiedAt: sf.VerifiedAt,
		CreatedAt:  sf.CreatedAt,
		UpdatedAt:  sf.UpdatedAt.Time,
		ExpiresAt:  sf.ExpiresAt,
	}
}

func (s *Service) maskSensitiveData(metadata map[string]any) map[string]any {
	masked := make(map[string]any)
	for k, v := range metadata {
		// Mask sensitive fields
		if k == "email" {
			if email, ok := v.(string); ok {
				masked[k] = maskEmail(email)
			}
		} else if k == "phone" {
			if phone, ok := v.(string); ok {
				masked[k] = maskPhone(phone)
			}
		} else {
			masked[k] = v
		}
	}
	return masked
}

func (s *Service) encryptSecret(secret string) (string, error) {
	// Simple encryption - in production use proper key management
	// For now, just hex encode (placeholder)
	return hex.EncodeToString([]byte(secret)), nil
}

func (s *Service) generateSessionToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func convertFactorTypes(types []FactorType) []string {
	result := make([]string, len(types))
	for i, t := range types {
		result[i] = string(t)
	}
	return result
}

func ptrTime(t time.Time) *time.Time {
	return &t
}
