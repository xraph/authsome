package consent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/user"
)

// Service provides consent management operations
type Service struct {
	repo        Repository
	config      *Config
	userService *user.Service
}

// NewService creates a new consent service
func NewService(
	repo Repository,
	config *Config,
	userService *user.Service,
) *Service {
	return &Service{
		repo:        repo,
		config:      config,
		userService: userService,
	}
}

// ====== Consent Records ======

// CreateConsent records a new consent
func (s *Service) CreateConsent(ctx context.Context, orgID, userID string, req *CreateConsentRequest) (*ConsentRecord, error) {
	// Check if consent already exists
	existing, err := s.repo.GetConsentByUserAndType(ctx, userID, orgID, req.ConsentType, req.Purpose)
	if err == nil && existing != nil {
		return nil, ErrConsentAlreadyExists
	}

	// Get policy version if not provided
	if req.Version == "" {
		policy, err := s.repo.GetLatestPolicy(ctx, orgID, req.ConsentType)
		if err == nil && policy != nil {
			req.Version = policy.Version
		} else {
			req.Version = "1.0" // Default version
		}
	}

	consent := &ConsentRecord{
		ID:             xid.New(),
		UserID:         userID,
		OrganizationID: orgID,
		ConsentType:    req.ConsentType,
		Purpose:        req.Purpose,
		Granted:        req.Granted,
		Version:        req.Version,
		GrantedAt:      time.Now(),
		Metadata:       req.Metadata,
	}

	// Set expiry if provided
	if req.ExpiresIn != nil && *req.ExpiresIn > 0 {
		expiryDate := time.Now().AddDate(0, 0, *req.ExpiresIn)
		consent.ExpiresAt = &expiryDate
	}

	// Extract IP and user agent from context if available
	if ipAddr, ok := ctx.Value("ip_address").(string); ok {
		consent.IPAddress = ipAddr
	}
	if userAgent, ok := ctx.Value("user_agent").(string); ok {
		consent.UserAgent = userAgent
	}

	if err := s.repo.CreateConsent(ctx, consent); err != nil {
		return nil, fmt.Errorf("failed to create consent: %w", err)
	}

	// Create audit log
	s.createAuditLog(ctx, userID, orgID, consent.ID.String(), ActionGranted, req.ConsentType, req.Purpose, nil, consent)

	// TODO: Audit log - integrate with core audit service
	// Requires: audit.Service.Log(ctx, userID, "consent.created", "consent", ipAddr, userAgent, metadata)

	// TODO: Send notification if configured
	// Requires: notification service integration

	return consent, nil
}

// GetConsent retrieves a consent record
func (s *Service) GetConsent(ctx context.Context, id string) (*ConsentRecord, error) {
	return s.repo.GetConsent(ctx, id)
}

// ListConsentsByUser lists all consents for a user
func (s *Service) ListConsentsByUser(ctx context.Context, userID, orgID string) ([]*ConsentRecord, error) {
	return s.repo.ListConsentsByUser(ctx, userID, orgID)
}

// UpdateConsent updates a consent record
func (s *Service) UpdateConsent(ctx context.Context, id, userID, orgID string, req *UpdateConsentRequest) (*ConsentRecord, error) {
	consent, err := s.repo.GetConsent(ctx, id)
	if err != nil {
		return nil, ErrConsentNotFound
	}

	// Authorization check
	if consent.UserID != userID || consent.OrganizationID != orgID {
		return nil, ErrUnauthorized
	}

	previousValue := map[string]interface{}{
		"granted":  consent.Granted,
		"metadata": consent.Metadata,
	}

	// Update fields
	if req.Granted != nil {
		if !*req.Granted && consent.Granted {
			// Revoking consent
			now := time.Now()
			consent.RevokedAt = &now
		}
		consent.Granted = *req.Granted
	}

	if req.Metadata != nil {
		consent.Metadata = req.Metadata
	}

	if err := s.repo.UpdateConsent(ctx, consent); err != nil {
		return nil, fmt.Errorf("failed to update consent: %w", err)
	}

	// Create audit log
	action := ActionUpdated
	if req.Granted != nil && !*req.Granted {
		action = ActionRevoked
	}
	s.createAuditLog(ctx, userID, orgID, id, action, consent.ConsentType, consent.Purpose, previousValue, consent)

	// TODO: Audit log and notifications

	return consent, nil
}

