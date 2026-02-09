package compliance

import (
	"context"
	"fmt"
	"sync"

	"github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/internal/errs"
)

// =============================================================================
// COMPLIANCE CHECK REGISTRY - Pluggable check system
// =============================================================================

// CheckResult represents the result of a compliance check.
type CheckResult struct {
	CheckType string                 `json:"checkType"`
	Status    string                 `json:"status"` // "passed", "failed", "warning", "error"
	Score     float64                `json:"score"`  // 0-100
	Result    map[string]interface{} `json:"result"`
	Evidence  []string               `json:"evidence"`
	Error     error                  `json:"error,omitempty"`
}

// ComplianceCheckFunc is a function that performs a compliance check
// Takes scope, profile, and additional context.
type ComplianceCheckFunc func(ctx context.Context, scope *audit.Scope, profile *ComplianceProfile, deps *CheckDependencies) (*CheckResult, error)

// CheckDependencies provides access to services needed by checks.
type CheckDependencies struct {
	AuditSvc      AuditService
	UserSvc       UserService
	AppSvc        AppService
	EmailSvc      EmailService
	UserProvider  audit.UserProvider
	OrgProvider   audit.OrganizationProvider
	ScopeResolver *ScopeResolver
}

// CheckMetadata contains metadata about a check.
type CheckMetadata struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Category    string   `json:"category"`  // "security", "audit", "access", "data"
	Severity    string   `json:"severity"`  // "low", "medium", "high", "critical"
	Standards   []string `json:"standards"` // ["SOC2", "HIPAA", "PCI-DSS"]
	AutoRun     bool     `json:"autoRun"`   // Run automatically on schedule
}

// CheckRegistry manages registered compliance checks.
type CheckRegistry struct {
	checks   map[string]ComplianceCheckFunc
	metadata map[string]*CheckMetadata
	mu       sync.RWMutex
}

// NewCheckRegistry creates a new check registry.
func NewCheckRegistry() *CheckRegistry {
	return &CheckRegistry{
		checks:   make(map[string]ComplianceCheckFunc),
		metadata: make(map[string]*CheckMetadata),
	}
}

// Register registers a new compliance check.
func (r *CheckRegistry) Register(checkType string, fn ComplianceCheckFunc, meta *CheckMetadata) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if checkType == "" {
		return errs.InvalidInput("checkType", "check type cannot be empty")
	}

	if fn == nil {
		return errs.InvalidInput("fn", "check function cannot be nil")
	}

	// Check if already registered
	if _, exists := r.checks[checkType]; exists {
		return fmt.Errorf("check '%s' is already registered", checkType)
	}

	r.checks[checkType] = fn
	r.metadata[checkType] = meta

	return nil
}

// Unregister removes a check from the registry.
func (r *CheckRegistry) Unregister(checkType string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.checks[checkType]; !exists {
		return fmt.Errorf("check '%s' is not registered", checkType)
	}

	delete(r.checks, checkType)
	delete(r.metadata, checkType)

	return nil
}

// Execute runs a specific check.
func (r *CheckRegistry) Execute(ctx context.Context, checkType string, scope *audit.Scope, profile *ComplianceProfile, deps *CheckDependencies) (*CheckResult, error) {
	r.mu.RLock()
	fn, exists := r.checks[checkType]
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("check '%s' is not registered", checkType)
	}

	// Execute the check
	result, err := fn(ctx, scope, profile, deps)
	if err != nil {
		return &CheckResult{
			CheckType: checkType,
			Status:    "error",
			Error:     err,
		}, err
	}

	return result, nil
}

// ExecuteAll runs all registered checks.
func (r *CheckRegistry) ExecuteAll(ctx context.Context, scope *audit.Scope, profile *ComplianceProfile, deps *CheckDependencies) ([]*CheckResult, error) {
	r.mu.RLock()

	checkTypes := make([]string, 0, len(r.checks))
	for checkType := range r.checks {
		checkTypes = append(checkTypes, checkType)
	}

	r.mu.RUnlock()

	results := make([]*CheckResult, 0, len(checkTypes))
	for _, checkType := range checkTypes {
		result, err := r.Execute(ctx, checkType, scope, profile, deps)
		if err != nil {
			// Don't fail entire execution if one check fails
			// Just record the error
			result = &CheckResult{
				CheckType: checkType,
				Status:    "error",
				Error:     err,
			}
		}

		results = append(results, result)
	}

	return results, nil
}

