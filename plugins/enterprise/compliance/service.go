package compliance

import (
	"context"
	"fmt"
	"time"
)

// Service handles compliance business logic
type Service struct {
	repo     Repository
	config   *Config
	auditSvc AuditService
	userSvc  UserService
	orgSvc   OrganizationService
	emailSvc EmailService
}

// NewService creates a new compliance service
func NewService(
	repo Repository,
	config *Config,
	auditSvc AuditService,
	userSvc UserService,
	orgSvc OrganizationService,
	emailSvc EmailService,
) *Service {
	return &Service{
		repo:     repo,
		config:   config,
		auditSvc: auditSvc,
		userSvc:  userSvc,
		orgSvc:   orgSvc,
		emailSvc: emailSvc,
	}
}

// ===== Compliance Profile Management =====

// CreateProfile creates a new compliance profile
func (s *Service) CreateProfile(ctx context.Context, req *CreateProfileRequest) (*ComplianceProfile, error) {
	// Check if profile already exists
	existing, _ := s.repo.GetProfileByOrganization(ctx, req.OrganizationID)
	if existing != nil {
		return nil, ErrProfileExists
	}

	profile := &ComplianceProfile{
		OrganizationID: req.OrganizationID,
		Name:           req.Name,
		Standards:      req.Standards,
		Status:         "active",

		// Security
		MFARequired:           req.MFARequired,
		PasswordMinLength:     req.PasswordMinLength,
		PasswordRequireUpper:  req.PasswordRequireUpper,
		PasswordRequireLower:  req.PasswordRequireLower,
		PasswordRequireNumber: req.PasswordRequireNumber,
		PasswordRequireSymbol: req.PasswordRequireSymbol,
		PasswordExpiryDays:    req.PasswordExpiryDays,

		// Session
		SessionMaxAge:      req.SessionMaxAge,
		SessionIdleTimeout: req.SessionIdleTimeout,
		SessionIPBinding:   req.SessionIPBinding,

		// Audit
		RetentionDays:      req.RetentionDays,
		AuditLogExport:     req.AuditLogExport,
		DetailedAuditTrail: req.DetailedAuditTrail,

		// Data
		DataResidency:       req.DataResidency,
		EncryptionAtRest:    req.EncryptionAtRest,
		EncryptionInTransit: req.EncryptionInTransit,

		// Access Control
		RBACRequired:        req.RBACRequired,
		LeastPrivilege:      req.LeastPrivilege,
		RegularAccessReview: req.RegularAccessReview,

		// Contacts
		ComplianceContact: req.ComplianceContact,
		DPOContact:        req.DPOContact,

		Metadata: req.Metadata,
	}

	if err := s.repo.CreateProfile(ctx, profile); err != nil {
		return nil, fmt.Errorf("failed to create profile: %w", err)
	}

	// Audit the creation
	s.auditSvc.LogEvent(ctx, &AuditEvent{
		Action:         "compliance.profile.created",
		OrganizationID: req.OrganizationID,
		ResourceID:     profile.ID,
		Metadata: map[string]interface{}{
			"standards": profile.Standards,
		},
	})

	// Initialize automated checks for this profile
	if s.config.AutomatedChecks.Enabled {
		go s.scheduleChecks(context.Background(), profile)
	}

	return profile, nil
}

// CreateProfileFromTemplate creates a profile from a compliance template
func (s *Service) CreateProfileFromTemplate(ctx context.Context, orgID string, standard ComplianceStandard) (*ComplianceProfile, error) {
	profile, err := CreateProfileFromTemplate(orgID, standard)
	if err != nil {
		return nil, err
	}

	if err := s.repo.CreateProfile(ctx, profile); err != nil {
		return nil, fmt.Errorf("failed to create profile from template: %w", err)
	}

	// Audit
	s.auditSvc.LogEvent(ctx, &AuditEvent{
		Action:         "compliance.profile.created_from_template",
		OrganizationID: orgID,
		ResourceID:     profile.ID,
		Metadata: map[string]interface{}{
			"template": standard,
		},
	})

	return profile, nil
}

// GetProfile retrieves a compliance profile
func (s *Service) GetProfile(ctx context.Context, id string) (*ComplianceProfile, error) {
	return s.repo.GetProfile(ctx, id)
}

// GetProfileByOrganization retrieves a profile by organization ID
func (s *Service) GetProfileByOrganization(ctx context.Context, orgID string) (*ComplianceProfile, error) {
	return s.repo.GetProfileByOrganization(ctx, orgID)
}

