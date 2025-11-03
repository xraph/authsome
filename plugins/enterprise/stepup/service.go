package stepup

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/xraph/authsome/core/audit"
)

// Service handles step-up authentication business logic
type Service struct {
	repo         Repository
	config       *Config
	auditService AuditServiceInterface
}

// AuditServiceInterface defines the interface for audit logging
type AuditServiceInterface interface {
	Log(ctx context.Context, event *audit.Event) error
}

// NewService creates a new step-up service
func NewService(repo Repository, config *Config, auditService AuditServiceInterface) *Service {
	if config == nil {
		config = DefaultConfig()
	}
	return &Service{
		repo:         repo,
		config:       config,
		auditService: auditService,
	}
}

// EvaluationContext contains context for evaluating step-up requirements
type EvaluationContext struct {
	UserID       string
	OrgID        string
	SessionID    string
	Route        string
	Method       string
	Amount       float64
	Currency     string
	ResourceType string
	Action       string
	IP           string
	UserAgent    string
	DeviceID     string
	RiskScore    float64
	Metadata     map[string]interface{}
}

// EvaluationResult contains the result of step-up evaluation
type EvaluationResult struct {
	Required          bool                   `json:"required"`
	SecurityLevel     SecurityLevel          `json:"security_level,omitempty"`
	CurrentLevel      SecurityLevel          `json:"current_level"`
	MatchedRules      []string               `json:"matched_rules,omitempty"`
	Reason            string                 `json:"reason,omitempty"`
	RequirementID     string                 `json:"requirement_id,omitempty"`
	ChallengeToken    string                 `json:"challenge_token,omitempty"`
	AllowedMethods    []VerificationMethod   `json:"allowed_methods,omitempty"`
	ExpiresAt         time.Time              `json:"expires_at,omitempty"`
	GracePeriodEndsAt time.Time              `json:"grace_period_ends_at,omitempty"`
	CanRemember       bool                   `json:"can_remember"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
}

// VerifyRequest contains the request to verify step-up authentication
type VerifyRequest struct {
	RequirementID  string             `json:"requirement_id,omitempty"`
	ChallengeToken string             `json:"challenge_token,omitempty"`
	Method         VerificationMethod `json:"method"`
	Credential     string             `json:"credential"`
	RememberDevice bool               `json:"remember_device"`
	DeviceID       string             `json:"device_id,omitempty"`
	DeviceName     string             `json:"device_name,omitempty"`
	IP             string             `json:"ip,omitempty"`
	UserAgent      string             `json:"user_agent,omitempty"`
}

// VerifyResponse contains the response from verification
type VerifyResponse struct {
	Success         bool                   `json:"success"`
	VerificationID  string                 `json:"verification_id,omitempty"`
	SecurityLevel   SecurityLevel          `json:"security_level,omitempty"`
	ExpiresAt       time.Time              `json:"expires_at,omitempty"`
	Error           string                 `json:"error,omitempty"`
	DeviceRemembered bool                  `json:"device_remembered,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// EvaluateRequirement evaluates whether step-up authentication is required
func (s *Service) EvaluateRequirement(ctx context.Context, evalCtx *EvaluationContext) (*EvaluationResult, error) {
	if !s.config.Enabled {
		return &EvaluationResult{
			Required:     false,
			CurrentLevel: SecurityLevelLow,
		}, nil
	}

	// Determine current authentication level
	currentLevel := s.determineCurrentLevel(ctx, evalCtx)

	// Check if device is remembered
	if s.config.RememberStepUp && evalCtx.DeviceID != "" {
		remembered, err := s.repo.GetRememberedDevice(ctx, evalCtx.UserID, evalCtx.OrgID, evalCtx.DeviceID)
		if err == nil && remembered != nil && remembered.ExpiresAt.After(time.Now()) {
			// Update last used time
			remembered.LastUsedAt = time.Now()
			_ = s.repo.UpdateRememberedDevice(ctx, remembered)

			return &EvaluationResult{
				Required:     false,
				CurrentLevel: remembered.SecurityLevel,
				Reason:       "Device is remembered",
			}, nil
		}
	}

	// Evaluate rules in order of specificity
	requiredLevel := SecurityLevelNone
	matchedRules := []string{}
	reason := ""

	// 1. Check organization-specific policies
	policies, _ := s.repo.ListPolicies(ctx, evalCtx.OrgID)
	for _, policy := range policies {
		if level, matches := s.evaluatePolicy(policy, evalCtx); matches {
			if s.isHigherLevel(level, requiredLevel) {
				requiredLevel = level
				matchedRules = append(matchedRules, policy.Name)
				reason = policy.Description
			}
		}
	}

	// 2. Check route rules
	if evalCtx.Route != "" {
		for _, rule := range s.config.RouteRules {
			if s.matchesRoute(rule.Pattern, evalCtx.Route) && (rule.Method == "" || rule.Method == evalCtx.Method) {
				if s.matchesOrg(rule.OrgID, evalCtx.OrgID) {
					if s.isHigherLevel(rule.SecurityLevel, requiredLevel) {
						requiredLevel = rule.SecurityLevel
						matchedRules = append(matchedRules, fmt.Sprintf("Route: %s %s", rule.Method, rule.Pattern))
						reason = rule.Description
					}
				}
			}
		}
	}

	// 3. Check amount rules
	if evalCtx.Amount > 0 {
		for _, rule := range s.config.AmountRules {
			if s.matchesAmount(rule, evalCtx.Amount, evalCtx.Currency) {
				if s.matchesOrg(rule.OrgID, evalCtx.OrgID) {
					if s.isHigherLevel(rule.SecurityLevel, requiredLevel) {
						requiredLevel = rule.SecurityLevel
						matchedRules = append(matchedRules, fmt.Sprintf("Amount: %.2f %s", evalCtx.Amount, evalCtx.Currency))
						reason = rule.Description
					}
				}
			}
		}
	}

	// 4. Check resource rules
	if evalCtx.ResourceType != "" {
		for _, rule := range s.config.ResourceRules {
			if rule.ResourceType == evalCtx.ResourceType && (rule.Action == "" || rule.Action == evalCtx.Action) {
				if s.matchesOrg(rule.OrgID, evalCtx.OrgID) {
					if s.isHigherLevel(rule.SecurityLevel, requiredLevel) {
						requiredLevel = rule.SecurityLevel
						matchedRules = append(matchedRules, fmt.Sprintf("Resource: %s:%s", rule.ResourceType, rule.Action))
						reason = rule.Description
					}
				}
			}
		}
	}

	// 5. Check risk-based rules
	if s.config.RiskBasedEnabled && evalCtx.RiskScore > 0 {
		riskLevel := s.evaluateRiskLevel(evalCtx.RiskScore)
		if s.isHigherLevel(riskLevel, requiredLevel) {
			requiredLevel = riskLevel
			matchedRules = append(matchedRules, fmt.Sprintf("Risk Score: %.2f", evalCtx.RiskScore))
			reason = fmt.Sprintf("Risk-based requirement (score: %.2f)", evalCtx.RiskScore)
		}
	}

	// No step-up required if no rules matched
	if requiredLevel == SecurityLevelNone {
		return &EvaluationResult{
			Required:     false,
			CurrentLevel: currentLevel,
		}, nil
	}

	// Check if current level is sufficient
	if !s.isHigherLevel(requiredLevel, currentLevel) {
		return &EvaluationResult{
			Required:     false,
			CurrentLevel: currentLevel,
			Reason:       "Current authentication level is sufficient",
		}, nil
	}

	// Step-up is required - create requirement
	requirementID := uuid.New().String()
	challengeToken := s.generateChallengeToken()
	expiresAt := time.Now().Add(10 * time.Minute) // 10 minute expiry for challenge

	requirement := &StepUpRequirement{
		ID:             requirementID,
		UserID:         evalCtx.UserID,
		OrgID:          evalCtx.OrgID,
		SessionID:      evalCtx.SessionID,
		RequiredLevel:  requiredLevel,
		CurrentLevel:   currentLevel,
		Route:          evalCtx.Route,
		Method:         evalCtx.Method,
		Amount:         evalCtx.Amount,
		Currency:       evalCtx.Currency,
		ResourceType:   evalCtx.ResourceType,
		ResourceAction: evalCtx.Action,
		RuleName:       strings.Join(matchedRules, ", "),
		Reason:         reason,
		Status:         "pending",
		ChallengeToken: challengeToken,
		IP:             evalCtx.IP,
		UserAgent:      evalCtx.UserAgent,
		RiskScore:      evalCtx.RiskScore,
		Metadata:       evalCtx.Metadata,
		CreatedAt:      time.Now(),
		ExpiresAt:      expiresAt,
	}

	if err := s.repo.CreateRequirement(ctx, requirement); err != nil {
		return nil, fmt.Errorf("failed to create requirement: %w", err)
	}

	// Audit log
	s.auditLog(ctx, evalCtx.UserID, evalCtx.OrgID, "stepup.required", map[string]interface{}{
		"requirement_id":  requirementID,
		"required_level":  requiredLevel,
		"current_level":   currentLevel,
		"matched_rules":   matchedRules,
		"reason":          reason,
		"ip":              evalCtx.IP,
		"user_agent":      evalCtx.UserAgent,
	}, "info")

	return &EvaluationResult{
		Required:          true,
		SecurityLevel:     requiredLevel,
		CurrentLevel:      currentLevel,
		MatchedRules:      matchedRules,
		Reason:            reason,
		RequirementID:     requirementID,
		ChallengeToken:    challengeToken,
		AllowedMethods:    s.getAllowedMethods(requiredLevel),
		ExpiresAt:         expiresAt,
		GracePeriodEndsAt: time.Now().Add(s.config.GracePeriod),
		CanRemember:       s.config.RememberStepUp,
	}, nil
}

// VerifyStepUp verifies a step-up authentication attempt
func (s *Service) VerifyStepUp(ctx context.Context, req *VerifyRequest) (*VerifyResponse, error) {
	// Get requirement
	var requirement *StepUpRequirement
	var err error

	if req.ChallengeToken != "" {
		requirement, err = s.repo.GetRequirementByToken(ctx, req.ChallengeToken)
	} else if req.RequirementID != "" {
		requirement, err = s.repo.GetRequirement(ctx, req.RequirementID)
	} else {
		return &VerifyResponse{
			Success: false,
			Error:   "requirement_id or challenge_token is required",
		}, nil
	}

	if err != nil || requirement == nil {
		return &VerifyResponse{
			Success: false,
			Error:   "Invalid or expired step-up requirement",
		}, nil
	}

	// Check if requirement is still pending
	if requirement.Status != "pending" {
		return &VerifyResponse{
			Success: false,
			Error:   "Step-up requirement is no longer pending",
		}, nil
	}

	// Check if requirement has expired
	if requirement.ExpiresAt.Before(time.Now()) {
		requirement.Status = "expired"
		_ = s.repo.UpdateRequirement(ctx, requirement)
		return &VerifyResponse{
			Success: false,
			Error:   "Step-up requirement has expired",
		}, nil
	}

	// Verify the credential based on method
	verified, err := s.verifyCredential(ctx, requirement, req.Method, req.Credential)
	
	// Record attempt
	attempt := &StepUpAttempt{
		ID:            uuid.New().String(),
		RequirementID: requirement.ID,
		UserID:        requirement.UserID,
		OrgID:         requirement.OrgID,
		Method:        req.Method,
		Success:       verified,
		IP:            req.IP,
		UserAgent:     req.UserAgent,
		CreatedAt:     time.Now(),
	}
	if !verified {
		attempt.FailureReason = "Invalid credential"
	}
	_ = s.repo.CreateAttempt(ctx, attempt)

	if !verified {
		// Audit failed attempt
		s.auditLog(ctx, requirement.UserID, requirement.OrgID, "stepup.failed", map[string]interface{}{
			"requirement_id": requirement.ID,
			"method":         req.Method,
			"ip":             req.IP,
		}, "warning")

		return &VerifyResponse{
			Success: false,
			Error:   "Verification failed",
		}, nil
	}

	// Create verification record
	verificationID := uuid.New().String()
	expiresAt := s.getExpiryTime(requirement.RequiredLevel)

	verification := &StepUpVerification{
		ID:            verificationID,
		UserID:        requirement.UserID,
		OrgID:         requirement.OrgID,
		SessionID:     requirement.SessionID,
		SecurityLevel: requirement.RequiredLevel,
		Method:        req.Method,
		IP:            req.IP,
		UserAgent:     req.UserAgent,
		DeviceID:      req.DeviceID,
		Reason:        requirement.Reason,
		RuleName:      requirement.RuleName,
		VerifiedAt:    time.Now(),
		ExpiresAt:     expiresAt,
		Metadata:      requirement.Metadata,
		CreatedAt:     time.Now(),
	}

	if err := s.repo.CreateVerification(ctx, verification); err != nil {
		return &VerifyResponse{
			Success: false,
			Error:   "Failed to create verification record",
		}, fmt.Errorf("failed to create verification: %w", err)
	}

	// Update requirement status
	now := time.Now()
	requirement.Status = "fulfilled"
	requirement.FulfilledAt = &now
	_ = s.repo.UpdateRequirement(ctx, requirement)

	// Remember device if requested
	deviceRemembered := false
	if req.RememberDevice && s.config.RememberStepUp && req.DeviceID != "" {
		device := &StepUpRememberedDevice{
			ID:            uuid.New().String(),
			UserID:        requirement.UserID,
			OrgID:         requirement.OrgID,
			DeviceID:      req.DeviceID,
			DeviceName:    req.DeviceName,
			SecurityLevel: requirement.RequiredLevel,
			IP:            req.IP,
			UserAgent:     req.UserAgent,
			RememberedAt:  time.Now(),
			ExpiresAt:     time.Now().Add(s.config.RememberDuration),
			LastUsedAt:    time.Now(),
			CreatedAt:     time.Now(),
		}
		if err := s.repo.CreateRememberedDevice(ctx, device); err == nil {
			deviceRemembered = true
		}
	}

	// Audit successful verification
	s.auditLog(ctx, requirement.UserID, requirement.OrgID, "stepup.verified", map[string]interface{}{
		"requirement_id":    requirement.ID,
		"verification_id":   verificationID,
		"method":            req.Method,
		"security_level":    requirement.RequiredLevel,
		"device_remembered": deviceRemembered,
		"ip":                req.IP,
	}, "info")

	return &VerifyResponse{
		Success:         true,
		VerificationID:  verificationID,
		SecurityLevel:   requirement.RequiredLevel,
		ExpiresAt:       expiresAt,
		DeviceRemembered: deviceRemembered,
	}, nil
}

// Helper methods

func (s *Service) determineCurrentLevel(ctx context.Context, evalCtx *EvaluationContext) SecurityLevel {
	// Check for recent verifications
	if verification, err := s.repo.GetLatestVerification(ctx, evalCtx.UserID, evalCtx.OrgID, SecurityLevelCritical); err == nil && verification != nil {
		if verification.ExpiresAt.After(time.Now()) {
			return SecurityLevelCritical
		}
	}

	if verification, err := s.repo.GetLatestVerification(ctx, evalCtx.UserID, evalCtx.OrgID, SecurityLevelHigh); err == nil && verification != nil {
		if verification.ExpiresAt.After(time.Now()) {
			return SecurityLevelHigh
		}
	}

	if verification, err := s.repo.GetLatestVerification(ctx, evalCtx.UserID, evalCtx.OrgID, SecurityLevelMedium); err == nil && verification != nil {
		if verification.ExpiresAt.After(time.Now()) {
			return SecurityLevelMedium
		}
	}

	// Default to low if user is authenticated
	return SecurityLevelLow
}

func (s *Service) evaluatePolicy(policy *StepUpPolicy, evalCtx *EvaluationContext) (SecurityLevel, bool) {
	// This would use a proper policy evaluation engine (e.g., CEL)
	// For now, return none to indicate no match
	// TODO: Implement policy evaluation
	return SecurityLevelNone, false
}

func (s *Service) matchesRoute(pattern, route string) bool {
	// Simple wildcard matching
	if pattern == route {
		return true
	}

	// Support trailing wildcard
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(route, prefix)
	}

	// Use path.Match for glob patterns
	matched, _ := path.Match(pattern, route)
	return matched
}

