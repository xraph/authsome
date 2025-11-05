package jwt

import (
	"context"
	"fmt"

	"github.com/rs/xid"
	"github.com/xraph/authsome/repository"
	"github.com/xraph/authsome/schema"
)

// repositoryWrapper wraps the schema-based repository and converts types
type repositoryWrapper struct {
	repo *repository.JWTKeyRepository
}

// NewRepositoryWrapper creates a new repository wrapper
func NewRepositoryWrapper(repo *repository.JWTKeyRepository) Repository {
	return &repositoryWrapper{repo: repo}
}

// Create creates a new JWT key
func (w *repositoryWrapper) Create(ctx context.Context, key *JWTKey) error {
	schemaKey := w.coreToSchema(key)
	return w.repo.Create(ctx, schemaKey)
}

// FindByID finds a JWT key by ID
func (w *repositoryWrapper) FindByID(ctx context.Context, id string) (*JWTKey, error) {
	schemaKey, err := w.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return w.schemaToCore(schemaKey), nil
}

// FindByKeyID finds a JWT key by key ID and organization
func (w *repositoryWrapper) FindByKeyID(ctx context.Context, keyID, orgID string) (*JWTKey, error) {
	// The schema repository doesn't have this method with orgID, so we need to implement it
	// For now, let's use the existing method and filter by orgID in the service
	schemaKey, err := w.repo.FindByKeyID(ctx, keyID)
	if err != nil {
		return nil, err
	}

	// Check if the key belongs to the correct organization
	if schemaKey.OrgID != orgID {
		return nil, fmt.Errorf("key not found for organization")
	}

	return w.schemaToCore(schemaKey), nil
}

// FindByOrgID finds all JWT keys for an organization
func (w *repositoryWrapper) FindByOrgID(ctx context.Context, orgID string, active *bool, offset, limit int) ([]*JWTKey, int64, error) {
	schemaKeys, err := w.repo.FindByOrgID(ctx, orgID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	// Filter by active status if specified
	var filteredKeys []*schema.JWTKey
	for _, key := range schemaKeys {
		if active == nil || key.Active == *active {
			filteredKeys = append(filteredKeys, key)
		}
	}

	// Convert to core types
	coreKeys := make([]*JWTKey, len(filteredKeys))
	for i, key := range filteredKeys {
		coreKeys[i] = w.schemaToCore(key)
	}

	// Get total count
	total, err := w.repo.CountByOrgID(ctx, orgID)
	if err != nil {
		return nil, 0, err
	}

	return coreKeys, int64(total), nil
}

// Update updates a JWT key
func (w *repositoryWrapper) Update(ctx context.Context, key *JWTKey) error {
	schemaKey := w.coreToSchema(key)
	return w.repo.Update(ctx, schemaKey)
}

// UpdateUsage updates the usage statistics for a JWT key
func (w *repositoryWrapper) UpdateUsage(ctx context.Context, keyID string) error {
	return w.repo.UpdateUsage(ctx, keyID)
}

// Deactivate deactivates a JWT key
func (w *repositoryWrapper) Deactivate(ctx context.Context, id string) error {
	return w.repo.Deactivate(ctx, id)
}

// Delete soft deletes a JWT key
func (w *repositoryWrapper) Delete(ctx context.Context, id string) error {
	return w.repo.Delete(ctx, id)
}

// CleanupExpired removes expired JWT keys
func (w *repositoryWrapper) CleanupExpired(ctx context.Context) (int64, error) {
	count, err := w.repo.CleanupExpired(ctx)
	return int64(count), err
}

// coreToSchema converts a core JWTKey to a schema JWTKey
func (w *repositoryWrapper) coreToSchema(core *JWTKey) *schema.JWTKey {
	id, _ := xid.FromString(core.ID.String())

	// Convert metadata
	metadata := make(map[string]string)
	for k, v := range core.Metadata {
		if str, ok := v.(string); ok {
			metadata[k] = str
		}
	}

	return &schema.JWTKey{
		ID:         id,
		OrgID:      core.OrgID,
		KeyID:      core.KeyID,
		Algorithm:  core.Algorithm,
		KeyType:    core.KeyType,
		Curve:      core.Curve,
		PrivateKey: []byte(core.PrivateKey),
		PublicKey:  []byte(core.PublicKey),
		Active:     core.IsActive,
		UsageCount: core.UsageCount,
		LastUsedAt: core.LastUsedAt,
		CreatedAt:  core.CreatedAt,
		UpdatedAt:  core.UpdatedAt,
		ExpiresAt:  core.ExpiresAt,
		Metadata:   metadata,
	}
}

// schemaToCore converts a schema JWTKey to a core JWTKey
func (w *repositoryWrapper) schemaToCore(schema *schema.JWTKey) *JWTKey {
	// Convert metadata
	metadata := make(map[string]interface{})
	for k, v := range schema.Metadata {
		metadata[k] = v
	}

	return &JWTKey{
		ID:         schema.ID,
		OrgID:      schema.OrgID,
		KeyID:      schema.KeyID,
		Algorithm:  schema.Algorithm,
		KeyType:    schema.KeyType,
		Curve:      schema.Curve,
		PrivateKey: string(schema.PrivateKey),
		PublicKey:  string(schema.PublicKey),
		IsActive:   schema.Active,
		UsageCount: schema.UsageCount,
		LastUsedAt: schema.LastUsedAt,
		CreatedAt:  schema.CreatedAt,
		UpdatedAt:  schema.UpdatedAt,
		ExpiresAt:  schema.ExpiresAt,
		Metadata:   metadata,
	}
}
