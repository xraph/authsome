package compliance

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"time"
	"unicode"
)

// PolicyEngine enforces compliance policies at runtime.
type PolicyEngine struct {
	service *Service
}

// NewPolicyEngine creates a new policy engine.
func NewPolicyEngine(service *Service) *PolicyEngine {
	return &PolicyEngine{
		service: service,
	}
}

// EnforcePasswordPolicy validates password against compliance requirements.
func (e *PolicyEngine) EnforcePasswordPolicy(ctx context.Context, appID, password string) error {
	profile, err := e.service.GetProfileByApp(ctx, appID)
	if err != nil {
		// No profile means no enforcement
		return nil
	}

	// Check minimum length
	if len(password) < profile.PasswordMinLength {
		return WeakPassword(fmt.Sprintf("password must be at least %d characters", profile.PasswordMinLength))
	}

	// Check uppercase requirement
	if profile.PasswordRequireUpper && !containsUppercase(password) {
		return WeakPassword("password must contain at least one uppercase letter")
	}

	// Check lowercase requirement
	if profile.PasswordRequireLower && !containsLowercase(password) {
		return WeakPassword("password must contain at least one lowercase letter")
	}

	// Check number requirement
	if profile.PasswordRequireNumber && !containsNumber(password) {
		return WeakPassword("password must contain at least one number")
	}

	// Check symbol requirement
	if profile.PasswordRequireSymbol && !containsSymbol(password) {
		return WeakPassword("password must contain at least one special character")
	}

	// Check against common weak passwords
	if isCommonPassword(password) {
		return WeakPassword("password is too common")
	}

	return nil
}

// EnforceMFA checks if MFA is required and enabled.
func (e *PolicyEngine) EnforceMFA(ctx context.Context, appID, userID string, mfaEnabled bool) error {
	profile, err := e.service.GetProfileByApp(ctx, appID)
	if err != nil {
		return nil
	}

	if profile.MFARequired && !mfaEnabled {
		// Create violation
		violation := &ComplianceViolation{
			ProfileID:     profile.ID,
			AppID:         appID,
			UserID:        userID,
			ViolationType: "mfa_not_enabled",
			Severity:      "high",
			Description:   "User does not have MFA enabled despite app requirement",
			Status:        "open",
		}
		e.service.repo.CreateViolation(ctx, violation)

		return MFARequired()
	}

	return nil
}

// EnforceSessionPolicy validates session against compliance requirements.
func (e *PolicyEngine) EnforceSessionPolicy(ctx context.Context, appID string, session *Session) error {
	profile, err := e.service.GetProfileByApp(ctx, appID)
	if err != nil {
		return nil
	}

	// Check session age
	sessionAge := time.Since(session.CreatedAt)
	maxAge := time.Duration(profile.SessionMaxAge) * time.Second

	if sessionAge > maxAge {
		return SessionExpired("session has exceeded maximum age")
	}

	// Check idle timeout
	idleTime := time.Since(session.LastActivityAt)
	idleTimeout := time.Duration(profile.SessionIdleTimeout) * time.Second

	if idleTime > idleTimeout {
		return SessionExpired("session has been idle for too long")
	}

	// Check IP binding
	if profile.SessionIPBinding && session.CreatedIP != session.CurrentIP {
		// Create violation
		violation := &ComplianceViolation{
			ProfileID:     profile.ID,
			AppID:         appID,
			UserID:        session.UserID,
			ViolationType: "session_ip_mismatch",
			Severity:      "critical",
			Description:   fmt.Sprintf("Session IP changed from %s to %s", session.CreatedIP, session.CurrentIP),
			Status:        "open",
			Metadata: map[string]any{
				"session_id": session.ID,
				"created_ip": session.CreatedIP,
				"current_ip": session.CurrentIP,
			},
		}
		e.service.repo.CreateViolation(ctx, violation)

		return AccessDenied("session IP mismatch detected")
	}

	return nil
}

// EnforceAccessControl checks if user has proper access.
func (e *PolicyEngine) EnforceAccessControl(ctx context.Context, appID, userID string, resource string, action string) error {
	profile, err := e.service.GetProfileByApp(ctx, appID)
	if err != nil {
		return nil
	}

	// If RBAC is required, verify proper role assignment
	if profile.RBACRequired {
		// This would integrate with RBAC plugin
		// Placeholder for now
	}

	// If least privilege is enforced, check for over-permissions
	if profile.LeastPrivilege {
		// This would check if user has more permissions than needed
	}

	return nil
}