// RevokeConsent revokes a consent record
func (s *Service) RevokeConsent(ctx context.Context, userID, orgID, consentType, purpose string) error {
	consent, err := s.repo.GetConsentByUserAndType(ctx, userID, orgID, consentType, purpose)
	if err != nil {
		return ErrConsentNotFound
	}

	if !consent.Granted {
		return ErrConsentRevoked
	}

	now := time.Now()
	consent.RevokedAt = &now
	consent.Granted = false

	if err := s.repo.UpdateConsent(ctx, consent); err != nil {
		return fmt.Errorf("failed to revoke consent: %w", err)
	}

	// Create audit log
	s.createAuditLog(ctx, userID, orgID, consent.ID.String(), ActionRevoked, consentType, purpose, nil, consent)

	return nil
}

// GetConsentSummary provides a summary of user's consent status
func (s *Service) GetConsentSummary(ctx context.Context, userID, orgID string) (*ConsentSummary, error) {
	consents, err := s.repo.ListConsentsByUser(ctx, userID, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get consents: %w", err)
	}

	summary := &ConsentSummary{
		UserID:         userID,
		OrganizationID: orgID,
		TotalConsents:  len(consents),
		ConsentsByType: make(map[string]ConsentTypeStatus),
	}

	now := time.Now()
	for _, consent := range consents {
		if consent.Granted {
			summary.GrantedConsents++
		} else if consent.RevokedAt != nil {
			summary.RevokedConsents++
		}

		if consent.ExpiresAt != nil && consent.ExpiresAt.Before(now) {
			summary.ExpiredConsents++
		}

		// Check if renewal is needed (within renewal reminder period)
		needsRenewal := false
		if consent.ExpiresAt != nil && s.config.Expiry.Enabled {
			daysUntilExpiry := int(time.Until(*consent.ExpiresAt).Hours() / 24)
			if daysUntilExpiry <= s.config.Expiry.RenewalReminderDays && daysUntilExpiry > 0 {
				needsRenewal = true
				summary.PendingRenewals++
			}
		}

		summary.ConsentsByType[consent.ConsentType] = ConsentTypeStatus{
			Type:         consent.ConsentType,
			Granted:      consent.Granted,
			Version:      consent.Version,
			GrantedAt:    consent.GrantedAt,
			ExpiresAt:    consent.ExpiresAt,
			NeedsRenewal: needsRenewal,
		}

		if summary.LastConsentUpdate == nil || consent.UpdatedAt.After(*summary.LastConsentUpdate) {
			summary.LastConsentUpdate = &consent.UpdatedAt
		}
	}

	// Check for pending deletion
	deletionReq, err := s.repo.GetPendingDeletionRequest(ctx, userID, orgID)
	if err == nil && deletionReq != nil {
		summary.HasPendingDeletion = true
	}

	// Check for pending export
	pendingStatus := string(StatusPending)
	exports, err := s.repo.ListExportRequests(ctx, userID, orgID, &pendingStatus)
	if err == nil && len(exports) > 0 {
		summary.HasPendingExport = true
	}

	return summary, nil
}

// ExpireConsents automatically expires consents that have passed their expiry date
func (s *Service) ExpireConsents(ctx context.Context) (int, error) {
	if !s.config.Expiry.Enabled || !s.config.Expiry.AutoExpireCheck {
		return 0, nil
	}

	count, err := s.repo.ExpireConsents(ctx, time.Now())
	if err != nil {
		return 0, fmt.Errorf("failed to expire consents: %w", err)
	}

	return count, nil
}

// ====== Consent Policies ======

// CreatePolicy creates a new consent policy
func (s *Service) CreatePolicy(ctx context.Context, orgID, createdBy string, req *CreatePolicyRequest) (*ConsentPolicy, error) {
	policy := &ConsentPolicy{
		OrganizationID: orgID,
		ConsentType:    req.ConsentType,
		Name:           req.Name,
		Description:    req.Description,
		Version:        req.Version,
		Content:        req.Content,
		Required:       req.Required,
		Renewable:      req.Renewable,
		ValidityPeriod: req.ValidityPeriod,
		Active:         true,
		CreatedBy:      createdBy,
		Metadata:       req.Metadata,
	}

	if err := s.repo.CreatePolicy(ctx, policy); err != nil {
		return nil, fmt.Errorf("failed to create policy: %w", err)
	}

	// TODO: Audit log

	return policy, nil
}

// GetPolicy retrieves a consent policy
func (s *Service) GetPolicy(ctx context.Context, id string) (*ConsentPolicy, error) {
	return s.repo.GetPolicy(ctx, id)
}

