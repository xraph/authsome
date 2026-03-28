package authsome

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/apikey"
	"github.com/xraph/authsome/app"
	"github.com/xraph/authsome/environment"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/rbac"
	"github.com/xraph/authsome/store"
	"github.com/xraph/authsome/user"
)

// BootstrapConfig holds settings for automatic platform app setup.
type BootstrapConfig struct {
	// Platform app settings.
	AppName string // default: "Platform"
	AppSlug string // default: "platform"
	AppLogo string // optional logo URL

	// Default environments to create for every new app.
	Environments []BootstrapEnv

	// Whether to skip creating default environments.
	SkipDefaultEnvs bool

	// Default roles to create for every new app.
	DefaultRoles []BootstrapRole

	// Whether to skip creating default roles.
	SkipDefaultRoles bool

	// Callback for custom post-bootstrap logic.
	OnBootstrap func(ctx context.Context, e *Engine, appID id.AppID) error
}

// BootstrapEnv describes an environment to auto-create.
type BootstrapEnv struct {
	Name      string
	Slug      string
	Type      environment.Type
	IsDefault bool
}

// BootstrapPermission describes a permission to attach to a bootstrap role.
type BootstrapPermission struct {
	Action   string
	Resource string
}

// BootstrapRole describes a role to auto-create.
type BootstrapRole struct {
	Name        string
	Slug        string
	Description string
	ParentSlug  string // slug of parent role (empty = root)
	Permissions []BootstrapPermission
}

// BootstrapOption configures the bootstrap system.
type BootstrapOption func(*BootstrapConfig)

// DefaultBootstrapConfig returns sensible defaults.
func DefaultBootstrapConfig() *BootstrapConfig {
	return &BootstrapConfig{
		AppName: "Platform",
		AppSlug: "platform",
		Environments: []BootstrapEnv{
			{Name: "Development", Slug: "development", Type: environment.TypeDevelopment, IsDefault: true},
			{Name: "Staging", Slug: "staging", Type: environment.TypeStaging},
			{Name: "Production", Slug: "production", Type: environment.TypeProduction},
		},
		DefaultRoles: []BootstrapRole{
			// Platform-scoped roles (cross-app access).
			// Hierarchy: platform_user ← platform_admin ← platform_owner
			{
				Name:        "Platform User",
				Slug:        rbac.PlatformUserSlug,
				Description: "Standard cross-app access",
				Permissions: []BootstrapPermission{
					{Action: "read", Resource: "user"},
					{Action: "read", Resource: "session"},
					{Action: "read", Resource: "device"},
					{Action: "read", Resource: "environment"},
					{Action: "create", Resource: "session"},
					{Action: "delete", Resource: "session"},
					{Action: "create", Resource: "device"},
					{Action: "update", Resource: "device"},
				},
			},
			{
				Name:        "Platform Admin",
				Slug:        rbac.PlatformAdminSlug,
				Description: "Admin-level cross-app access",
				ParentSlug:  rbac.PlatformUserSlug,
				Permissions: []BootstrapPermission{
					{Action: "*", Resource: "*"},
				},
			},
			{
				Name:        "Platform Owner",
				Slug:        rbac.PlatformOwnerSlug,
				Description: "Full cross-app platform access",
				ParentSlug:  rbac.PlatformAdminSlug,
				Permissions: []BootstrapPermission{
					{Action: "*", Resource: "*"},
				},
			},
			// App-scoped roles.
			// Hierarchy: user ← admin ← owner
			{
				Name:        "User",
				Slug:        rbac.AppUserSlug,
				Description: "Standard user access",
				Permissions: []BootstrapPermission{
					{Action: "read", Resource: "user"},
					{Action: "read", Resource: "session"},
					{Action: "read", Resource: "device"},
					{Action: "read", Resource: "environment"},
					{Action: "create", Resource: "session"},
					{Action: "delete", Resource: "session"},
					{Action: "create", Resource: "device"},
					{Action: "update", Resource: "device"},
				},
			},
			{
				Name:        "Admin",
				Slug:        rbac.AppAdminSlug,
				Description: "Full app administration",
				ParentSlug:  rbac.AppUserSlug,
				Permissions: []BootstrapPermission{
					{Action: "*", Resource: "*"},
				},
			},
			{
				Name:        "Owner",
				Slug:        rbac.AppOwnerSlug,
				Description: "Full app ownership",
				ParentSlug:  rbac.AppAdminSlug,
				Permissions: []BootstrapPermission{
					{Action: "*", Resource: "*"},
				},
			},
		},
	}
}

