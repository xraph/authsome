package secrets

import (
	"context"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/forge"

	"github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/plugins/secrets/core"
	"github.com/xraph/authsome/plugins/secrets/schema"
)

// Service provides secret management operations
type Service struct {
	repo       Repository
	encryption *EncryptionService
	validator  *SchemaValidator
	auditSvc   *audit.Service
	config     *Config
	logger     forge.Logger
}

// NewService creates a new secrets service
func NewService(
	repo Repository,
	encryption *EncryptionService,
	validator *SchemaValidator,
	auditSvc *audit.Service,
	config *Config,
	logger forge.Logger,
) *Service {
	return &Service{
		repo:       repo,
		encryption: encryption,
		validator:  validator,
		auditSvc:   auditSvc,
		config:     config,
		logger:     logger,
	}
}

// =============================================================================
// Secret CRUD Operations
// =============================================================================

// Create creates a new secret
func (s *Service) Create(ctx context.Context, req *core.CreateSecretRequest) (*core.SecretDTO, error) {
	// Extract context values
	appID, err := contexts.RequireAppID(ctx)
	if err != nil {
		return nil, core.ErrAppContextRequired()
	}

	envID, err := contexts.RequireEnvironmentID(ctx)
	if err != nil {
		return nil, core.ErrEnvironmentContextRequired()
	}

	userID, _ := contexts.GetUserID(ctx)

	// Validate path
	if req.Path == "" {
		return nil, core.ErrPathRequired()
	}

	path := core.NormalizePath(req.Path)
	_, key, err := core.ParsePath(path)
	if err != nil {
		return nil, err
	}

	// Validate value
	if req.Value == nil {
		return nil, core.ErrValueRequired()
	}

	// Determine value type
	valueType := core.SecretValueTypePlain
	if req.ValueType != "" {
		vt, ok := core.ParseSecretValueType(req.ValueType)
		if !ok {
			return nil, core.ErrInvalidValueType(req.ValueType)
		}
		valueType = vt
	} else {
		// Auto-detect value type
		valueType = s.validator.DetectValueType(req.Value)
	}

	// Validate schema if provided
	if req.Schema != "" {
		if err := s.validator.ValidateSchema(req.Schema); err != nil {
			return nil, err
		}
	}

	// Validate value against schema
	if err := s.validator.ValidateValue(req.Value, valueType, req.Schema); err != nil {
		return nil, err
	}

	// Serialize value
	serialized, err := s.validator.SerializeValue(req.Value, valueType)
	if err != nil {
		return nil, err
	}

	// Encrypt value
	encrypted, nonce, err := s.encryption.Encrypt(serialized, appID.String(), envID.String())
	if err != nil {
		return nil, err
	}

	// Create secret
	now := time.Now().UTC()
	secret := &schema.Secret{
		ID:             xid.New(),
		AppID:          appID,
		EnvironmentID:  envID,
		Path:           path,
		Key:            key,
		ValueType:      schema.SecretValueType(valueType),
		EncryptedValue: encrypted,
		Nonce:          nonce,
		SchemaJSON:     req.Schema,
		Description:    req.Description,
		Tags:           req.Tags,
		Metadata:       req.Metadata,
		Version:        1,
		IsActive:       true,
		ExpiresAt:      req.ExpiresAt,
		CreatedBy:      userID,
		UpdatedBy:      userID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.repo.Create(ctx, secret); err != nil {
		// Check for unique constraint violation
		if isUniqueViolation(err) {
			return nil, core.ErrSecretExists(path)
		}
		return nil, err
	}

	// Create initial version history
	version := &schema.SecretVersion{
		ID:             xid.New(),
		SecretID:       secret.ID,
		Version:        1,
		EncryptedValue: encrypted,
		Nonce:          nonce,
		ValueType:      schema.SecretValueType(valueType),
		SchemaJSON:     req.Schema,
		ChangedBy:      userID,
		ChangeReason:   "Initial creation",
		CreatedAt:      now,
	}
	_ = s.repo.CreateVersion(ctx, version)

	// Log access
	s.logAccess(ctx, secret.ID, path, "create", true, "")

	return s.toDTO(secret), nil
}

// Get retrieves a secret by ID (without value)
func (s *Service) Get(ctx context.Context, id xid.ID) (*core.SecretDTO, error) {
	secret, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return s.toDTO(secret), nil
}

// GetByPath retrieves a secret by path
func (s *Service) GetByPath(ctx context.Context, path string) (*core.SecretDTO, error) {
	appID, err := contexts.RequireAppID(ctx)
	if err != nil {
		return nil, core.ErrAppContextRequired()
	}

	envID, err := contexts.RequireEnvironmentID(ctx)
	if err != nil {
		return nil, core.ErrEnvironmentContextRequired()
	}

	path = core.NormalizePath(path)
	secret, err := s.repo.FindByPath(ctx, appID, envID, path)
	if err != nil {
		return nil, err
	}

	return s.toDTO(secret), nil
}

// GetValue retrieves and decrypts the secret value
func (s *Service) GetValue(ctx context.Context, id xid.ID) (interface{}, error) {
	secret, err := s.repo.FindByID(ctx, id)
	if err != nil {
		s.logAccess(ctx, id, "", "read", false, "secret not found")
		return nil, err
	}

	// Check expiry
	if secret.IsExpired() {
		s.logAccess(ctx, id, secret.Path, "read", false, "secret expired")
		return nil, core.ErrSecretExpired(secret.Path)
	}

	// Decrypt value
	plaintext, err := s.encryption.Decrypt(
		secret.EncryptedValue,
		secret.Nonce,
		secret.AppID.String(),
		secret.EnvironmentID.String(),
	)
	if err != nil {
		s.logAccess(ctx, id, secret.Path, "read", false, "decryption failed")
		return nil, err
	}

	// Parse value based on type
	value, err := s.validator.ParseValue(string(plaintext), core.SecretValueType(secret.ValueType))
	if err != nil {
		return nil, err
	}

	// Log access (only if config allows)
	if s.config != nil && s.config.Audit.LogReads {
		s.logAccess(ctx, id, secret.Path, "read", true, "")
	}

	return value, nil
}

// GetValueByPath retrieves and decrypts a secret by path
func (s *Service) GetValueByPath(ctx context.Context, path string) (interface{}, error) {
	appID, err := contexts.RequireAppID(ctx)
	if err != nil {
		return nil, core.ErrAppContextRequired()
	}

	envID, err := contexts.RequireEnvironmentID(ctx)
	if err != nil {
		return nil, core.ErrEnvironmentContextRequired()
	}

	path = core.NormalizePath(path)
	secret, err := s.repo.FindByPath(ctx, appID, envID, path)
	if err != nil {
		return nil, err
	}

	return s.GetValue(ctx, secret.ID)
}

// GetWithValue retrieves a secret including its decrypted value
func (s *Service) GetWithValue(ctx context.Context, id xid.ID) (*core.SecretWithValueDTO, error) {
	dto, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	value, err := s.GetValue(ctx, id)
	if err != nil {
		return nil, err
	}

	return &core.SecretWithValueDTO{
		SecretDTO: *dto,
		Value:     value,
	}, nil
}

// Update updates a secret and creates a new version
func (s *Service) Update(ctx context.Context, id xid.ID, req *core.UpdateSecretRequest) (*core.SecretDTO, error) {
	secret, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	userID, _ := contexts.GetUserID(ctx)
	now := time.Now().UTC()
	valueChanged := false

	// Update value if provided
	if req.Value != nil {
		valueType := core.SecretValueType(secret.ValueType)
		if req.ValueType != "" {
			vt, ok := core.ParseSecretValueType(req.ValueType)
			if !ok {
				return nil, core.ErrInvalidValueType(req.ValueType)
			}
			valueType = vt
		}

		schemaJSON := secret.SchemaJSON
		if req.Schema != "" {
			if err := s.validator.ValidateSchema(req.Schema); err != nil {
				return nil, err
			}
			schemaJSON = req.Schema
		}

		// Validate value
		if err := s.validator.ValidateValue(req.Value, valueType, schemaJSON); err != nil {
			return nil, err
		}

		// Serialize and encrypt
		serialized, err := s.validator.SerializeValue(req.Value, valueType)
		if err != nil {
			return nil, err
		}

		encrypted, nonce, err := s.encryption.Encrypt(serialized, secret.AppID.String(), secret.EnvironmentID.String())
		if err != nil {
			return nil, err
		}

		secret.EncryptedValue = encrypted
		secret.Nonce = nonce
		secret.ValueType = schema.SecretValueType(valueType)
		secret.SchemaJSON = schemaJSON
		secret.Version++
		valueChanged = true

		// Create version history
		version := &schema.SecretVersion{
			ID:             xid.New(),
			SecretID:       secret.ID,
			Version:        secret.Version,
			EncryptedValue: encrypted,
			Nonce:          nonce,
			ValueType:      schema.SecretValueType(valueType),
			SchemaJSON:     schemaJSON,
			ChangedBy:      userID,
			ChangeReason:   req.ChangeReason,
			CreatedAt:      now,
		}
		_ = s.repo.CreateVersion(ctx, version)

		// Clean up old versions if needed
		if s.config != nil && s.config.Versioning.MaxVersions > 0 {
			_ = s.repo.DeleteOldVersions(ctx, secret.ID, s.config.Versioning.MaxVersions)
		}
	}

	// Update metadata fields
	if req.Description != "" {
		secret.Description = req.Description
	}
	if req.Tags != nil {
		secret.Tags = req.Tags
	}
	if req.Metadata != nil {
		secret.Metadata = req.Metadata
	}
	if req.ExpiresAt != nil {
		secret.ExpiresAt = req.ExpiresAt
	}
	if req.ClearExpiry {
		secret.ExpiresAt = nil
	}

	secret.UpdatedBy = userID
	secret.UpdatedAt = now

	if err := s.repo.Update(ctx, secret); err != nil {
		return nil, err
	}

	// Log access
	action := "update"
	if valueChanged {
		action = "update_value"
	}
	s.logAccess(ctx, id, secret.Path, action, true, "")

	return s.toDTO(secret), nil
}

// Delete soft-deletes a secret
func (s *Service) Delete(ctx context.Context, id xid.ID) error {
	secret, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}

	s.logAccess(ctx, id, secret.Path, "delete", true, "")
	return nil
}