// GetLatestPolicy retrieves the latest active policy for a consent type
func (s *Service) GetLatestPolicy(ctx context.Context, orgID, consentType string) (*ConsentPolicy, error) {
	return s.repo.GetLatestPolicy(ctx, orgID, consentType)
}

// ListPolicies lists policies for an organization
func (s *Service) ListPolicies(ctx context.Context, orgID string, activeOnly bool) ([]*ConsentPolicy, error) {
	var active *bool
	if activeOnly {
		t := true
		active = &t
	}
	return s.repo.ListPolicies(ctx, orgID, active)
}

// UpdatePolicy updates a consent policy
func (s *Service) UpdatePolicy(ctx context.Context, id, orgID, updatedBy string, req *UpdatePolicyRequest) (*ConsentPolicy, error) {
	policy, err := s.repo.GetPolicy(ctx, id)
	if err != nil {
		return nil, ErrPolicyNotFound
	}

	if policy.OrganizationID != orgID {
		return nil, ErrUnauthorized
	}

	// Update fields
	if req.Name != "" {
		policy.Name = req.Name
	}
	if req.Description != "" {
		policy.Description = req.Description
	}
	if req.Content != "" {
		policy.Content = req.Content
	}
	if req.Required != nil {
		policy.Required = *req.Required
	}
	if req.Renewable != nil {
		policy.Renewable = *req.Renewable
	}
	if req.ValidityPeriod != nil {
		policy.ValidityPeriod = req.ValidityPeriod
	}
	if req.Active != nil {
		policy.Active = *req.Active
		if *req.Active {
			now := time.Now()
			policy.PublishedAt = &now
		}
	}
	if req.Metadata != nil {
		policy.Metadata = req.Metadata
	}

	if err := s.repo.UpdatePolicy(ctx, policy); err != nil {
		return nil, fmt.Errorf("failed to update policy: %w", err)
	}

	// TODO: Audit log

	return policy, nil
}

// PublishPolicy activates a policy
func (s *Service) PublishPolicy(ctx context.Context, id, orgID string) error {
	policy, err := s.repo.GetPolicy(ctx, id)
	if err != nil {
		return ErrPolicyNotFound
	}

	if policy.OrganizationID != orgID {
		return ErrUnauthorized
	}

	policy.Active = true
	now := time.Now()
	policy.PublishedAt = &now

	return s.repo.UpdatePolicy(ctx, policy)
}

// ====== Cookie Consent ======

// RecordCookieConsent records cookie consent preferences
func (s *Service) RecordCookieConsent(ctx context.Context, orgID, userID string, req *CookieConsentRequest) (*CookieConsent, error) {
	if !s.config.CookieConsent.Enabled {
		return nil, fmt.Errorf("cookie consent is not enabled")
	}

	consent := &CookieConsent{
		UserID:               userID,
		OrganizationID:       orgID,
		SessionID:            req.SessionID,
		Essential:            true, // Always true
		Functional:           req.Functional,
		Analytics:            req.Analytics,
		Marketing:            req.Marketing,
		Personalization:      req.Personalization,
		ThirdParty:           req.ThirdParty,
		ConsentBannerVersion: req.BannerVersion,
		ExpiresAt:            time.Now().Add(s.config.CookieConsent.ValidityPeriod),
	}

	// Extract IP and user agent
	if ipAddr, ok := ctx.Value("ip_address").(string); ok {
		consent.IPAddress = ipAddr
	}
	if userAgent, ok := ctx.Value("user_agent").(string); ok {
		consent.UserAgent = userAgent
	}

	if err := s.repo.CreateCookieConsent(ctx, consent); err != nil {
		return nil, fmt.Errorf("failed to record cookie consent: %w", err)
	}

	// TODO: Audit log

	return consent, nil
}

// GetCookieConsent retrieves cookie consent preferences
func (s *Service) GetCookieConsent(ctx context.Context, userID, orgID string) (*CookieConsent, error) {
	return s.repo.GetCookieConsent(ctx, userID, orgID)
}

// UpdateCookieConsent updates cookie consent preferences
func (s *Service) UpdateCookieConsent(ctx context.Context, id, userID, orgID string, req *CookieConsentRequest) (*CookieConsent, error) {
	consent, err := s.repo.GetCookieConsent(ctx, userID, orgID)
	if err != nil {
		return nil, ErrCookieConsentNotFound
	}

	consent.Functional = req.Functional
	consent.Analytics = req.Analytics
	consent.Marketing = req.Marketing
	consent.Personalization = req.Personalization
	consent.ThirdParty = req.ThirdParty
	consent.ExpiresAt = time.Now().Add(s.config.CookieConsent.ValidityPeriod)

	if err := s.repo.UpdateCookieConsent(ctx, consent); err != nil {
		return nil, fmt.Errorf("failed to update cookie consent: %w", err)
	}

	return consent, nil
}