// WithBootstrapAppName sets the platform app name.
func WithBootstrapAppName(name string) BootstrapOption {
	return func(cfg *BootstrapConfig) { cfg.AppName = name }
}

// WithBootstrapAppSlug sets the platform app slug.
func WithBootstrapAppSlug(slug string) BootstrapOption {
	return func(cfg *BootstrapConfig) { cfg.AppSlug = slug }
}

// WithBootstrapAppLogo sets the platform app logo URL.
func WithBootstrapAppLogo(logo string) BootstrapOption {
	return func(cfg *BootstrapConfig) { cfg.AppLogo = logo }
}

// WithBootstrapEnvs overrides the default environments to create.
func WithBootstrapEnvs(envs []BootstrapEnv) BootstrapOption {
	return func(cfg *BootstrapConfig) { cfg.Environments = envs }
}

// WithSkipDefaultEnvs disables automatic environment creation.
func WithSkipDefaultEnvs() BootstrapOption {
	return func(cfg *BootstrapConfig) { cfg.SkipDefaultEnvs = true }
}

// WithBootstrapRoles overrides the default roles to create.
func WithBootstrapRoles(roles []BootstrapRole) BootstrapOption {
	return func(cfg *BootstrapConfig) { cfg.DefaultRoles = roles }
}

// WithSkipDefaultRoles disables automatic role creation.
func WithSkipDefaultRoles() BootstrapOption {
	return func(cfg *BootstrapConfig) { cfg.SkipDefaultRoles = true }
}

// WithOnBootstrap sets a callback invoked after the platform app is created.
func WithOnBootstrap(fn func(ctx context.Context, e *Engine, appID id.AppID) error) BootstrapOption {
	return func(cfg *BootstrapConfig) { cfg.OnBootstrap = fn }
}

// bootstrap creates the platform app and its default resources if it doesn't exist.
func (e *Engine) bootstrap(ctx context.Context) error {
	// 1. Try to find existing platform app by slug.
	existing, err := e.store.GetAppBySlug(ctx, e.bootstrapCfg.AppSlug)
	if err == nil && existing != nil {
		e.platformAppID = existing.ID
		if e.config.AppID == "" {
			e.config.AppID = existing.ID.String()
		}
		// Still run bootstrapApp to ensure roles/envs exist (idempotent).
		if setupErr := e.bootstrapApp(ctx, existing.ID); setupErr != nil {
			e.logger.Warn("bootstrap: setup existing platform app",
				log.String("error", setupErr.Error()),
			)
		}
		return nil
	}
	// If the error is anything other than "not found", surface it.
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		return fmt.Errorf("bootstrap: lookup platform app: %w", err)
	}

	// 2. Even if the slug doesn't match, check if ANY platform app already
	//    exists. There must only be one platform app per database. If one
	//    exists with a different slug, adopt it instead of creating a second.
	if platformApp, findErr := e.store.GetPlatformApp(ctx); findErr == nil && platformApp != nil {
		e.platformAppID = platformApp.ID
		if e.config.AppID == "" {
			e.config.AppID = platformApp.ID.String()
		}
		e.logger.Info("bootstrap: adopted existing platform app (different slug)",
			log.String("app_id", platformApp.ID.String()),
			log.String("existing_slug", platformApp.Slug),
			log.String("requested_slug", e.bootstrapCfg.AppSlug),
		)
		if err := e.bootstrapApp(ctx, platformApp.ID); err != nil {
			e.logger.Warn("bootstrap: setup existing platform app",
				log.String("error", err.Error()),
			)
		}
		return nil
	}

	// 3. No platform app exists at all — create one.
	now := time.Now()
	platformApp := &app.App{
		ID:         id.NewAppID(),
		Name:       e.bootstrapCfg.AppName,
		Slug:       e.bootstrapCfg.AppSlug,
		Logo:       e.bootstrapCfg.AppLogo,
		IsPlatform: true,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	// Generate publishable key for the platform app.
	b := make([]byte, 32)
	if _, err := rand.Read(b); err == nil {
		platformApp.PublishableKey = apikey.PublicKeyMarker(environment.TypeProduction) + hex.EncodeToString(b)
	}

	if err := e.store.CreateApp(ctx, platformApp); err != nil {
		return fmt.Errorf("bootstrap: create platform app: %w", err)
	}
	e.platformAppID = platformApp.ID
	if e.config.AppID == "" {
		e.config.AppID = platformApp.ID.String()
	}

	// Run shared app bootstrap (envs, roles).
	if err := e.bootstrapApp(ctx, platformApp.ID); err != nil {
		return fmt.Errorf("bootstrap: setup platform app: %w", err)
	}

	// Custom callback.
	if e.bootstrapCfg.OnBootstrap != nil {
		if err := e.bootstrapCfg.OnBootstrap(ctx, e, platformApp.ID); err != nil {
			e.logger.Warn("bootstrap: custom callback failed",
				log.String("error", err.Error()),
			)
		}
	}

	e.logger.Info("bootstrap: platform app created",
		log.String("app_id", platformApp.ID.String()),
		log.String("slug", platformApp.Slug),
	)
	return nil
}