// List lists secrets with filtering and pagination
func (s *Service) List(ctx context.Context, query *core.ListSecretsQuery) ([]*core.SecretDTO, *pagination.Pagination, error) {
	appID, err := contexts.RequireAppID(ctx)
	if err != nil {
		return nil, nil, core.ErrAppContextRequired()
	}

	envID, err := contexts.RequireEnvironmentID(ctx)
	if err != nil {
		return nil, nil, core.ErrEnvironmentContextRequired()
	}

	if query == nil {
		query = &core.ListSecretsQuery{}
	}
	if query.PageSize == 0 {
		query.PageSize = 20
	}
	if query.Page == 0 {
		query.Page = 1
	}

	secrets, total, err := s.repo.List(ctx, appID, envID, query)
	if err != nil {
		return nil, nil, err
	}

	dtos := make([]*core.SecretDTO, len(secrets))
	for i, secret := range secrets {
		dtos[i] = s.toDTO(secret)
	}

	pag := &pagination.Pagination{
		Page:       query.Page,
		PageSize:   query.PageSize,
		TotalItems: total,
		TotalPages: (total + query.PageSize - 1) / query.PageSize,
	}

	return dtos, pag, nil
}

// =============================================================================
// Version Operations
// =============================================================================

// GetVersions retrieves version history for a secret
func (s *Service) GetVersions(ctx context.Context, id xid.ID, page, pageSize int) ([]*core.SecretVersionDTO, *pagination.Pagination, error) {
	// Verify secret exists
	_, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}

	if pageSize == 0 {
		pageSize = 20
	}
	if page == 0 {
		page = 1
	}

	versions, total, err := s.repo.ListVersions(ctx, id, page, pageSize)
	if err != nil {
		return nil, nil, err
	}

	dtos := make([]*core.SecretVersionDTO, len(versions))
	for i, v := range versions {
		dtos[i] = &core.SecretVersionDTO{
			ID:           v.ID.String(),
			Version:      v.Version,
			ValueType:    string(v.ValueType),
			HasSchema:    v.SchemaJSON != "",
			ChangedBy:    v.ChangedBy.String(),
			ChangeReason: v.ChangeReason,
			CreatedAt:    v.CreatedAt,
		}
	}

	pag := &pagination.Pagination{
		Page:       page,
		PageSize:   pageSize,
		TotalItems: total,
		TotalPages: (total + pageSize - 1) / pageSize,
	}

	return dtos, pag, nil
}