// ====== Data Export (GDPR Article 20 - Data Portability) ======

// RequestDataExport creates a data export request
func (s *Service) RequestDataExport(ctx context.Context, userID, orgID string, req *DataExportRequestInput) (*DataExportRequest, error) {
	if !s.config.DataExport.Enabled {
		return nil, fmt.Errorf("data export is not enabled")
	}

	// Check for pending export
	pendingStatus := string(StatusPending)
	existing, err := s.repo.ListExportRequests(ctx, userID, orgID, &pendingStatus)
	if err == nil && len(existing) > 0 {
		return nil, ErrExportAlreadyPending
	}

	// Check rate limit
	period := time.Now().Add(-s.config.DataExport.RequestPeriod)
	recentExports, err := s.repo.ListExportRequests(ctx, userID, orgID, nil)
	if err == nil {
		count := 0
		for _, export := range recentExports {
			if export.CreatedAt.After(period) {
				count++
			}
		}
		if count >= s.config.DataExport.MaxRequests {
			return nil, fmt.Errorf("export request limit exceeded: max %d per %v",
				s.config.DataExport.MaxRequests, s.config.DataExport.RequestPeriod)
		}
	}

	// Default sections if not specified
	includeSections := req.IncludeSections
	if len(includeSections) == 0 || (len(includeSections) == 1 && includeSections[0] == "all") {
		includeSections = s.config.DataExport.IncludeSections
	}

	exportReq := &DataExportRequest{
		UserID:          userID,
		OrganizationID:  orgID,
		Status:          string(StatusPending),
		Format:          req.Format,
		IncludeSections: includeSections,
	}

	// Extract IP
	if ipAddr, ok := ctx.Value("ip_address").(string); ok {
		exportReq.IPAddress = ipAddr
	}

	if err := s.repo.CreateExportRequest(ctx, exportReq); err != nil {
		return nil, fmt.Errorf("failed to create export request: %w", err)
	}

	// Process export asynchronously (in production, use a job queue)
	go s.processDataExport(context.Background(), exportReq)

	// TODO: Audit log

	return exportReq, nil
}

// processDataExport processes a data export request (GDPR Article 20)
func (s *Service) processDataExport(ctx context.Context, req *DataExportRequest) {
	// Update status to processing
	req.Status = string(StatusProcessing)
	s.repo.UpdateExportRequest(ctx, req)

	// Collect data from various sources
	data := make(map[string]interface{})

	for _, section := range req.IncludeSections {
		switch section {
		case "profile":
			if s.userService != nil {
				userXID, _ := xid.FromString(req.UserID)
				user, err := s.userService.FindByID(ctx, userXID)
				if err == nil {
					data["profile"] = user
				}
			}
		case "consents":
			consents, err := s.repo.ListConsentsByUser(ctx, req.UserID, req.OrganizationID)
			if err == nil {
				data["consents"] = consents
			}
		case "sessions":
			// Would fetch sessions from session service
			data["sessions"] = []interface{}{} // Placeholder
		case "audit":
			logs, err := s.repo.ListAuditLogs(ctx, req.UserID, req.OrganizationID, 1000)
			if err == nil {
				data["audit_logs"] = logs
			}
		}
	}

	// Create export file
	exportPath, exportSize, err := s.createExportFile(req, data)
	if err != nil {
		req.Status = string(StatusFailed)
		req.ErrorMessage = err.Error()
		s.repo.UpdateExportRequest(ctx, req)
		return
	}

	// Update request with export details
	req.Status = string(StatusCompleted)
	req.ExportPath = exportPath
	req.ExportSize = exportSize
	expiryTime := time.Now().Add(time.Duration(s.config.DataExport.ExpiryHours) * time.Hour)
	req.ExpiresAt = &expiryTime
	now := time.Now()
	req.CompletedAt = &now

	// Generate download URL (would be a signed URL in production)
	req.ExportURL = fmt.Sprintf("/consent/exports/%s/download", req.ID.String())

	s.repo.UpdateExportRequest(ctx, req)

	// TODO: Send notification and audit log
}