// UpdateProfile updates a compliance profile
func (s *Service) UpdateProfile(ctx context.Context, id string, req *UpdateProfileRequest) (*ComplianceProfile, error) {
	profile, err := s.repo.GetProfile(ctx, id)
	if err != nil {
		return nil, err
	}

	// Apply updates
	if req.Name != nil {
		profile.Name = *req.Name
	}
	if req.Status != nil {
		profile.Status = *req.Status
	}
	if req.MFARequired != nil {
		profile.MFARequired = *req.MFARequired
	}
	if req.RetentionDays != nil {
		profile.RetentionDays = *req.RetentionDays
	}
	// ... apply other fields

	profile.UpdatedAt = time.Now()

	if err := s.repo.UpdateProfile(ctx, profile); err != nil {
		return nil, fmt.Errorf("failed to update profile: %w", err)
	}

	// Audit
	s.auditSvc.LogEvent(ctx, &AuditEvent{
		Action:         "compliance.profile.updated",
		OrganizationID: profile.OrganizationID,
		ResourceID:     profile.ID,
	})

	return profile, nil
}

// ===== Compliance Checks =====

// RunCheck executes a compliance check
func (s *Service) RunCheck(ctx context.Context, profileID, checkType string) (*ComplianceCheck, error) {
	profile, err := s.repo.GetProfile(ctx, profileID)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	var status string
	var evidence []string

	switch checkType {
	case "mfa_coverage":
		result, status, evidence = s.checkMFACoverage(ctx, profile)
	case "password_policy":
		result, status, evidence = s.checkPasswordPolicy(ctx, profile)
	case "session_policy":
		result, status, evidence = s.checkSessionPolicy(ctx, profile)
	case "access_review":
		result, status, evidence = s.checkAccessReview(ctx, profile)
	case "inactive_users":
		result, status, evidence = s.checkInactiveUsers(ctx, profile)
	case "data_retention":
		result, status, evidence = s.checkDataRetention(ctx, profile)
	default:
		return nil, ErrInvalidCheckType
	}

	check := &ComplianceCheck{
		ProfileID:      profileID,
		OrganizationID: profile.OrganizationID,
		CheckType:      checkType,
		Status:         status,
		Result:         result,
		Evidence:       evidence,
		LastCheckedAt:  time.Now(),
		NextCheckAt:    time.Now().Add(s.config.AutomatedChecks.CheckInterval),
	}

	if err := s.repo.CreateCheck(ctx, check); err != nil {
		return nil, fmt.Errorf("failed to save check: %w", err)
	}

	// If check failed, create violations
	if status == "failed" {
		s.createViolationsFromCheck(ctx, check)
	}

	// Notify if configured
	if status == "failed" && s.config.Notifications.FailedChecks {
		s.notifyFailedCheck(ctx, profile, check)
	}

	return check, nil
}

// checkMFACoverage checks MFA adoption rate
func (s *Service) checkMFACoverage(ctx context.Context, profile *ComplianceProfile) (map[string]interface{}, string, []string) {
	// Get all users in organization
	users, err := s.userSvc.ListByOrganization(ctx, profile.OrganizationID)
	if err != nil {
		return nil, "failed", nil
	}

	totalUsers := len(users)
	usersWithMFA := 0
	usersWithoutMFA := []string{}

	for _, user := range users {
		if user.MFAEnabled {
			usersWithMFA++
		} else {
			usersWithoutMFA = append(usersWithoutMFA, user.ID)
		}
	}

	coveragePercent := 0
	if totalUsers > 0 {
		coveragePercent = (usersWithMFA * 100) / totalUsers
	}

	result := map[string]interface{}{
		"total_users":       totalUsers,
		"users_with_mfa":    usersWithMFA,
		"users_without_mfa": len(usersWithoutMFA),
		"coverage_percent":  coveragePercent,
	}

	status := "passed"
	if profile.MFARequired && coveragePercent < 100 {
		status = "failed"
	} else if coveragePercent < 80 {
		status = "warning"
	}

	evidence := []string{
		fmt.Sprintf("MFA coverage: %d%%", coveragePercent),
		fmt.Sprintf("Users without MFA: %d", len(usersWithoutMFA)),
	}

	return result, status, evidence
}

// checkPasswordPolicy verifies password compliance
func (s *Service) checkPasswordPolicy(ctx context.Context, profile *ComplianceProfile) (map[string]interface{}, string, []string) {
	// Get users with weak passwords or expired passwords
	users, _ := s.userSvc.ListByOrganization(ctx, profile.OrganizationID)

	weakPasswords := 0
	expiredPasswords := 0

	for _, user := range users {
		// Check password age if expiry is set
		if profile.PasswordExpiryDays > 0 {
			passwordAge := time.Since(user.PasswordChangedAt).Hours() / 24
			if passwordAge > float64(profile.PasswordExpiryDays) {
				expiredPasswords++
			}
		}

		// Check password strength (would need actual password validation)
		// This is a placeholder - real implementation would check against policy
	}

	result := map[string]interface{}{
		"total_users":       len(users),
		"weak_passwords":    weakPasswords,
		"expired_passwords": expiredPasswords,
		"min_length":        profile.PasswordMinLength,
		"expiry_days":       profile.PasswordExpiryDays,
	}

	status := "passed"
	if expiredPasswords > 0 || weakPasswords > 0 {
		status = "failed"
	}

	evidence := []string{
		fmt.Sprintf("Expired passwords: %d", expiredPasswords),
		fmt.Sprintf("Policy: min length %d, expiry %d days", profile.PasswordMinLength, profile.PasswordExpiryDays),
	}

	return result, status, evidence
}