func (s *Service) matchesAmount(rule AmountRule, amount float64, currency string) bool {
	if rule.Currency != "" && rule.Currency != currency {
		return false
	}
	if amount < rule.MinAmount {
		return false
	}
	if rule.MaxAmount > 0 && amount > rule.MaxAmount {
		return false
	}
	return true
}

func (s *Service) matchesOrg(ruleOrgID, contextOrgID string) bool {
	// Empty ruleOrgID means global rule
	return ruleOrgID == "" || ruleOrgID == contextOrgID
}

func (s *Service) isHigherLevel(level1, level2 SecurityLevel) bool {
	levels := map[SecurityLevel]int{
		SecurityLevelNone:     0,
		SecurityLevelLow:      1,
		SecurityLevelMedium:   2,
		SecurityLevelHigh:     3,
		SecurityLevelCritical: 4,
	}
	return levels[level1] > levels[level2]
}

func (s *Service) evaluateRiskLevel(riskScore float64) SecurityLevel {
	if riskScore >= s.config.RiskThresholdHigh {
		return SecurityLevelCritical
	}
	if riskScore >= s.config.RiskThresholdMedium {
		return SecurityLevelHigh
	}
	if riskScore >= s.config.RiskThresholdLow {
		return SecurityLevelMedium
	}
	return SecurityLevelLow
}