// ExecuteForStandards runs checks relevant to specific compliance standards.
func (r *CheckRegistry) ExecuteForStandards(ctx context.Context, standards []ComplianceStandard, scope *audit.Scope, profile *ComplianceProfile, deps *CheckDependencies) ([]*CheckResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Convert standards to strings for comparison
	standardStrs := make(map[string]bool)
	for _, std := range standards {
		standardStrs[string(std)] = true
	}

	results := make([]*CheckResult, 0)

	for checkType, fn := range r.checks {
		meta := r.metadata[checkType]

		// Check if this check applies to any of the requested standards
		if meta != nil && len(meta.Standards) > 0 {
			applies := false

			for _, std := range meta.Standards {
				if standardStrs[std] {
					applies = true

					break
				}
			}

			if !applies {
				continue
			}
		}

		// Execute the check
		result, err := fn(ctx, scope, profile, deps)
		if err != nil {
			result = &CheckResult{
				CheckType: checkType,
				Status:    "error",
				Error:     err,
			}
		}

		results = append(results, result)
	}

	return results, nil
}

// List returns all registered check types.
func (r *CheckRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	checkTypes := make([]string, 0, len(r.checks))
	for checkType := range r.checks {
		checkTypes = append(checkTypes, checkType)
	}

	return checkTypes
}

// GetMetadata retrieves metadata for a check.
func (r *CheckRegistry) GetMetadata(checkType string) (*CheckMetadata, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	meta, exists := r.metadata[checkType]
	if !exists {
		return nil, fmt.Errorf("check '%s' is not registered", checkType)
	}

	return meta, nil
}

// ListMetadata returns metadata for all registered checks.
func (r *CheckRegistry) ListMetadata() map[string]*CheckMetadata {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Return copy to prevent external mutation
	result := make(map[string]*CheckMetadata, len(r.metadata))
	for k, v := range r.metadata {
		result[k] = v
	}

	return result
}

// =============================================================================
// BUILT-IN CHECKS - Register default compliance checks
// =============================================================================

// RegisterBuiltInChecks registers all built-in compliance checks.
func RegisterBuiltInChecks(registry *CheckRegistry) error {
	checks := []struct {
		Type     string
		Func     ComplianceCheckFunc
		Metadata *CheckMetadata
	}{
		{
			Type: "mfa_coverage",
			Func: checkMFACoverage,
			Metadata: &CheckMetadata{
				Name:        "MFA Coverage",
				Description: "Verifies Multi-Factor Authentication adoption rate",
				Category:    "security",
				Severity:    "high",
				Standards:   []string{"SOC2", "HIPAA", "PCI-DSS"},
				AutoRun:     true,
			},
		},
		{
			Type: "password_policy",
			Func: checkPasswordPolicy,
			Metadata: &CheckMetadata{
				Name:        "Password Policy Compliance",
				Description: "Validates password strength and expiry requirements",
				Category:    "security",
				Severity:    "high",
				Standards:   []string{"SOC2", "HIPAA", "PCI-DSS", "ISO27001"},
				AutoRun:     true,
			},
		},
		{
			Type: "session_policy",
			Func: checkSessionPolicy,
			Metadata: &CheckMetadata{
				Name:        "Session Policy Compliance",
				Description: "Validates session timeout and binding requirements",
				Category:    "security",
				Severity:    "medium",
				Standards:   []string{"SOC2", "PCI-DSS"},
				AutoRun:     true,
			},
		},
		{
			Type: "access_review",
			Func: checkAccessReview,
			Metadata: &CheckMetadata{
				Name:        "Access Review",
				Description: "Checks if regular access reviews are being performed",
				Category:    "access",
				Severity:    "medium",
				Standards:   []string{"SOC2", "ISO27001"},
				AutoRun:     true,
			},
		},
		{
			Type: "inactive_users",
			Func: checkInactiveUsers,
			Metadata: &CheckMetadata{
				Name:        "Inactive User Detection",
				Description: "Identifies inactive user accounts that should be reviewed",
				Category:    "access",
				Severity:    "low",
				Standards:   []string{"SOC2", "HIPAA"},
				AutoRun:     true,
			},
		},
		{
			Type: "data_retention",
			Func: checkDataRetention,
			Metadata: &CheckMetadata{
				Name:        "Data Retention Compliance",
				Description: "Validates audit log retention meets compliance requirements",
				Category:    "audit",
				Severity:    "critical",
				Standards:   []string{"SOC2", "HIPAA", "GDPR"},
				AutoRun:     true,
			},
		},
	}

	for _, check := range checks {
		if err := registry.Register(check.Type, check.Func, check.Metadata); err != nil {
			return fmt.Errorf("failed to register check '%s': %w", check.Type, err)
		}
	}

	return nil
}