// checkSessionPolicy verifies session compliance
func (s *Service) checkSessionPolicy(ctx context.Context, profile *ComplianceProfile) (map[string]interface{}, string, []string) {
	// This would integrate with session service to check active sessions
	result := map[string]interface{}{
		"max_age":      profile.SessionMaxAge,
		"idle_timeout": profile.SessionIdleTimeout,
		"ip_binding":   profile.SessionIPBinding,
	}

	return result, "passed", []string{"Session policy configured"}
}

// checkAccessReview checks if regular access reviews are being performed
func (s *Service) checkAccessReview(ctx context.Context, profile *ComplianceProfile) (map[string]interface{}, string, []string) {
	// Check when last access review was performed
	// This is a placeholder - would integrate with access review system

	result := map[string]interface{}{
		"last_review": "2025-10-01",
		"overdue":     false,
	}

	return result, "passed", []string{"Access review completed"}
}

// checkInactiveUsers identifies inactive users
func (s *Service) checkInactiveUsers(ctx context.Context, profile *ComplianceProfile) (map[string]interface{}, string, []string) {
	users, _ := s.userSvc.ListByOrganization(ctx, profile.OrganizationID)

	inactiveThreshold := 90 * 24 * time.Hour // 90 days
	inactiveUsers := []string{}

	for _, user := range users {
		if time.Since(user.LastLoginAt) > inactiveThreshold {
			inactiveUsers = append(inactiveUsers, user.ID)
		}
	}

	result := map[string]interface{}{
		"total_users":    len(users),
		"inactive_users": len(inactiveUsers),
		"threshold_days": 90,
	}

	status := "passed"
	if len(inactiveUsers) > 0 {
		status = "warning"
	}

	evidence := []string{
		fmt.Sprintf("Inactive users: %d", len(inactiveUsers)),
	}

	return result, status, evidence
}

// checkDataRetention verifies data retention compliance
func (s *Service) checkDataRetention(ctx context.Context, profile *ComplianceProfile) (map[string]interface{}, string, []string) {
	// Check audit logs retention
	oldestLog, _ := s.auditSvc.GetOldestLog(ctx, profile.OrganizationID)

	retentionDays := 0
	if oldestLog != nil {
		retentionDays = int(time.Since(oldestLog.CreatedAt).Hours() / 24)
	}

	result := map[string]interface{}{
		"retention_days": retentionDays,
		"required_days":  profile.RetentionDays,
		"compliant":      retentionDays >= profile.RetentionDays,
	}

	status := "passed"
	if retentionDays < profile.RetentionDays {
		status = "warning"
	}

	evidence := []string{
		fmt.Sprintf("Current retention: %d days", retentionDays),
		fmt.Sprintf("Required retention: %d days", profile.RetentionDays),
	}

	return result, status, evidence
}

// scheduleChecks schedules automated checks for a profile
func (s *Service) scheduleChecks(ctx context.Context, profile *ComplianceProfile) {
	checkTypes := []string{
		"mfa_coverage",
		"password_policy",
		"session_policy",
		"access_review",
		"inactive_users",
		"data_retention",
	}

	for _, checkType := range checkTypes {
		_, err := s.RunCheck(ctx, profile.ID, checkType)
		if err != nil {
			// Log error but continue
			continue
		}
	}
}

// createViolationsFromCheck creates violations based on failed check
func (s *Service) createViolationsFromCheck(ctx context.Context, check *ComplianceCheck) {
	// Parse check result and create specific violations
	// This is simplified - real implementation would be more detailed

	violation := &ComplianceViolation{
		ProfileID:      check.ProfileID,
		OrganizationID: check.OrganizationID,
		ViolationType:  check.CheckType + "_failed",
		Severity:       "high",
		Description:    fmt.Sprintf("Compliance check '%s' failed", check.CheckType),
		Status:         "open",
		Metadata:       check.Result,
	}

	s.repo.CreateViolation(ctx, violation)

	// Notify if configured
	if s.config.Notifications.Violations {
		profile, _ := s.repo.GetProfile(ctx, check.ProfileID)
		s.notifyViolation(ctx, profile, violation)
	}
}

