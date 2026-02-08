package compliance

import (
	"context"
	"fmt"

	"github.com/xraph/authsome/core/audit"
)

// =============================================================================
// SCOPE RESOLVER - Hierarchical compliance profile resolution
// =============================================================================

// ScopeResolver resolves compliance profiles based on hierarchical scopes
// Supports inheritance where child scopes can make policies MORE strict (never less)
type ScopeResolver struct {
	repo Repository
}

// NewScopeResolver creates a new scope resolver
func NewScopeResolver(repo Repository) *ScopeResolver {
	return &ScopeResolver{
		repo: repo,
	}
}

// Resolve retrieves the effective compliance profile for a scope
// Walks up the hierarchy and merges profiles with child overrides taking precedence
func (r *ScopeResolver) Resolve(ctx context.Context, scope *audit.Scope) (*ComplianceProfile, error) {
	if scope == nil {
		return nil, fmt.Errorf("scope cannot be nil")
	}

	// Get profile for this scope
	profile, err := r.getProfileForScope(ctx, scope)
	if err != nil {
		return nil, err
	}

	// If no profile found and has parent, walk up
	if profile == nil && scope.ParentID != nil {
		parentScope, err := r.getParentScope(ctx, scope)
		if err != nil {
			return nil, err
		}
		if parentScope != nil {
			return r.Resolve(ctx, parentScope)
		}
	}

	return profile, nil
}

// ResolveWithInheritance resolves profile with full inheritance chain
// Returns the merged profile from system → app → org → team → role → user
func (r *ScopeResolver) ResolveWithInheritance(ctx context.Context, scope *audit.Scope) (*ComplianceProfile, error) {
	// Build inheritance chain
	chain, err := r.buildInheritanceChain(ctx, scope)
	if err != nil {
		return nil, err
	}

	if len(chain) == 0 {
		return nil, fmt.Errorf("no compliance profile found in scope hierarchy")
	}

	// Start with system-level defaults (root of chain)
	merged := chain[0]

	// Apply each level's overrides
	for i := 1; i < len(chain); i++ {
		merged = r.Inherit(merged, chain[i])
	}

	return merged, nil
}

// Inherit merges parent and child profiles with child overrides
// Child can make policies MORE strict (never less)
func (r *ScopeResolver) Inherit(parent, child *ComplianceProfile) *ComplianceProfile {
	if parent == nil {
		return child
	}
	if child == nil {
		return parent
	}

	// Start with parent as base
	merged := &ComplianceProfile{
		AppID:     child.AppID, // Use child's app ID
		Name:      child.Name,
		Standards: r.mergeStandards(parent.Standards, child.Standards),
		Status:    child.Status,

		// Security - Child can make MORE strict
		MFARequired:           parent.MFARequired || child.MFARequired,
		PasswordMinLength:     max(parent.PasswordMinLength, child.PasswordMinLength),
		PasswordRequireUpper:  parent.PasswordRequireUpper || child.PasswordRequireUpper,
		PasswordRequireLower:  parent.PasswordRequireLower || child.PasswordRequireLower,
		PasswordRequireNumber: parent.PasswordRequireNumber || child.PasswordRequireNumber,
		PasswordRequireSymbol: parent.PasswordRequireSymbol || child.PasswordRequireSymbol,
		PasswordExpiryDays:    minPositive(parent.PasswordExpiryDays, child.PasswordExpiryDays),

		// Session - Child can make MORE strict (shorter timeouts)
		SessionMaxAge:      minPositive(parent.SessionMaxAge, child.SessionMaxAge),
		SessionIdleTimeout: minPositive(parent.SessionIdleTimeout, child.SessionIdleTimeout),
		SessionIPBinding:   parent.SessionIPBinding || child.SessionIPBinding,

		// Audit - Child can require MORE retention
		RetentionDays:      max(parent.RetentionDays, child.RetentionDays),
		AuditLogExport:     parent.AuditLogExport || child.AuditLogExport,
		DetailedAuditTrail: parent.DetailedAuditTrail || child.DetailedAuditTrail,

		// Data - Child can be MORE restrictive
		DataResidency:       r.mergeDataResidency(parent.DataResidency, child.DataResidency),
		EncryptionAtRest:    parent.EncryptionAtRest || child.EncryptionAtRest,
		EncryptionInTransit: parent.EncryptionInTransit || child.EncryptionInTransit,

		// Access Control - Child can be MORE strict
		RBACRequired:        parent.RBACRequired || child.RBACRequired,
		LeastPrivilege:      parent.LeastPrivilege || child.LeastPrivilege,
		RegularAccessReview: parent.RegularAccessReview || child.RegularAccessReview,

		// Contacts - Child overrides
		ComplianceContact: r.coalesce(child.ComplianceContact, parent.ComplianceContact),
		DPOContact:        r.coalesce(child.DPOContact, parent.DPOContact),

		// Metadata - Merge with child taking precedence
		Metadata: r.mergeMetadata(parent.Metadata, child.Metadata),

		CreatedAt: child.CreatedAt,
		UpdatedAt: child.UpdatedAt,
	}

	return merged
}