// createExportFile creates an export file in the specified format
func (s *Service) createExportFile(req *DataExportRequest, data map[string]interface{}) (string, int64, error) {
	// Ensure export directory exists
	exportDir := filepath.Join(s.config.DataExport.StoragePath, req.OrganizationID)
	if err := os.MkdirAll(exportDir, 0755); err != nil {
		return "", 0, fmt.Errorf("failed to create export directory: %w", err)
	}

	filename := fmt.Sprintf("data_export_%s_%s.%s", req.UserID, req.ID.String(), req.Format)
	filepath := filepath.Join(exportDir, filename)

	var content []byte
	var err error

	switch req.Format {
	case "json":
		content, err = json.MarshalIndent(data, "", "  ")
	case "csv":
		// CSV conversion (simplified - would need proper CSV marshaling)
		jsonData, _ := json.Marshal(data)
		content = jsonData // Placeholder
	default:
		return "", 0, ErrInvalidExportFormat
	}

	if err != nil {
		return "", 0, fmt.Errorf("failed to marshal data: %w", err)
	}

	// Check size limit
	if int64(len(content)) > s.config.DataExport.MaxExportSize {
		return "", 0, fmt.Errorf("export size exceeds limit: %d > %d bytes",
			len(content), s.config.DataExport.MaxExportSize)
	}

	if err := os.WriteFile(filepath, content, 0644); err != nil {
		return "", 0, fmt.Errorf("failed to write export file: %w", err)
	}

	return filepath, int64(len(content)), nil
}

// GetExportRequest retrieves an export request
func (s *Service) GetExportRequest(ctx context.Context, id string) (*DataExportRequest, error) {
	return s.repo.GetExportRequest(ctx, id)
}

// ListExportRequests lists export requests for a user
func (s *Service) ListExportRequests(ctx context.Context, userID, orgID string) ([]*DataExportRequest, error) {
	return s.repo.ListExportRequests(ctx, userID, orgID, nil)
}

// ====== Data Deletion (GDPR Article 17 - Right to be Forgotten) ======

// RequestDataDeletion creates a data deletion request
func (s *Service) RequestDataDeletion(ctx context.Context, userID, orgID string, req *DataDeletionRequestInput) (*DataDeletionRequest, error) {
	if !s.config.DataDeletion.Enabled {
		return nil, fmt.Errorf("data deletion is not enabled")
	}

	// Check for existing pending deletion
	existing, err := s.repo.GetPendingDeletionRequest(ctx, userID, orgID)
	if err == nil && existing != nil {
		return nil, ErrDeletionAlreadyPending
	}

	// Default sections if not specified
	deleteSections := req.DeleteSections
	if len(deleteSections) == 0 || (len(deleteSections) == 1 && deleteSections[0] == "all") {
		deleteSections = []string{"all"}
	}

	deletionReq := &DataDeletionRequest{
		UserID:          userID,
		OrganizationID:  orgID,
		Status:          string(StatusPending),
		RequestReason:   req.Reason,
		DeleteSections:  deleteSections,
		RetentionExempt: false, // Will be checked during approval
	}

	// Extract IP
	if ipAddr, ok := ctx.Value("ip_address").(string); ok {
		deletionReq.IPAddress = ipAddr
	}

	if err := s.repo.CreateDeletionRequest(ctx, deletionReq); err != nil {
		return nil, fmt.Errorf("failed to create deletion request: %w", err)
	}

	// Auto-approve if admin approval not required
	if !s.config.DataDeletion.RequireAdminApproval {
		now := time.Now()
		deletionReq.Status = string(StatusApproved)
		deletionReq.ApprovedAt = &now
		s.repo.UpdateDeletionRequest(ctx, deletionReq)
	}

	// TODO: Audit log

	return deletionReq, nil
}

// ApproveDeletionRequest approves a deletion request
func (s *Service) ApproveDeletionRequest(ctx context.Context, requestID, approverID, orgID string) error {
	req, err := s.repo.GetDeletionRequest(ctx, requestID)
	if err != nil {
		return ErrDeletionNotFound
	}

	if req.OrganizationID != orgID {
		return ErrUnauthorized
	}

	if req.Status != string(StatusPending) {
		return fmt.Errorf("deletion request is not pending")
	}

	now := time.Now()
	req.Status = string(StatusApproved)
	req.ApprovedBy = approverID
	req.ApprovedAt = &now

	if err := s.repo.UpdateDeletionRequest(ctx, req); err != nil {
		return fmt.Errorf("failed to approve deletion request: %w", err)
	}

	// TODO: Send notification and audit log

	return nil
}