// Rollback rolls back a secret to a previous version
func (s *Service) Rollback(ctx context.Context, id xid.ID, targetVersion int, reason string) (*core.SecretDTO, error) {
	secret, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	version, err := s.repo.FindVersion(ctx, id, targetVersion)
	if err != nil {
		return nil, err
	}

	userID, _ := contexts.GetUserID(ctx)
	now := time.Now().UTC()

	// Create new version from rollback
	newVersion := &schema.SecretVersion{
		ID:             xid.New(),
		SecretID:       secret.ID,
		Version:        secret.Version + 1,
		EncryptedValue: version.EncryptedValue,
		Nonce:          version.Nonce,
		ValueType:      version.ValueType,
		SchemaJSON:     version.SchemaJSON,
		ChangedBy:      userID,
		ChangeReason:   reason,
		CreatedAt:      now,
	}
	if newVersion.ChangeReason == "" {
		newVersion.ChangeReason = "Rollback to version " + string(rune(targetVersion+'0'))
	}
	_ = s.repo.CreateVersion(ctx, newVersion)

	// Update secret
	secret.EncryptedValue = version.EncryptedValue
	secret.Nonce = version.Nonce
	secret.ValueType = version.ValueType
	secret.SchemaJSON = version.SchemaJSON
	secret.Version++
	secret.UpdatedBy = userID
	secret.UpdatedAt = now

	if err := s.repo.Update(ctx, secret); err != nil {
		return nil, core.ErrRollbackFailed("failed to update secret", err)
	}

	s.logAccess(ctx, id, secret.Path, "rollback", true, "")
	return s.toDTO(secret), nil
}