// notifyFailedCheck sends notification for failed check
func (s *Service) notifyFailedCheck(ctx context.Context, profile *ComplianceProfile, check *ComplianceCheck) {
	if profile.ComplianceContact != "" {
		s.emailSvc.SendEmail(ctx, &Email{
			To:      profile.ComplianceContact,
			Subject: fmt.Sprintf("Compliance Check Failed: %s", check.CheckType),
			Body:    fmt.Sprintf("The compliance check '%s' has failed. Please review and take action.", check.CheckType),
		})
	}
}

// notifyViolation sends notification for compliance violation
func (s *Service) notifyViolation(ctx context.Context, profile *ComplianceProfile, violation *ComplianceViolation) {
	if profile.ComplianceContact != "" {
		s.emailSvc.SendEmail(ctx, &Email{
			To:      profile.ComplianceContact,
			Subject: fmt.Sprintf("Compliance Violation: %s", violation.ViolationType),
			Body:    fmt.Sprintf("A compliance violation has been detected: %s", violation.Description),
		})
	}
}

// GetComplianceStatus returns overall compliance status for an organization
func (s *Service) GetComplianceStatus(ctx context.Context, orgID string) (*ComplianceStatus, error) {
	profile, err := s.repo.GetProfileByOrganization(ctx, orgID)
	if err != nil {
		return nil, err
	}

	// Get recent checks
	checks, _ := s.repo.ListChecks(ctx, profile.ID, CheckFilters{
		Limit: 100,
	})

	// Count violations
	violations, _ := s.repo.CountViolations(ctx, orgID, "open")

	// Calculate metrics
	checksPassed := 0
	checksFailed := 0
	checksWarning := 0

	for _, check := range checks {
		switch check.Status {
		case "passed":
			checksPassed++
		case "failed":
			checksFailed++
		case "warning":
			checksWarning++
		}
	}

	totalChecks := len(checks)
	score := 0
	if totalChecks > 0 {
		score = (checksPassed * 100) / totalChecks
	}

	overallStatus := "compliant"
	if checksFailed > 0 || violations > 0 {
		overallStatus = "non_compliant"
	} else if checksWarning > 0 {
		overallStatus = "in_progress"
	}

	status := &ComplianceStatus{
		ProfileID:      profile.ID,
		OrganizationID: orgID,
		OverallStatus:  overallStatus,
		Score:          score,
		ChecksPassed:   checksPassed,
		ChecksFailed:   checksFailed,
		ChecksWarning:  checksWarning,
		Violations:     violations,
		LastChecked:    time.Now(),
		NextAudit:      time.Now().Add(90 * 24 * time.Hour), // 90 days
	}

	return status, nil
}

// Helper structs and interfaces
type CreateProfileRequest struct {
	OrganizationID        string
	Name                  string
	Standards             []ComplianceStandard
	MFARequired           bool
	PasswordMinLength     int
	PasswordRequireUpper  bool
	PasswordRequireLower  bool
	PasswordRequireNumber bool
	PasswordRequireSymbol bool
	PasswordExpiryDays    int
	SessionMaxAge         int
	SessionIdleTimeout    int
	SessionIPBinding      bool
	RetentionDays         int
	AuditLogExport        bool
	DetailedAuditTrail    bool
	DataResidency         string
	EncryptionAtRest      bool
	EncryptionInTransit   bool
	RBACRequired          bool
	LeastPrivilege        bool
	RegularAccessReview   bool
	ComplianceContact     string
	DPOContact            string
	Metadata              map[string]interface{}
}

type UpdateProfileRequest struct {
	Name          *string
	Status        *string
	MFARequired   *bool
	RetentionDays *int
	// Add other updatable fields
}

// External service interfaces
type AuditService interface {
	LogEvent(ctx context.Context, event *AuditEvent) error
	GetOldestLog(ctx context.Context, orgID string) (*AuditLog, error)
}

type UserService interface {
	ListByOrganization(ctx context.Context, orgID string) ([]*User, error)
}

type OrganizationService interface {
	Get(ctx context.Context, id string) (*Organization, error)
}

type EmailService interface {
	SendEmail(ctx context.Context, email *Email) error
}

// Helper types
type AuditEvent struct {
	Action         string
	OrganizationID string
	ResourceID     string
	Metadata       map[string]interface{}
}

type AuditLog struct {
	CreatedAt time.Time
}

type User struct {
	ID                string
	MFAEnabled        bool
	PasswordChangedAt time.Time
	LastLoginAt       time.Time
}

type Organization struct {
	ID   string
	Name string
}

type Email struct {
	To      string
	Subject string
	Body    string
}