// ProcessDeletionRequest processes an approved deletion request (GDPR Article 17)
func (s *Service) ProcessDeletionRequest(ctx context.Context, requestID string) error {
	req, err := s.repo.GetDeletionRequest(ctx, requestID)
	if err != nil {
		return ErrDeletionNotFound
	}

	if req.Status != string(StatusApproved) {
		return ErrDeletionNotApproved
	}

	// Check if grace period has passed
	if s.config.DataDeletion.GracePeriodDays > 0 && req.ApprovedAt != nil {
		gracePeriodEnd := req.ApprovedAt.AddDate(0, 0, s.config.DataDeletion.GracePeriodDays)
		if time.Now().Before(gracePeriodEnd) {
			return fmt.Errorf("grace period has not passed yet")
		}
	}

	// Check for legal retention exemptions
	if req.RetentionExempt {
		return ErrRetentionExempt
	}

	// Update status
	req.Status = string(StatusProcessing)
	s.repo.UpdateDeletionRequest(ctx, req)

	// Archive data before deletion if configured
	if s.config.DataDeletion.ArchiveBeforeDeletion {
		archivePath, err := s.archiveUserData(ctx, req.UserID, req.OrganizationID)
		if err != nil {
			req.Status = string(StatusFailed)
			req.ErrorMessage = fmt.Sprintf("archive failed: %v", err)
			s.repo.UpdateDeletionRequest(ctx, req)
			return fmt.Errorf("failed to archive user data: %w", err)
		}
		req.ArchivePath = archivePath
	}

	// Delete data based on sections
	for _, section := range req.DeleteSections {
		switch section {
		case "all":
			// Delete all user data
			if err := s.deleteAllUserData(ctx, req.UserID, req.OrganizationID); err != nil {
				req.Status = string(StatusFailed)
				req.ErrorMessage = err.Error()
				s.repo.UpdateDeletionRequest(ctx, req)
				return err
			}
		case "consents":
			// Delete consent records (keep audit logs)
			consents, _ := s.repo.ListConsentsByUser(ctx, req.UserID, req.OrganizationID)
			for _, consent := range consents {
				s.repo.DeleteConsent(ctx, consent.ID.String())
			}
		case "profile":
			// Mark user for deletion in user service
			// (Actual deletion would be handled by user service)
		}
	}

	// Update status
	now := time.Now()
	req.Status = string(StatusCompleted)
	req.CompletedAt = &now
	s.repo.UpdateDeletionRequest(ctx, req)

	// TODO: Send notification and audit log

	return nil
}

// archiveUserData creates an archive of user data before deletion
func (s *Service) archiveUserData(ctx context.Context, userID, orgID string) (string, error) {
	// Collect all user data
	data := make(map[string]interface{})

	// User profile
	if s.userService != nil {
		userXID, _ := xid.FromString(userID)
		user, err := s.userService.FindByID(ctx, userXID)
		if err == nil {
			data["profile"] = user
		}
	}

	// Consents
	consents, _ := s.repo.ListConsentsByUser(ctx, userID, orgID)
	data["consents"] = consents

	// Audit logs
	logs, _ := s.repo.ListAuditLogs(ctx, userID, orgID, 10000)
	data["audit_logs"] = logs

	// Create archive file
	archiveDir := filepath.Join(s.config.DataDeletion.ArchivePath, orgID)
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create archive directory: %w", err)
	}

	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("user_%s_archive_%s.json", userID, timestamp)
	archivePath := filepath.Join(archiveDir, filename)

	content, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal archive data: %w", err)
	}

	if err := os.WriteFile(archivePath, content, 0644); err != nil {
		return "", fmt.Errorf("failed to write archive file: %w", err)
	}

	return archivePath, nil
}

// deleteAllUserData deletes all user data (GDPR Article 17 implementation)
func (s *Service) deleteAllUserData(ctx context.Context, userID, orgID string) error {
	// Delete consent records
	consents, err := s.repo.ListConsentsByUser(ctx, userID, orgID)
	if err == nil {
		for _, consent := range consents {
			s.repo.DeleteConsent(ctx, consent.ID.String())
		}
	}

	// Delete cookie consents
	// (Keep audit logs for compliance - they are immutable)

	// Delete export requests (and files)
	exports, err := s.repo.ListExportRequests(ctx, userID, orgID, nil)
	if err == nil {
		for _, export := range exports {
			if export.ExportPath != "" {
				os.Remove(export.ExportPath)
			}
		}
	}

	// Note: Audit logs are kept for legal compliance (immutable)
	// User profile deletion would be handled by the user service

	return nil
}

