package settings

import "context"

// ResolveOpts provides the full scope context for cascade resolution.
type ResolveOpts struct {
	AppID  string
	OrgID  string
	UserID string
}

// ListOpts controls pagination and filtering for setting queries.
type ListOpts struct {
	Limit     int
	Offset    int
	Namespace string
	Scope     Scope
	ScopeID   string
	AppID     string
	OrgID     string
}

// Store defines the persistence contract for settings.
// Each backend (postgres, sqlite, mongo, memory) implements this.
type Store interface {
	// GetSetting retrieves a single setting by key + scope + scope_id.
	// Returns ErrNotFound if no setting exists for the given combination.
	GetSetting(ctx context.Context, key string, scope Scope, scopeID string) (*Setting, error)

	// SetSetting creates or updates a setting. Upsert semantics on (key, scope, scope_id).
	SetSetting(ctx context.Context, s *Setting) error

	// DeleteSetting removes a specific setting override.
	DeleteSetting(ctx context.Context, key string, scope Scope, scopeID string) error

	// ListSettings returns all settings matching the filter.
	ListSettings(ctx context.Context, opts ListOpts) ([]*Setting, error)

	// ResolveSettings performs cascade lookup for a key across all scopes.
	// Returns all matching settings ordered by specificity (global → app → org → user).
	// The Manager uses these to apply enforcement logic.
	ResolveSettings(ctx context.Context, key string, opts ResolveOpts) ([]*Setting, error)

	// BatchResolve resolves multiple keys at once for performance.
	// Returns a map of key → list of matching settings (ordered by specificity).
	BatchResolve(ctx context.Context, keys []string, opts ResolveOpts) (map[string][]*Setting, error)

	// DeleteSettingsByNamespace removes all settings for a given namespace.
	DeleteSettingsByNamespace(ctx context.Context, namespace string) error
}