// ValidateInheritance checks if child profile violates parent constraints
// Returns error if child is LESS strict than parent
func (r *ScopeResolver) ValidateInheritance(parent, child *ComplianceProfile) error {
	if parent == nil || child == nil {
		return nil
	}

	// Validate security requirements (child must be >= parent)
	if parent.MFARequired && !child.MFARequired {
		return fmt.Errorf("child scope cannot disable MFA when parent requires it")
	}

	if child.PasswordMinLength < parent.PasswordMinLength {
		return fmt.Errorf("child password min length (%d) cannot be less than parent (%d)",
			child.PasswordMinLength, parent.PasswordMinLength)
	}

	if parent.PasswordRequireUpper && !child.PasswordRequireUpper {
		return fmt.Errorf("child cannot disable uppercase requirement when parent requires it")
	}

	if parent.PasswordRequireLower && !child.PasswordRequireLower {
		return fmt.Errorf("child cannot disable lowercase requirement when parent requires it")
	}

	if parent.PasswordRequireNumber && !child.PasswordRequireNumber {
		return fmt.Errorf("child cannot disable number requirement when parent requires it")
	}

	if parent.PasswordRequireSymbol && !child.PasswordRequireSymbol {
		return fmt.Errorf("child cannot disable symbol requirement when parent requires it")
	}

	// Password expiry - 0 means never expires, so child can't have longer expiry than parent
	if parent.PasswordExpiryDays > 0 && (child.PasswordExpiryDays == 0 || child.PasswordExpiryDays > parent.PasswordExpiryDays) {
		return fmt.Errorf("child password expiry (%d days) cannot be longer than parent (%d days)",
			child.PasswordExpiryDays, parent.PasswordExpiryDays)
	}

	// Session - Child must be <= parent (shorter or equal timeout)
	if child.SessionMaxAge > parent.SessionMaxAge {
		return fmt.Errorf("child session max age (%d) cannot exceed parent (%d)",
			child.SessionMaxAge, parent.SessionMaxAge)
	}

	if child.SessionIdleTimeout > parent.SessionIdleTimeout {
		return fmt.Errorf("child session idle timeout (%d) cannot exceed parent (%d)",
			child.SessionIdleTimeout, parent.SessionIdleTimeout)
	}

	// Audit retention - Child must be >= parent (longer or equal retention)
	if child.RetentionDays < parent.RetentionDays {
		return fmt.Errorf("child retention days (%d) cannot be less than parent (%d)",
			child.RetentionDays, parent.RetentionDays)
	}

	return nil
}

// =============================================================================
// PRIVATE HELPER METHODS
// =============================================================================

// getProfileForScope retrieves profile directly for a scope (no inheritance)
func (r *ScopeResolver) getProfileForScope(ctx context.Context, scope *audit.Scope) (*ComplianceProfile, error) {
	// For now, only app-level profiles are stored in DB
	// Future: Support org, team, role, user-level profiles
	if scope.Type == audit.ScopeTypeApp {
		return r.repo.GetProfileByApp(ctx, scope.ID)
	}

	// TODO: Implement for other scope types when schema is extended
	return nil, nil
}

// getParentScope retrieves the parent scope
func (r *ScopeResolver) getParentScope(ctx context.Context, scope *audit.Scope) (*audit.Scope, error) {
	if scope.ParentID == nil {
		return nil, nil
	}

	// Determine parent type based on current type
	var parentType audit.ScopeType
	switch scope.Type {
	case audit.ScopeTypeUser:
		parentType = audit.ScopeTypeRole
	case audit.ScopeTypeRole:
		parentType = audit.ScopeTypeTeam
	case audit.ScopeTypeTeam:
		parentType = audit.ScopeTypeOrg
	case audit.ScopeTypeOrg:
		parentType = audit.ScopeTypeApp
	case audit.ScopeTypeApp:
		parentType = audit.ScopeTypeSystem
	case audit.ScopeTypeSystem:
		return nil, nil // System is root
	default:
		return nil, nil
	}

	return &audit.Scope{
		Type: parentType,
		ID:   *scope.ParentID,
	}, nil
}

// buildInheritanceChain walks up the scope hierarchy
func (r *ScopeResolver) buildInheritanceChain(ctx context.Context, scope *audit.Scope) ([]*ComplianceProfile, error) {
	chain := make([]*ComplianceProfile, 0)

	current := scope
	for current != nil {
		profile, err := r.getProfileForScope(ctx, current)
		if err != nil {
			return nil, err
		}

		if profile != nil {
			// Prepend to chain (so system is first, user is last)
			chain = append([]*ComplianceProfile{profile}, chain...)
		}

		// Get parent scope
		current, err = r.getParentScope(ctx, current)
		if err != nil {
			return nil, err
		}
	}

	return chain, nil
}

// Utility functions

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minPositive(a, b int) int {
	// 0 means unlimited/disabled
	if a == 0 {
		return b
	}
	if b == 0 {
		return a
	}
	if a < b {
		return a
	}
	return b
}

func (r *ScopeResolver) mergeStandards(parent, child []ComplianceStandard) []ComplianceStandard {
	// Merge unique standards from both
	seen := make(map[ComplianceStandard]bool)
	result := make([]ComplianceStandard, 0)

	for _, s := range parent {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}

	for _, s := range child {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}

	return result
}

func (r *ScopeResolver) mergeDataResidency(parent, child string) string {
	// Child residency requirement takes precedence if specified
	if child != "" {
		return child
	}
	return parent
}

func (r *ScopeResolver) coalesce(preferred, fallback string) string {
	if preferred != "" {
		return preferred
	}
	return fallback
}

func (r *ScopeResolver) mergeMetadata(parent, child map[string]interface{}) map[string]interface{} {
	if parent == nil {
		return child
	}
	if child == nil {
		return parent
	}

	// Deep merge with child taking precedence
	merged := make(map[string]interface{})
	for k, v := range parent {
		merged[k] = v
	}
	for k, v := range child {
		merged[k] = v
	}

	return merged
}