// =============================================================================
// Stats Operations
// =============================================================================

// GetStats returns statistics about secrets
func (s *Service) GetStats(ctx context.Context) (*core.StatsDTO, error) {
	appID, err := contexts.RequireAppID(ctx)
	if err != nil {
		return nil, core.ErrAppContextRequired()
	}

	envID, err := contexts.RequireEnvironmentID(ctx)
	if err != nil {
		return nil, core.ErrEnvironmentContextRequired()
	}

	totalSecrets, err := s.repo.CountSecrets(ctx, appID, envID)
	if err != nil {
		return nil, err
	}

	totalVersions, err := s.repo.CountVersions(ctx, appID, envID)
	if err != nil {
		return nil, err
	}

	secretsByType, err := s.repo.GetSecretsByType(ctx, appID, envID)
	if err != nil {
		return nil, err
	}

	expiringSecrets, err := s.repo.CountExpiringSecrets(ctx, appID, envID, 30)
	if err != nil {
		return nil, err
	}

	return &core.StatsDTO{
		TotalSecrets:    totalSecrets,
		TotalVersions:   totalVersions,
		SecretsByType:   secretsByType,
		ExpiringSecrets: expiringSecrets,
	}, nil
}

// =============================================================================
// Tree Operations
// =============================================================================

// GetTree builds a tree structure of secrets
func (s *Service) GetTree(ctx context.Context, prefix string) ([]*core.SecretTreeNode, error) {
	appID, err := contexts.RequireAppID(ctx)
	if err != nil {
		return nil, core.ErrAppContextRequired()
	}

	envID, err := contexts.RequireEnvironmentID(ctx)
	if err != nil {
		return nil, core.ErrEnvironmentContextRequired()
	}

	// Get all secrets with optional prefix
	query := &core.ListSecretsQuery{
		Prefix:   prefix,
		PageSize: 1000, // Get all for tree view
	}

	secrets, _, err := s.repo.List(ctx, appID, envID, query)
	if err != nil {
		return nil, err
	}

	// Build tree
	return s.buildTree(secrets), nil
}