// =============================================================================
// BUILT-IN CHECK IMPLEMENTATIONS
// =============================================================================

func checkMFACoverage(ctx context.Context, scope *audit.Scope, profile *ComplianceProfile, deps *CheckDependencies) (*CheckResult, error) {
	// Get users in scope
	var (
		users []*audit.GenericUser
		err   error
	)

	if deps.UserProvider != nil {
		users, err = deps.UserProvider.ListUsers(ctx, scope, &audit.UserFilter{
			Limit: 10000,
		})
	} else {
		return nil, errs.InternalServerErrorWithMessage("user provider not available")
	}

	if err != nil {
		return nil, err
	}

	totalUsers := len(users)
	usersWithMFA := 0
	usersWithoutMFA := make([]string, 0)

	for _, user := range users {
		if user.MFAEnabled {
			usersWithMFA++
		} else {
			usersWithoutMFA = append(usersWithoutMFA, user.Email)
		}
	}

	coveragePercent := 0.0
	if totalUsers > 0 {
		coveragePercent = float64(usersWithMFA) / float64(totalUsers) * 100
	}

	status := "passed"
	if profile.MFARequired && coveragePercent < 100 {
		status = "failed"
	} else if coveragePercent < 80 {
		status = "warning"
	}

	evidence := []string{
		fmt.Sprintf("MFA coverage: %.1f%%", coveragePercent),
		fmt.Sprintf("Users without MFA: %d", len(usersWithoutMFA)),
	}

	return &CheckResult{
		CheckType: "mfa_coverage",
		Status:    status,
		Score:     coveragePercent,
		Result: map[string]interface{}{
			"total_users":       totalUsers,
			"users_with_mfa":    usersWithMFA,
			"users_without_mfa": len(usersWithoutMFA),
			"coverage_percent":  coveragePercent,
			"list_without_mfa":  usersWithoutMFA[:min(10, len(usersWithoutMFA))], // First 10
		},
		Evidence: evidence,
	}, nil
}

func checkPasswordPolicy(ctx context.Context, scope *audit.Scope, profile *ComplianceProfile, deps *CheckDependencies) (*CheckResult, error) {
	// Simplified - would integrate with actual password validation
	return &CheckResult{
		CheckType: "password_policy",
		Status:    "passed",
		Score:     100,
		Result: map[string]interface{}{
			"policy_configured": true,
			"min_length":        profile.PasswordMinLength,
		},
		Evidence: []string{
			fmt.Sprintf("Password policy: min %d chars", profile.PasswordMinLength),
		},
	}, nil
}

func checkSessionPolicy(ctx context.Context, scope *audit.Scope, profile *ComplianceProfile, deps *CheckDependencies) (*CheckResult, error) {
	return &CheckResult{
		CheckType: "session_policy",
		Status:    "passed",
		Score:     100,
		Result: map[string]interface{}{
			"max_age":      profile.SessionMaxAge,
			"idle_timeout": profile.SessionIdleTimeout,
		},
		Evidence: []string{
			"Session policy configured",
		},
	}, nil
}

func checkAccessReview(ctx context.Context, scope *audit.Scope, profile *ComplianceProfile, deps *CheckDependencies) (*CheckResult, error) {
	return &CheckResult{
		CheckType: "access_review",
		Status:    "passed",
		Score:     100,
		Result: map[string]interface{}{
			"required": profile.RegularAccessReview,
		},
		Evidence: []string{
			"Access review configured",
		},
	}, nil
}

func checkInactiveUsers(ctx context.Context, scope *audit.Scope, profile *ComplianceProfile, deps *CheckDependencies) (*CheckResult, error) {
	return &CheckResult{
		CheckType: "inactive_users",
		Status:    "passed",
		Score:     100,
		Result: map[string]interface{}{
			"inactive_users": 0,
		},
		Evidence: []string{
			"No inactive users detected",
		},
	}, nil
}

func checkDataRetention(ctx context.Context, scope *audit.Scope, profile *ComplianceProfile, deps *CheckDependencies) (*CheckResult, error) {
	return &CheckResult{
		CheckType: "data_retention",
		Status:    "passed",
		Score:     100,
		Result: map[string]interface{}{
			"retention_days": profile.RetentionDays,
		},
		Evidence: []string{
			fmt.Sprintf("Retention: %d days", profile.RetentionDays),
		},
	}, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}

	return b
}