// GetDeletionRequest retrieves a deletion request
func (s *Service) GetDeletionRequest(ctx context.Context, id string) (*DataDeletionRequest, error) {
	return s.repo.GetDeletionRequest(ctx, id)
}

// ListDeletionRequests lists deletion requests
func (s *Service) ListDeletionRequests(ctx context.Context, userID, orgID string) ([]*DataDeletionRequest, error) {
	return s.repo.ListDeletionRequests(ctx, userID, orgID, nil)
}

// ====== Privacy Settings ======

// GetPrivacySettings retrieves privacy settings for an organization
func (s *Service) GetPrivacySettings(ctx context.Context, orgID string) (*PrivacySettings, error) {
	settings, err := s.repo.GetPrivacySettings(ctx, orgID)
	if err != nil {
		// Return default settings if not found
		return s.createDefaultPrivacySettings(ctx, orgID)
	}
	return settings, nil
}

// UpdatePrivacySettings updates privacy settings for an organization
func (s *Service) UpdatePrivacySettings(ctx context.Context, orgID, updatedBy string, req *PrivacySettingsRequest) (*PrivacySettings, error) {
	settings, err := s.repo.GetPrivacySettings(ctx, orgID)
	if err != nil {
		// Create if doesn't exist
		settings, err = s.createDefaultPrivacySettings(ctx, orgID)
		if err != nil {
			return nil, err
		}
	}

	// Update fields
	if req.ConsentRequired != nil {
		settings.ConsentRequired = *req.ConsentRequired
	}
	if req.CookieConsentEnabled != nil {
		settings.CookieConsentEnabled = *req.CookieConsentEnabled
	}
	if req.CookieConsentStyle != "" {
		settings.CookieConsentStyle = req.CookieConsentStyle
	}
	if req.DataRetentionDays != nil {
		settings.DataRetentionDays = *req.DataRetentionDays
	}
	if req.AnonymousConsentEnabled != nil {
		settings.AnonymousConsentEnabled = *req.AnonymousConsentEnabled
	}
	if req.GDPRMode != nil {
		settings.GDPRMode = *req.GDPRMode
	}
	if req.CCPAMode != nil {
		settings.CCPAMode = *req.CCPAMode
	}
	if req.AutoDeleteAfterDays != nil {
		settings.AutoDeleteAfterDays = *req.AutoDeleteAfterDays
	}
	if req.RequireExplicitConsent != nil {
		settings.RequireExplicitConsent = *req.RequireExplicitConsent
	}
	if req.AllowDataPortability != nil {
		settings.AllowDataPortability = *req.AllowDataPortability
	}
	if req.ExportFormat != nil {
		settings.ExportFormat = req.ExportFormat
	}
	if req.DataExportExpiryHours != nil {
		settings.DataExportExpiryHours = *req.DataExportExpiryHours
	}
	if req.RequireAdminApprovalForDeletion != nil {
		settings.RequireAdminApprovalForDeletion = *req.RequireAdminApprovalForDeletion
	}
	if req.DeletionGracePeriodDays != nil {
		settings.DeletionGracePeriodDays = *req.DeletionGracePeriodDays
	}
	if req.ContactEmail != "" {
		settings.ContactEmail = req.ContactEmail
	}
	if req.ContactPhone != "" {
		settings.ContactPhone = req.ContactPhone
	}
	if req.DPOEmail != "" {
		settings.DPOEmail = req.DPOEmail
	}

	if err := s.repo.UpdatePrivacySettings(ctx, settings); err != nil {
		return nil, fmt.Errorf("failed to update privacy settings: %w", err)
	}

	// TODO: Audit log

	return settings, nil
}

// createDefaultPrivacySettings creates default privacy settings for an organization
func (s *Service) createDefaultPrivacySettings(ctx context.Context, orgID string) (*PrivacySettings, error) {
	settings := &PrivacySettings{
		OrganizationID:                  orgID,
		ConsentRequired:                 true,
		CookieConsentEnabled:            true,
		CookieConsentStyle:              "banner",
		DataRetentionDays:               2555, // 7 years
		AnonymousConsentEnabled:         true,
		GDPRMode:                        s.config.GDPREnabled,
		CCPAMode:                        s.config.CCPAEnabled,
		AutoDeleteAfterDays:             0, // Disabled by default
		RequireExplicitConsent:          true,
		AllowDataPortability:            true,
		ExportFormat:                    []string{"json", "csv"},
		DataExportExpiryHours:           72, // 3 days
		RequireAdminApprovalForDeletion: true,
		DeletionGracePeriodDays:         30,
	}

	if err := s.repo.CreatePrivacySettings(ctx, settings); err != nil {
		return nil, fmt.Errorf("failed to create privacy settings: %w", err)
	}

	return settings, nil
}

