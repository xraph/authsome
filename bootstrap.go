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
	wardenseed "github.com/xraph/authsome/bootstrap/warden"
	"github.com/xraph/authsome/environment"
	"github.com/xraph/authsome/id"
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

	// Whether to skip applying the default warden DSL roles + permission catalog.
	SkipDefaultRoles bool

	// SeedOptions configures the warden DSL seed apply path. Use this to
	// override the embedded .warden files (e.g. with a custom directory
	// shipped via cfg.WardenDir, or a custom fs.FS).
	SeedOptions wardenseed.ApplyOptions

	// InitialOwners is a list of email addresses that should automatically
	// receive the platform-owner role when they sign up, regardless of whether
	// they are the first user. Comparison is case-insensitive.
	InitialOwners []string

	// InitialOwnerCount controls how many of the first N users to sign up on
	// the platform app are automatically promoted to platform-owner. Defaults
	// to 3. Set to 1 for the original single-owner behaviour; set to 0 to
	// disable the count-based promotion entirely (only InitialOwners emails
	// will be used).
	InitialOwnerCount int

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

// BootstrapOption configures the bootstrap system.
type BootstrapOption func(*BootstrapConfig)

// DefaultBootstrapConfig returns sensible defaults. Roles + permissions
// are sourced from the embedded warden DSL files (see bootstrap/warden/embed),
// not from this struct.
func DefaultBootstrapConfig() *BootstrapConfig {
	return &BootstrapConfig{
		AppName:           "Platform",
		AppSlug:           "platform",
		InitialOwnerCount: 3,
		Environments: []BootstrapEnv{
			{Name: "Development", Slug: "development", Type: environment.TypeDevelopment, IsDefault: true},
			{Name: "Staging", Slug: "staging", Type: environment.TypeStaging},
			{Name: "Production", Slug: "production", Type: environment.TypeProduction},
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

// WithSkipDefaultRoles disables automatic role + permission seeding via
// the warden DSL.
func WithSkipDefaultRoles() BootstrapOption {
	return func(cfg *BootstrapConfig) { cfg.SkipDefaultRoles = true }
}

// WithBootstrapWardenDir tells bootstrap to load .warden files from the
// given directory instead of the embedded defaults. The directory must
// contain a `shared/` subtree (applied to every app); a `platform/`
// subtree is consulted only for the platform app and is optional.
func WithBootstrapWardenDir(dir string) BootstrapOption {
	return func(cfg *BootstrapConfig) {
		shared := wardenseed.Source{Dir: dir + "/shared"}
		platform := wardenseed.Source{Dir: dir + "/platform"}
		cfg.SeedOptions.SharedOverride = &shared
		cfg.SeedOptions.PlatformOverride = &platform
	}
}

// WithBootstrapWardenSources lets advanced callers replace the embedded
// .warden sources with custom ones (e.g. their own fs.FS or directory).
// Either argument may be nil to fall back to the embedded default for
// that scope.
func WithBootstrapWardenSources(shared, platform *wardenseed.Source) BootstrapOption {
	return func(cfg *BootstrapConfig) {
		cfg.SeedOptions.SharedOverride = shared
		cfg.SeedOptions.PlatformOverride = platform
	}
}

// WithInitialOwners registers email addresses that should receive the
// platform-owner role on sign-up, regardless of whether they are the first
// user. Comparison is case-insensitive. The option is additive — calling it
// multiple times appends to the existing list.
func WithInitialOwners(emails ...string) BootstrapOption {
	return func(cfg *BootstrapConfig) { cfg.InitialOwners = append(cfg.InitialOwners, emails...) }
}

// WithInitialOwnerCount sets how many of the first N users to register on the
// platform app are automatically promoted to platform-owner. The default is 3.
// Pass 1 to restore the original single-owner behaviour; pass 0 to disable
// count-based promotion entirely.
func WithInitialOwnerCount(n int) BootstrapOption {
	return func(cfg *BootstrapConfig) { cfg.InitialOwnerCount = n }
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

// bootstrapApp runs shared bootstrap logic for any app: default
// environments + warden DSL role/permission seeding.
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

	// Apply default roles + permission catalog from the embedded warden DSL.
	// Warden is required for RBAC, so when it isn't wired we have nothing
	// to apply against — silently skip and let the missing-warden path
	// elsewhere (rbacStore) surface that as a panic when something tries to
	// touch RBAC at runtime.
	if e.bootstrapCfg.SkipDefaultRoles || e.wardenEng == nil {
		return nil
	}

	isPlatform := appID == e.platformAppID
	res, err := wardenseed.ApplyForApp(ctx, e.wardenEng, appID, isPlatform, e.bootstrapCfg.SeedOptions)
	if err != nil {
		e.logger.Warn("bootstrap: warden DSL apply failed",
			log.String("app_id", appID.String()),
			log.String("error", err.Error()),
		)
		return nil
	}
	if res != nil && (len(res.Created) > 0 || len(res.Updated) > 0 || len(res.Deleted) > 0) {
		e.logger.Info("bootstrap: warden DSL applied",
			log.String("app_id", appID.String()),
			log.Int("created", len(res.Created)),
			log.Int("updated", len(res.Updated)),
			log.Int("deleted", len(res.Deleted)),
			log.Int("noops", res.NoOps),
		)
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