func (s *Service) getAllowedMethods(level SecurityLevel) []VerificationMethod {
	switch level {
	case SecurityLevelLow:
		return s.config.LowMethods
	case SecurityLevelMedium:
		return s.config.MediumMethods
	case SecurityLevelHigh:
		return s.config.HighMethods
	case SecurityLevelCritical:
		return s.config.CriticalMethods
	default:
		return []VerificationMethod{MethodPassword}
	}
}

func (s *Service) getExpiryTime(level SecurityLevel) time.Time {
	switch level {
	case SecurityLevelMedium:
		return time.Now().Add(s.config.MediumAuthWindow)
	case SecurityLevelHigh:
		return time.Now().Add(s.config.HighAuthWindow)
	case SecurityLevelCritical:
		return time.Now().Add(s.config.CriticalAuthWindow)
	default:
		return time.Now().Add(15 * time.Minute)
	}
}

func (s *Service) generateChallengeToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

func (s *Service) verifyCredential(ctx context.Context, requirement *StepUpRequirement, method VerificationMethod, credential string) (bool, error) {
	// This should integrate with actual verification services
	// For now, return true for demonstration
	// TODO: Implement actual verification logic with:
	// - Password verification via UserService
	// - TOTP verification via TwoFAService
	// - SMS/Email verification via respective services
	// - WebAuthn verification via PasskeyService
	
	// Placeholder: In production, integrate with actual verification services
	return len(credential) > 0, nil
}