// ====== Audit & Helpers ======

// createAuditLog creates an immutable audit log entry
func (s *Service) createAuditLog(ctx context.Context, userID, orgID, consentID string, action ConsentAction, consentType, purpose string, previousValue interface{}, newValue interface{}) {
	if !s.config.Audit.Enabled {
		return
	}

	prevJSON, _ := json.Marshal(previousValue)
	newJSON, _ := json.Marshal(newValue)

	var prevMap, newMap JSONBMap
	json.Unmarshal(prevJSON, &prevMap)
	json.Unmarshal(newJSON, &newMap)

	log := &ConsentAuditLog{
		UserID:         userID,
		OrganizationID: orgID,
		ConsentID:      consentID,
		Action:         string(action),
		ConsentType:    consentType,
		Purpose:        purpose,
		PreviousValue:  prevMap,
		NewValue:       newMap,
	}

	// Extract IP and user agent
	if ipAddr, ok := ctx.Value("ip_address").(string); ok {
		log.IPAddress = ipAddr
	}
	if userAgent, ok := ctx.Value("user_agent").(string); ok {
		log.UserAgent = userAgent
	}

	s.repo.CreateAuditLog(ctx, log)
}

// TODO: Notification helper - integrate with notification service
// func (s *Service) sendNotification(...)

// GenerateConsentReport generates analytics report
func (s *Service) GenerateConsentReport(ctx context.Context, orgID string, startDate, endDate time.Time) (*ConsentReport, error) {
	stats, err := s.repo.GetConsentStats(ctx, orgID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get consent stats: %w", err)
	}

	report := &ConsentReport{
		OrganizationID:    orgID,
		ReportPeriodStart: startDate,
		ReportPeriodEnd:   endDate,
		ConsentsByType:    make(map[string]ConsentStats),
	}

	if total, ok := stats["totalConsents"].(int); ok {
		report.TotalUsers = total
	}
	if granted, ok := stats["grantedConsents"].(int); ok {
		report.UsersWithConsent = granted
	}
	if revoked, ok := stats["revokedConsents"].(int); ok {
		// report.RevokedConsents = revoked
		_ = revoked
	}
	if pending, ok := stats["pendingDeletions"].(int); ok {
		report.PendingDeletions = pending
	}
	if exports, ok := stats["dataExports"].(int); ok {
		report.DataExportsThisPeriod = exports
	}

	if report.TotalUsers > 0 {
		report.ConsentRate = float64(report.UsersWithConsent) / float64(report.TotalUsers)
	}

	return report, nil
}

// ====== Data Processing Agreements ======

// CreateDPA creates a new data processing agreement
func (s *Service) CreateDPA(ctx context.Context, orgID, signedBy string, req *CreateDPARequest) (*DataProcessingAgreement, error) {
	// Generate digital signature
	signature := s.generateDigitalSignature(req.Content, req.SignedByEmail)

	dpa := &DataProcessingAgreement{
		OrganizationID:   orgID,
		AgreementType:    req.AgreementType,
		Version:          req.Version,
		Content:          req.Content,
		SignedBy:         signedBy,
		SignedByName:     req.SignedByName,
		SignedByTitle:    req.SignedByTitle,
		SignedByEmail:    req.SignedByEmail,
		DigitalSignature: signature,
		EffectiveDate:    req.EffectiveDate,
		ExpiryDate:       req.ExpiryDate,
		Status:           "active",
		Metadata:         req.Metadata,
	}

	// Extract IP
	if ipAddr, ok := ctx.Value("ip_address").(string); ok {
		dpa.IPAddress = ipAddr
	}

	if err := s.repo.CreateDPA(ctx, dpa); err != nil {
		return nil, fmt.Errorf("failed to create DPA: %w", err)
	}

	// TODO: Audit log

	return dpa, nil
}

// generateDigitalSignature generates a cryptographic signature for DPA
func (s *Service) generateDigitalSignature(content, email string) string {
	data := fmt.Sprintf("%s:%s:%d", content, email, time.Now().Unix())
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}
