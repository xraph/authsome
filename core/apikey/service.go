package apikey

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/schema"
)

// Config holds the API key service configuration
type Config struct {
	DefaultRateLimit int           `json:"default_rate_limit"`
	MaxRateLimit     int           `json:"max_rate_limit"`
	DefaultExpiry    time.Duration `json:"default_expiry"`
	MaxKeysPerUser   int           `json:"max_keys_per_user"`
	MaxKeysPerOrg    int           `json:"max_keys_per_org"`
	KeyLength        int           `json:"key_length"`
}

// Service handles API key operations
// Updated for V2 architecture: App → Environment → Organization
type Service struct {
	repo     Repository
	roleRepo RoleRepository // RBAC integration
	auditSvc *audit.Service
	config   Config
}

// NewService creates a new API key service
func NewService(repo Repository, auditSvc *audit.Service, cfg Config) *Service {
	// Set defaults
	if cfg.DefaultRateLimit == 0 {
		cfg.DefaultRateLimit = 1000
	}
	if cfg.MaxRateLimit == 0 {
		cfg.MaxRateLimit = 10000
	}
	if cfg.DefaultExpiry == 0 {
		cfg.DefaultExpiry = 365 * 24 * time.Hour // 1 year
	}
	if cfg.MaxKeysPerUser == 0 {
		cfg.MaxKeysPerUser = 10
	}
	if cfg.MaxKeysPerOrg == 0 {
		cfg.MaxKeysPerOrg = 100
	}
	if cfg.KeyLength == 0 {
		cfg.KeyLength = 32
	}

	return &Service{
		repo:     repo,
		roleRepo: nil, // Set via SetRoleRepository
		auditSvc: auditSvc,
		config:   cfg,
	}
}

// SetRoleRepository sets the role repository (for RBAC integration)
// This is set after service initialization to avoid circular dependencies
func (s *Service) SetRoleRepository(roleRepo RoleRepository) {
	s.roleRepo = roleRepo
}