// bootstrapApp runs shared bootstrap logic for any app (envs, roles).
// Called both during platform bootstrap and when any new app is created.
func (e *Engine) bootstrapApp(ctx context.Context, appID id.AppID) error { //nolint:unparam // error return reserved for future use
	// Create default environments.
	if !e.bootstrapCfg.SkipDefaultEnvs {
		for _, env := range e.bootstrapCfg.Environments {
			if err := e.CreateEnvironment(ctx, &environment.Environment{
				AppID:     appID,
				Name:      env.Name,
				Slug:      env.Slug,
				Type:      env.Type,
				IsDefault: env.IsDefault,
			}); err != nil {
				// Ignore duplicate errors — idempotent.
				e.logger.Debug("bootstrap: environment may already exist",
					log.String("slug", env.Slug),
					log.String("error", err.Error()),
				)
			}
		}
	}

	// Create default roles with permissions (if an RBAC store is available).
	if e.hasRBACStore() && !e.bootstrapCfg.SkipDefaultRoles {
		// Pass 1: Create or fetch roles and attach permissions, collecting slug → ID map.
		slugToID := make(map[string]string, len(e.bootstrapCfg.DefaultRoles))
		for _, role := range e.bootstrapCfg.DefaultRoles {
			// Check if the role already exists (idempotent restart).
			existing, _ := e.GetRoleBySlug(ctx, appID, role.Slug) //nolint:errcheck // not-found is expected
			var r *rbac.Role
			if existing != nil {
				r = existing
				e.logger.Debug("bootstrap: role already exists, skipping create",
					log.String("slug", role.Slug),
				)
			} else {
				r = &rbac.Role{
					AppID:       appID.String(),
					Name:        role.Name,
					Slug:        role.Slug,
					Description: role.Description,
				}
				if err := e.CreateRole(ctx, r); err != nil {
					e.logger.Warn("bootstrap: role creation failed",
						log.String("slug", role.Slug),
						log.String("error", err.Error()),
					)
					continue
				}
			}
			slugToID[role.Slug] = r.ID

			// Attach permissions to the role (AddPermission is already
			// idempotent — duplicate permissions are looked up and reused).
			for _, perm := range role.Permissions {
				if err := e.AddPermission(ctx, &rbac.Permission{
					RoleID:   r.ID,
					Action:   perm.Action,
					Resource: perm.Resource,
				}); err != nil {
					e.logger.Debug("bootstrap: permission already exists",
						log.String("role", role.Slug),
						log.String("perm", perm.Action+":"+perm.Resource),
						log.String("error", err.Error()),
					)
				}
			}
		}

		// Pass 2: Wire up role hierarchy via ParentID.
		for _, role := range e.bootstrapCfg.DefaultRoles {
			if role.ParentSlug == "" {
				continue
			}
			childID, ok := slugToID[role.Slug]
			if !ok {
				continue
			}
			parentID, ok := slugToID[role.ParentSlug]
			if !ok {
				continue
			}
			if err := e.updateRoleParent(ctx, childID, parentID); err != nil {
				e.logger.Debug("bootstrap: failed to set role parent",
					log.String("child", role.Slug),
					log.String("parent", role.ParentSlug),
					log.String("error", err.Error()),
				)
			}
		}
	}

	return nil
}

// HasUsers returns true if the platform app has at least one user.
func (e *Engine) HasUsers(ctx context.Context) bool {
	if !e.started {
		return false
	}
	appID := e.PlatformAppID()
	if appID.IsNil() {
		return false
	}
	list, err := e.store.ListUsers(ctx, &user.Query{
		AppID: appID,
		Limit: 1,
	})
	if err != nil {
		return false
	}
	return list.Total > 0
}

// PlatformAppID returns the platform app's ID.
func (e *Engine) PlatformAppID() id.AppID {
	return e.platformAppID
}

// updateRoleParent sets the parent of a role by updating it via the RBAC store.
func (e *Engine) updateRoleParent(ctx context.Context, roleID, parentID string) error {
	role, err := e.rbacStore().GetRole(ctx, roleID)
	if err != nil {
		return err
	}
	role.ParentID = parentID
	return e.rbacStore().UpdateRole(ctx, role)
}