// EnforceTraining checks if user has completed required training.
func (e *PolicyEngine) EnforceTraining(ctx context.Context, appID, userID string) error {
	profile, err := e.service.GetProfileByApp(ctx, appID)
	if err != nil {
		return nil
	}

	// Check if there are required training for this profile's standards
	requiredTraining := e.getRequiredTraining(profile.Standards)
	if len(requiredTraining) == 0 {
		return nil
	}

	// Get user's training status
	userTraining, _ := e.service.repo.GetUserTrainingStatus(ctx, userID)

	// Check if all required training is completed
	completedTraining := make(map[string]bool)

	for _, training := range userTraining {
		if training.Status == "completed" {
			// Check if not expired
			if training.ExpiresAt == nil || training.ExpiresAt.After(time.Now()) {
				completedTraining[training.TrainingType] = true
			}
		}
	}

	// Find missing or expired training
	var missingTraining []string

	for _, required := range requiredTraining {
		if !completedTraining[required] {
			missingTraining = append(missingTraining, required)
		}
	}

	if len(missingTraining) > 0 {
		// Create violation
		violation := &ComplianceViolation{
			ProfileID:     profile.ID,
			AppID:         appID,
			UserID:        userID,
			ViolationType: "training_incomplete",
			Severity:      "medium",
			Description:   fmt.Sprintf("User has not completed required training: %v", missingTraining),
			Status:        "open",
			Metadata: map[string]any{
				"missing_training": missingTraining,
			},
		}
		e.service.repo.CreateViolation(ctx, violation)

		return TrainingRequired(fmt.Sprintf("%v", missingTraining))
	}

	return nil
}

// EnforceDataResidency checks if data access complies with residency requirements.
func (e *PolicyEngine) EnforceDataResidency(ctx context.Context, appID, region string) error {
	profile, err := e.service.GetProfileByApp(ctx, appID)
	if err != nil {
		return nil
	}

	// If data residency is specified, verify it matches
	if profile.DataResidency != "" && profile.DataResidency != region {
		return AccessDenied(fmt.Sprintf("data access from region %s not allowed (required: %s)", region, profile.DataResidency))
	}

	return nil
}

// CheckPasswordExpiry checks if user's password has expired.
func (e *PolicyEngine) CheckPasswordExpiry(ctx context.Context, appID string, passwordChangedAt time.Time) (bool, error) {
	profile, err := e.service.GetProfileByApp(ctx, appID)
	if err != nil {
		return false, nil
	}

	// If password expiry is set
	if profile.PasswordExpiryDays > 0 {
		expiryDuration := time.Duration(profile.PasswordExpiryDays) * 24 * time.Hour
		expiryTime := passwordChangedAt.Add(expiryDuration)

		if time.Now().After(expiryTime) {
			return true, nil
		}
	}

	return false, nil
}

// getRequiredTraining returns required training for given standards.
func (e *PolicyEngine) getRequiredTraining(standards []ComplianceStandard) []string {
	trainingSet := make(map[string]bool)

	for _, standard := range standards {
		template, ok := GetTemplate(standard)
		if ok {
			for _, training := range template.RequiredTraining {
				trainingSet[training] = true
			}
		}
	}

	training := make([]string, 0, len(trainingSet))
	for t := range trainingSet {
		training = append(training, t)
	}

	return training
}

// Helper functions

func containsUppercase(s string) bool {
	for _, r := range s {
		if unicode.IsUpper(r) {
			return true
		}
	}

	return false
}

func containsLowercase(s string) bool {
	for _, r := range s {
		if unicode.IsLower(r) {
			return true
		}
	}

	return false
}

func containsNumber(s string) bool {
	for _, r := range s {
		if unicode.IsDigit(r) {
			return true
		}
	}

	return false
}

func containsSymbol(s string) bool {
	symbols := regexp.MustCompile(`[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]`)

	return symbols.MatchString(s)
}

func isCommonPassword(password string) bool {
	// List of common weak passwords
	commonPasswords := []string{
		"password", "password123", "123456", "12345678", "qwerty",
		"abc123", "monkey", "1234567", "letmein", "trustno1",
		"dragon", "baseball", "iloveyou", "master", "sunshine",
		"ashley", "bailey", "passw0rd", "shadow", "123123",
		"654321", "superman", "qazwsx", "michael", "football",
	}

	lowerPassword := []byte(password)
	for i, b := range lowerPassword {
		lowerPassword[i] = byte(unicode.ToLower(rune(b)))
	}

	lowerPasswordStr := string(lowerPassword)

	return slices.Contains(commonPasswords, lowerPasswordStr)
}

// Session represents a user session.
type Session struct {
	ID             string
	UserID         string
	CreatedAt      time.Time
	LastActivityAt time.Time
	CreatedIP      string
	CurrentIP      string
}