// CreateAPIKey creates a new API key
func (s *Service) CreateAPIKey(ctx context.Context, req *CreateAPIKeyRequest) (*APIKey, error) {
	// Validate request
	if err := s.validateCreateRequest(ctx, req); err != nil {
		return nil, err
	}

	// Check limits
	if err := s.checkLimits(ctx, req.AppID, req.UserID, req.OrgID); err != nil {
		return nil, err
	}

	// Generate key
	keyBytes := make([]byte, s.config.KeyLength)
	if _, err := rand.Read(keyBytes); err != nil {
		return nil, APIKeyCreationFailed(fmt.Errorf("failed to generate key: %w", err))
	}
	key := base64.URLEncoding.EncodeToString(keyBytes)

	// Validate key type and scopes
	if err := s.validateKeyTypeAndScopes(req.KeyType, req.Scopes); err != nil {
		return nil, err
	}

	// Generate prefix for identification (includes key type)
	prefix := s.generatePrefix(req.KeyType, req.EnvironmentID)

	// Hash the key for storage
	keyHash := s.hashKey(key)

	// Set defaults
	rateLimit := req.RateLimit
	if rateLimit == 0 {
		rateLimit = s.config.DefaultRateLimit
	}
	if rateLimit > s.config.MaxRateLimit {
		rateLimit = s.config.MaxRateLimit
	}

	expiresAt := req.ExpiresAt
	if expiresAt == nil {
		expiry := time.Now().Add(s.config.DefaultExpiry)
		expiresAt = &expiry
	}

	// Create API key schema
	apiKey := &schema.APIKey{
		AppID:          req.AppID,
		EnvironmentID:  req.EnvironmentID,
		OrganizationID: req.OrgID,
		UserID:         req.UserID,
		Name:           req.Name,
		Description:    req.Description,
		Prefix:         prefix,
		KeyType:        string(req.KeyType),
		KeyHash:        keyHash,
		Scopes:         req.Scopes,
		Permissions:    req.Permissions,
		RateLimit:      rateLimit,
		AllowedIPs:     req.AllowedIPs,
		Active:         true,
		ExpiresAt:      expiresAt,
		Metadata:       req.Metadata,
		AuditableModel: schema.AuditableModel{
			ID:        xid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	if err := s.repo.CreateAPIKey(ctx, apiKey); err != nil {
		return nil, APIKeyCreationFailed(err)
	}

	// Audit log
	_ = s.auditSvc.Log(ctx, &req.UserID, "api_key.created", "api_key:"+apiKey.ID.String(), "", "", fmt.Sprintf(`{"name":"%s","scopes":["%s"]}`, req.Name, strings.Join(req.Scopes, `","`)))

	// Convert to DTO and return the full key only once
	result := FromSchemaAPIKey(apiKey)
	result.Key = prefix + "." + key
	return result, nil
}

// VerifyAPIKey verifies an API key and returns the associated key info
func (s *Service) VerifyAPIKey(ctx context.Context, req *VerifyAPIKeyRequest) (*VerifyAPIKeyResponse, error) {
	parts := strings.Split(req.Key, ".")
	if len(parts) != 2 {
		err := InvalidKeyFormat()
		return &VerifyAPIKeyResponse{
			Valid: false,
			Error: err.Message,
		}, nil
	}

	prefix := parts[0]
	keyPart := parts[1]

	// Find by prefix
	apiKey, err := s.repo.FindAPIKeyByPrefix(ctx, prefix)
	if err != nil {
		notFoundErr := APIKeyNotFound()
		return &VerifyAPIKeyResponse{
			Valid: false,
			Error: notFoundErr.Message,
		}, nil
	}

	// Verify hash
	if !s.verifyKeyHash(keyPart, apiKey.KeyHash) {
		invalidErr := InvalidAPIKeyHash()
		return &VerifyAPIKeyResponse{
			Valid: false,
			Error: invalidErr.Message,
		}, nil
	}

	// Check if active
	if !apiKey.Active {
		inactiveErr := APIKeyInactive()
		return &VerifyAPIKeyResponse{
			Valid: false,
			Error: inactiveErr.Message,
		}, nil
	}

	// Check expiration
	if apiKey.IsExpired() {
		expiredErr := APIKeyExpired()
		return &VerifyAPIKeyResponse{
			Valid: false,
			Error: expiredErr.Message,
		}, nil
	}

	// Check scope if required
	if req.RequiredScope != "" && !apiKey.HasScope(req.RequiredScope) {
		scopeErr := InsufficientScope(req.RequiredScope)
		return &VerifyAPIKeyResponse{
			Valid: false,
			Error: scopeErr.Message,
		}, nil
	}

	// Check permission if required
	if req.RequiredPermission != "" && !apiKey.HasPermission(req.RequiredPermission) {
		permErr := InsufficientPermission(req.RequiredPermission)
		return &VerifyAPIKeyResponse{
			Valid: false,
			Error: permErr.Message,
		}, nil
	}

	// Update usage
	if err := s.repo.UpdateAPIKeyUsage(ctx, apiKey.ID, req.IP, req.UserAgent); err != nil {
		// Log error but don't fail verification
		fmt.Printf("Failed to update API key usage: %v\n", err)
	}

	return &VerifyAPIKeyResponse{
		Valid:  true,
		APIKey: FromSchemaAPIKey(apiKey),
	}, nil
}

// GetAPIKey retrieves an API key by ID
func (s *Service) GetAPIKey(ctx context.Context, appID, id, userID xid.ID, orgID *xid.ID) (*APIKey, error) {
	apiKey, err := s.repo.FindAPIKeyByID(ctx, id)
	if err != nil {
		return nil, APIKeyNotFound()
	}

	// Check ownership - app context
	if apiKey.AppID != appID {
		return nil, AccessDenied("wrong app")
	}

	// Check ownership - user context
	if apiKey.UserID != userID {
		return nil, AccessDenied("wrong user")
	}

	// Check ownership - org context (if org-scoped)
	if orgID != nil && !orgID.IsNil() {
		if apiKey.OrganizationID == nil || *apiKey.OrganizationID != *orgID {
			return nil, AccessDenied("wrong organization")
		}
	}

	return FromSchemaAPIKey(apiKey), nil
}

// ListAPIKeys lists API keys with filtering and pagination
func (s *Service) ListAPIKeys(ctx context.Context, filter *ListAPIKeysFilter) (*ListAPIKeysResponse, error) {
	// Get paginated results from repository (returns schema types)
	pageResp, err := s.repo.ListAPIKeys(ctx, filter)
	if err != nil {
		return nil, err
	}

	// Convert schema types to DTOs
	dtoKeys := FromSchemaAPIKeys(pageResp.Data)

	// Return paginated response with DTOs
	return &ListAPIKeysResponse{
		Data:       dtoKeys,
		Pagination: pageResp.Pagination,
	}, nil
}

// UpdateAPIKey updates an API key
func (s *Service) UpdateAPIKey(ctx context.Context, appID, id, userID xid.ID, orgID *xid.ID, req *UpdateAPIKeyRequest) (*APIKey, error) {
	apiKey, err := s.repo.FindAPIKeyByID(ctx, id)
	if err != nil {
		return nil, APIKeyNotFound()
	}

	// Check ownership
	if apiKey.AppID != appID {
		return nil, AccessDenied("wrong app")
	}
	if apiKey.UserID != userID {
		return nil, AccessDenied("wrong user")
	}
	if orgID != nil && !orgID.IsNil() {
		if apiKey.OrganizationID == nil || *apiKey.OrganizationID != *orgID {
			return nil, AccessDenied("wrong organization")
		}
	}

	// Update fields
	if req.Name != nil {
		apiKey.Name = *req.Name
	}
	if req.Description != nil {
		apiKey.Description = *req.Description
	}
	if req.Scopes != nil {
		apiKey.Scopes = req.Scopes
	}
	if req.Permissions != nil {
		apiKey.Permissions = req.Permissions
	}
	if req.RateLimit != nil {
		rateLimit := *req.RateLimit
		if rateLimit > s.config.MaxRateLimit {
			rateLimit = s.config.MaxRateLimit
		}
		apiKey.RateLimit = rateLimit
	}
	if req.ExpiresAt != nil {
		apiKey.ExpiresAt = req.ExpiresAt
	}
	if req.Active != nil {
		apiKey.Active = *req.Active
	}
	if req.Metadata != nil {
		apiKey.Metadata = req.Metadata
	}

	if err := s.repo.UpdateAPIKey(ctx, apiKey); err != nil {
		return nil, APIKeyUpdateFailed(err)
	}

	// Audit log
	_ = s.auditSvc.Log(ctx, &userID, "api_key.updated", "api_key:"+id.String(), "", "", fmt.Sprintf(`{"name":"%s"}`, apiKey.Name))

	return FromSchemaAPIKey(apiKey), nil
}

// DeleteAPIKey deletes an API key
func (s *Service) DeleteAPIKey(ctx context.Context, appID, id, userID xid.ID, orgID *xid.ID) error {
	apiKey, err := s.repo.FindAPIKeyByID(ctx, id)
	if err != nil {
		return APIKeyNotFound()
	}

	// Check ownership
	if apiKey.AppID != appID {
		return AccessDenied("wrong app")
	}
	if apiKey.UserID != userID {
		return AccessDenied("wrong user")
	}
	if orgID != nil && !orgID.IsNil() {
		if apiKey.OrganizationID == nil || *apiKey.OrganizationID != *orgID {
			return AccessDenied("wrong organization")
		}
	}

	if err := s.repo.DeleteAPIKey(ctx, id); err != nil {
		return APIKeyDeletionFailed(err)
	}

	// Audit log
	_ = s.auditSvc.Log(ctx, &userID, "api_key.deleted", "api_key:"+id.String(), "", "", fmt.Sprintf(`{"name":"%s"}`, apiKey.Name))

	return nil
}

// RotateAPIKey rotates an API key (creates a new key with same settings)
func (s *Service) RotateAPIKey(ctx context.Context, req *RotateAPIKeyRequest) (*APIKey, error) {
	// Get existing key
	existingKey, err := s.repo.FindAPIKeyByID(ctx, req.ID)
	if err != nil {
		return nil, APIKeyNotFound()
	}

	// Check ownership
	if existingKey.AppID != req.AppID {
		return nil, AccessDenied("wrong app")
	}
	if existingKey.UserID != req.UserID {
		return nil, AccessDenied("wrong user")
	}
	if req.OrganizationID != nil && !req.OrganizationID.IsNil() {
		if existingKey.OrganizationID == nil || *existingKey.OrganizationID != *req.OrganizationID {
			return nil, AccessDenied("wrong organization")
		}
	}

	// Create new key with same settings
	createReq := &CreateAPIKeyRequest{
		AppID:         existingKey.AppID,
		EnvironmentID: existingKey.EnvironmentID,
		OrgID:         existingKey.OrganizationID,
		UserID:        existingKey.UserID,
		Name:          existingKey.Name,
		Description:   existingKey.Description,
		Scopes:        existingKey.Scopes,
		Permissions:   existingKey.Permissions,
		RateLimit:     existingKey.RateLimit,
		ExpiresAt:     req.ExpiresAt,
		Metadata:      existingKey.Metadata,
	}

	newKey, err := s.CreateAPIKey(ctx, createReq)
	if err != nil {
		return nil, APIKeyRotationFailed(err)
	}

	// Deactivate old key
	if err := s.repo.DeactivateAPIKey(ctx, req.ID); err != nil {
		// Log error but don't fail rotation
		fmt.Printf("Failed to deactivate old API key: %v\n", err)
	}

	// Audit log
	_ = s.auditSvc.Log(ctx, &req.UserID, "api_key.rotated", "api_key:"+req.ID.String(), "", "", fmt.Sprintf(`{"name":"%s","old_key_id":"%s","new_key_id":"%s"}`, existingKey.Name, req.ID.String(), newKey.ID.String()))

	return newKey, nil
}

// CleanupExpired removes expired API keys
func (s *Service) CleanupExpired(ctx context.Context) (int, error) {
	return s.repo.CleanupExpiredAPIKeys(ctx)
}

// Helper methods

func (s *Service) validateCreateRequest(ctx context.Context, req *CreateAPIKeyRequest) error {
	if req.AppID.IsNil() {
		return MissingAppContext()
	}
	if req.EnvironmentID.IsNil() {
		return MissingEnvContext()
	}
	if req.UserID.IsNil() {
		return AccessDenied("user ID required")
	}
	if req.Name == "" {
		return AccessDenied("name is required")
	}
	if req.KeyType == "" {
		return AccessDenied("key type is required")
	}
	if !req.KeyType.IsValid() {
		return AccessDenied("invalid key type: must be pk, sk, or rk")
	}
	// Note: Scopes are validated in validateKeyTypeAndScopes
	return nil
}

func (s *Service) checkLimits(ctx context.Context, appID, userID xid.ID, orgID *xid.ID) error {
	// Check user limit (per app)
	userIDPtr := &userID
	userCount, err := s.repo.CountAPIKeys(ctx, appID, nil, nil, userIDPtr)
	if err != nil {
		return err
	}
	if userCount >= s.config.MaxKeysPerUser {
		return MaxKeysReached(s.config.MaxKeysPerUser)
	}

	// Check org limit (if org-scoped)
	if orgID != nil && !orgID.IsNil() {
		orgCount, err := s.repo.CountAPIKeys(ctx, appID, nil, orgID, nil)
		if err != nil {
			return err
		}
		if orgCount >= s.config.MaxKeysPerOrg {
			return MaxKeysReached(s.config.MaxKeysPerOrg)
		}
	}

	return nil
}

func (s *Service) generatePrefix(keyType KeyType, envID xid.ID) string {
	// Generate a random suffix for uniqueness
	bytes := make([]byte, 6)
	rand.Read(bytes)
	suffix := base64.URLEncoding.EncodeToString(bytes)[:8]

	// Determine environment name (could be enhanced with actual env lookup)
	envName := "prod" // Default to prod
	// You could add environment name lookup here based on envID
	// For now, we'll use a simple default

	// Create prefix based on key type
	// Format: {type}_{env}_{random}
	// Examples:
	// - pk_test_a1b2c3d4
	// - sk_prod_x9y8z7w6
	// - rk_dev_m3n2o1p0
	return fmt.Sprintf("%s_%s_%s", keyType, envName, suffix)
}

// validateKeyTypeAndScopes validates that the key type and scopes are compatible
func (s *Service) validateKeyTypeAndScopes(keyType KeyType, scopes []string) error {
	// For restricted keys, require explicit scopes
	if keyType == KeyTypeRestricted && len(scopes) == 0 {
		return AccessDenied("restricted keys require at least one explicit scope")
	}

	// For publishable keys, only allow safe scopes
	if keyType == KeyTypePublishable {
		for _, scope := range scopes {
			if !IsSafeForPublicKey(scope) {
				return AccessDenied(fmt.Sprintf("scope '%s' is not allowed for publishable keys", scope))
			}
		}
	}

	// For secret keys, no restrictions (admin:full is implied)
	// But still validate that scopes are provided
	if len(scopes) == 0 && keyType != KeyTypeSecret {
		return AccessDenied("at least one scope is required")
	}

	return nil
}

func (s *Service) hashKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return base64.URLEncoding.EncodeToString(hash[:])
}

func (s *Service) verifyKeyHash(key, hash string) bool {
	return s.hashKey(key) == hash
}

// ============================================================================
// RBAC Integration Methods (Hybrid Approach)
// ============================================================================

// AssignRole assigns a role to an API key
func (s *Service) AssignRole(ctx context.Context, apiKeyID, roleID xid.ID, orgID *xid.ID, createdBy *xid.ID) error {
	if s.roleRepo == nil {
		return fmt.Errorf("RBAC not configured")
	}

	// Verify API key exists
	_, err := s.repo.FindAPIKeyByID(ctx, apiKeyID)
	if err != nil {
		return APIKeyNotFound()
	}

	// Assign role
	if err := s.roleRepo.AssignRole(ctx, apiKeyID, roleID, orgID, createdBy); err != nil {
		return fmt.Errorf("failed to assign role: %w", err)
	}

	// Audit log
	if createdBy != nil {
		_ = s.auditSvc.Log(ctx, createdBy, "api_key.role_assigned", "api_key:"+apiKeyID.String(), "", "",
			fmt.Sprintf(`{"api_key_id":"%s","role_id":"%s"}`, apiKeyID, roleID))
	}

	return nil
}

// UnassignRole removes a role from an API key
func (s *Service) UnassignRole(ctx context.Context, apiKeyID, roleID xid.ID, orgID *xid.ID, actorID *xid.ID) error {
	if s.roleRepo == nil {
		return fmt.Errorf("RBAC not configured")
	}

	if err := s.roleRepo.UnassignRole(ctx, apiKeyID, roleID, orgID); err != nil {
		return fmt.Errorf("failed to unassign role: %w", err)
	}

	// Audit log
	if actorID != nil {
		_ = s.auditSvc.Log(ctx, actorID, "api_key.role_unassigned", "api_key:"+apiKeyID.String(), "", "",
			fmt.Sprintf(`{"api_key_id":"%s","role_id":"%s"}`, apiKeyID, roleID))
	}

	return nil
}

// GetRoles retrieves all roles assigned to an API key
func (s *Service) GetRoles(ctx context.Context, apiKeyID xid.ID, orgID *xid.ID) ([]*Role, error) {
	if s.roleRepo == nil {
		return []*Role{}, nil
	}

	roles, err := s.roleRepo.GetRoles(ctx, apiKeyID, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get API key roles: %w", err)
	}

	// Convert to DTOs
	result := make([]*Role, len(roles))
	for i, role := range roles {
		result[i] = &Role{
			ID:          role.ID,
			Name:        role.Name,
			Description: role.Description,
		}
	}

	return result, nil
}

// GetPermissions retrieves all permissions for an API key through its roles
func (s *Service) GetPermissions(ctx context.Context, apiKeyID xid.ID, orgID *xid.ID) ([]*Permission, error) {
	if s.roleRepo == nil {
		return []*Permission{}, nil
	}

	permissions, err := s.roleRepo.GetPermissions(ctx, apiKeyID, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get API key permissions: %w", err)
	}

	// Convert to DTOs (parse action:resource from name)
	result := make([]*Permission, len(permissions))
	for i, perm := range permissions {
		action, resource := parsePermissionName(perm.Name)
		result[i] = &Permission{
			ID:       perm.ID,
			Action:   action,
			Resource: resource,
		}
	}

	return result, nil
}

// GetEffectivePermissions computes all effective permissions for an API key
// This includes:
// 1. API key's own permissions (scopes + roles)
// 2. If delegation enabled: creator's permissions
// 3. If impersonation set: target user's permissions
func (s *Service) GetEffectivePermissions(ctx context.Context, apiKeyID xid.ID, orgID *xid.ID) (*EffectivePermissions, error) {
	// Get API key
	apiKey, err := s.repo.FindAPIKeyByID(ctx, apiKeyID)
	if err != nil {
		return nil, APIKeyNotFound()
	}

	result := &EffectivePermissions{
		Scopes:      apiKey.Scopes,
		Permissions: []*Permission{},
	}

	if s.roleRepo == nil {
		// RBAC not configured, return scopes only
		return result, nil
	}

	// 1. Get API key's own permissions
	keyPerms, err := s.roleRepo.GetPermissions(ctx, apiKeyID, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get key permissions: %w", err)
	}
	for _, perm := range keyPerms {
		action, resource := parsePermissionName(perm.Name)
		result.Permissions = append(result.Permissions, &Permission{
			ID:       perm.ID,
			Action:   action,
			Resource: resource,
			Source:   "key",
		})
	}

	// 2. If delegation enabled, add creator's permissions
	if apiKey.DelegateUserPermissions {
		creatorPerms, err := s.roleRepo.GetCreatorPermissions(ctx, apiKey.UserID, orgID)
		if err != nil {
			return nil, fmt.Errorf("failed to get creator permissions: %w", err)
		}
		for _, perm := range creatorPerms {
			action, resource := parsePermissionName(perm.Name)
			result.Permissions = append(result.Permissions, &Permission{
				ID:       perm.ID,
				Action:   action,
				Resource: resource,
				Source:   "creator",
			})
		}
		result.DelegatedFromCreator = true
	}

	// 3. If impersonation set, add target user's permissions
	if apiKey.ImpersonateUserID != nil {
		impersonatePerms, err := s.roleRepo.GetCreatorPermissions(ctx, *apiKey.ImpersonateUserID, orgID)
		if err != nil {
			return nil, fmt.Errorf("failed to get impersonation permissions: %w", err)
		}
		for _, perm := range impersonatePerms {
			action, resource := parsePermissionName(perm.Name)
			result.Permissions = append(result.Permissions, &Permission{
				ID:       perm.ID,
				Action:   action,
				Resource: resource,
				Source:   "impersonation",
			})
		}
		result.ImpersonatingUser = apiKey.ImpersonateUserID
	}

	// Deduplicate permissions
	result.Permissions = deduplicatePermissions(result.Permissions)

	return result, nil
}

// CanAccess checks if an API key can perform a specific action on a resource
// This checks both scopes (legacy) and RBAC permissions (new)
func (s *Service) CanAccess(ctx context.Context, apiKey *APIKey, action, resource string, orgID *xid.ID) (bool, error) {
	// Check scopes first (backward compatibility)
	scopeString := fmt.Sprintf("%s:%s", resource, action)
	if apiKey.HasScope(scopeString) || apiKey.HasScope("admin:full") {
		return true, nil
	}

	// Check wildcard scopes
	if apiKey.HasScopeWildcard(scopeString) {
		return true, nil
	}

	// Check RBAC permissions
	if s.roleRepo != nil {
		effectivePerms, err := s.GetEffectivePermissions(ctx, apiKey.ID, orgID)
		if err != nil {
			return false, err
		}

		// Check if any permission matches
		for _, perm := range effectivePerms.Permissions {
			if matchesPermission(perm, action, resource) {
				return true, nil
			}
		}
	}

	return false, nil
}

// BulkAssignRoles assigns multiple roles to an API key
func (s *Service) BulkAssignRoles(ctx context.Context, apiKeyID xid.ID, roleIDs []xid.ID, orgID *xid.ID, createdBy *xid.ID) error {
	if s.roleRepo == nil {
		return fmt.Errorf("RBAC not configured")
	}

	if err := s.roleRepo.BulkAssignRoles(ctx, apiKeyID, roleIDs, orgID, createdBy); err != nil {
		return fmt.Errorf("failed to bulk assign roles: %w", err)
	}

	// Audit log
	if createdBy != nil {
		_ = s.auditSvc.Log(ctx, createdBy, "api_key.roles_bulk_assigned", "api_key:"+apiKeyID.String(), "", "",
			fmt.Sprintf(`{"api_key_id":"%s","role_count":%d}`, apiKeyID, len(roleIDs)))
	}

	return nil
}

// Helper functions

func deduplicatePermissions(permissions []*Permission) []*Permission {
	seen := make(map[string]bool)
	result := []*Permission{}

	for _, perm := range permissions {
		key := fmt.Sprintf("%s:%s", perm.Action, perm.Resource)
		if !seen[key] {
			seen[key] = true
			result = append(result, perm)
		}
	}

	return result
}

func matchesPermission(perm *Permission, action, resource string) bool {
	// Wildcard matching
	if perm.Action == "*" && perm.Resource == "*" {
		return true // Full admin
	}
	if perm.Action == "*" && perm.Resource == resource {
		return true // All actions on resource
	}
	if perm.Action == action && perm.Resource == "*" {
		return true // Specific action on all resources
	}
	if perm.Action == action && perm.Resource == resource {
		return true // Exact match
	}
	return false
}

// parsePermissionName parses a permission name like "view:users" into action and resource
func parsePermissionName(name string) (action, resource string) {
	for i := 0; i < len(name); i++ {
		if name[i] == ':' {
			return name[:i], name[i+1:]
		}
	}
	// If no colon, treat the whole name as action with empty resource
	return name, ""
}