func (s *Service) auditLog(ctx context.Context, userID, orgID, eventType string, data map[string]interface{}, severity string) {
	if !s.config.AuditEnabled || s.auditService == nil {
		return
	}

	log := &StepUpAuditLog{
		ID:        uuid.New().String(),
		UserID:    userID,
		OrgID:     orgID,
		EventType: eventType,
		EventData: data,
		Severity:  severity,
		CreatedAt: time.Now(),
	}

	if ip, ok := data["ip"].(string); ok {
		log.IP = ip
	}
	if ua, ok := data["user_agent"].(string); ok {
		log.UserAgent = ua
	}

	_ = s.repo.CreateAuditLog(ctx, log)
}

// ForgetDevice removes a remembered device
func (s *Service) ForgetDevice(ctx context.Context, userID, orgID, deviceID string) error {
	device, err := s.repo.GetRememberedDevice(ctx, userID, orgID, deviceID)
	if err != nil {
		return fmt.Errorf("device not found: %w", err)
	}

	if err := s.repo.DeleteRememberedDevice(ctx, device.ID); err != nil {
		return fmt.Errorf("failed to forget device: %w", err)
	}

	s.auditLog(ctx, userID, orgID, "stepup.device_forgotten", map[string]interface{}{
		"device_id": deviceID,
	}, "info")

	return nil
}

// CleanupExpired removes expired requirements and devices
func (s *Service) CleanupExpired(ctx context.Context) error {
	if err := s.repo.DeleteExpiredRequirements(ctx); err != nil {
		return fmt.Errorf("failed to cleanup expired requirements: %w", err)
	}

	if err := s.repo.DeleteExpiredRememberedDevices(ctx); err != nil {
		return fmt.Errorf("failed to cleanup expired devices: %w", err)
	}

	return nil
}