// buildTree builds a tree structure from a list of secrets
func (s *Service) buildTree(secrets []*schema.Secret) []*core.SecretTreeNode {
	// Create a map for quick lookups
	nodeMap := make(map[string]*core.SecretTreeNode)
	rootNodes := []*core.SecretTreeNode{}

	// First pass: create all nodes
	for _, secret := range secrets {
		dto := s.toDTO(secret)

		// Create secret node
		secretNode := &core.SecretTreeNode{
			Name:     secret.Key,
			Path:     secret.Path,
			IsSecret: true,
			Secret:   dto,
		}

		// Get parent path
		parentPath := core.GetParentPath(secret.Path)

		// Ensure all parent folders exist
		if parentPath != "" {
			s.ensureParentNodes(parentPath, nodeMap)
		}

		// Add to parent or root
		if parentPath == "" {
			rootNodes = append(rootNodes, secretNode)
		} else {
			if parent, ok := nodeMap[parentPath]; ok {
				parent.Children = append(parent.Children, secretNode)
			}
		}
	}

	// Add folder nodes to root if they have no parent
	for path, node := range nodeMap {
		if core.GetParentPath(path) == "" {
			rootNodes = append(rootNodes, node)
		}
	}

	// Sort nodes
	core.SortByPath(extractPaths(rootNodes))

	return rootNodes
}

// ensureParentNodes creates parent folder nodes
func (s *Service) ensureParentNodes(path string, nodeMap map[string]*core.SecretTreeNode) {
	parts := make([]string, 0)
	current := path

	// Collect all ancestors
	for current != "" {
		parts = append([]string{current}, parts...)
		current = core.GetParentPath(current)
	}

	// Create nodes for each level
	for _, part := range parts {
		if _, exists := nodeMap[part]; !exists {
			nodeMap[part] = &core.SecretTreeNode{
				Name:     core.GetKey(part),
				Path:     part,
				IsSecret: false,
				Children: []*core.SecretTreeNode{},
			}

			// Link to parent
			parentPath := core.GetParentPath(part)
			if parentPath != "" {
				if parent, ok := nodeMap[parentPath]; ok {
					parent.Children = append(parent.Children, nodeMap[part])
				}
			}
		}
	}
}

// extractPaths extracts paths from tree nodes
func extractPaths(nodes []*core.SecretTreeNode) []string {
	paths := make([]string, len(nodes))
	for i, n := range nodes {
		paths[i] = n.Path
	}
	return paths
}

// =============================================================================
// Helper Methods
// =============================================================================

// toDTO converts a schema.Secret to core.SecretDTO
func (s *Service) toDTO(secret *schema.Secret) *core.SecretDTO {
	return &core.SecretDTO{
		ID:          secret.ID.String(),
		Path:        secret.Path,
		Key:         secret.Key,
		ValueType:   string(secret.ValueType),
		Description: secret.Description,
		Tags:        secret.Tags,
		Metadata:    secret.Metadata,
		Version:     secret.Version,
		IsActive:    secret.IsActive,
		HasSchema:   secret.SchemaJSON != "",
		HasExpiry:   secret.ExpiresAt != nil,
		ExpiresAt:   secret.ExpiresAt,
		CreatedBy:   secret.CreatedBy.String(),
		UpdatedBy:   secret.UpdatedBy.String(),
		CreatedAt:   secret.CreatedAt,
		UpdatedAt:   secret.UpdatedAt,
	}
}

// logAccess logs an access event
func (s *Service) logAccess(ctx context.Context, secretID xid.ID, path, action string, success bool, errorMsg string) {
	if s.config != nil && !s.config.Audit.EnableAccessLog {
		return
	}

	appID, _ := contexts.GetAppID(ctx)
	envID, _ := contexts.GetEnvironmentID(ctx)
	userID, _ := contexts.GetUserID(ctx)

	log := &schema.SecretAccessLog{
		ID:            xid.New(),
		SecretID:      secretID,
		AppID:         appID,
		EnvironmentID: envID,
		Path:          path,
		Action:        action,
		AccessedBy:    userID,
		AccessMethod:  "api",
		Success:       success,
		ErrorMessage:  errorMsg,
		CreatedAt:     time.Now().UTC(),
	}

	_ = s.repo.LogAccess(ctx, log)
}

// isUniqueViolation checks if an error is a unique constraint violation
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return contains(errStr, "unique") || contains(errStr, "duplicate") || contains(errStr, "UNIQUE")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr, 0))
}

func containsAt(s, substr string, start int) bool {
	for i := start; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

