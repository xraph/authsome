package plugin_test

import (
	"context"
	"errors"
	"testing"

	log "github.com/xraph/go-utils/log"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugin"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/user"

	"github.com/xraph/grove/migrate"
)

// ──────────────────────────────────────────────────
// Mock plugins
// ──────────────────────────────────────────────────

type basePlugin struct{ name string }

func (p *basePlugin) Name() string { return p.name }

// fullPlugin implements multiple hook interfaces.
type fullPlugin struct {
	basePlugin
	beforeSignUpCalled        bool
	afterSignUpCalled         bool
	beforeSignInCalled        bool
	afterSignInCalled         bool
	beforeSignOutCalled       bool
	afterSignOutCalled        bool
	beforeUserCreateCalled    bool
	afterUserCreateCalled     bool
	beforeSessionCreateCalled bool
	afterSessionCreateCalled  bool
	err                       error // if set, hooks return this error
}

func (p *fullPlugin) OnBeforeSignUp(_ context.Context, _ *account.SignUpRequest) error {
	p.beforeSignUpCalled = true
	return p.err
}

func (p *fullPlugin) OnAfterSignUp(_ context.Context, _ *user.User, _ *session.Session) error {
	p.afterSignUpCalled = true
	return p.err
}

func (p *fullPlugin) OnBeforeSignIn(_ context.Context, _ *account.SignInRequest) error {
	p.beforeSignInCalled = true
	return p.err
}

func (p *fullPlugin) OnAfterSignIn(_ context.Context, _ *user.User, _ *session.Session) error {
	p.afterSignInCalled = true
	return p.err
}

func (p *fullPlugin) OnBeforeSignOut(_ context.Context, _ id.SessionID) error {
	p.beforeSignOutCalled = true
	return p.err
}

func (p *fullPlugin) OnAfterSignOut(_ context.Context, _ id.SessionID) error {
	p.afterSignOutCalled = true
	return p.err
}

func (p *fullPlugin) OnBeforeUserCreate(_ context.Context, _ *user.User) error {
	p.beforeUserCreateCalled = true
	return p.err
}

func (p *fullPlugin) OnAfterUserCreate(_ context.Context, _ *user.User) error {
	p.afterUserCreateCalled = true
	return p.err
}

func (p *fullPlugin) OnBeforeSessionCreate(_ context.Context, _ *session.Session) error {
	p.beforeSessionCreateCalled = true
	return p.err
}

func (p *fullPlugin) OnAfterSessionCreate(_ context.Context, _ *session.Session) error {
	p.afterSessionCreateCalled = true
	return p.err
}

// routePlugin implements RouteProvider.
type routePlugin struct {
	basePlugin
	registerCalled bool
}

func (p *routePlugin) RegisterRoutes(_ any) error {
	p.registerCalled = true
	return nil
}

// migrationPlugin implements MigrationProvider.
type migrationPlugin struct {
	basePlugin
	groups []*migrate.Group
}

func (p *migrationPlugin) MigrationGroups(_ string) []*migrate.Group {
	return p.groups
}

// ──────────────────────────────────────────────────
// Tests
// ──────────────────────────────────────────────────

func TestRegistry_Register(t *testing.T) {
	r := plugin.NewRegistry(log.NewNoopLogger())

	p1 := &fullPlugin{basePlugin: basePlugin{name: "test-plugin"}}
	r.Register(p1)

	assert.Len(t, r.Plugins(), 1)
	assert.Equal(t, "test-plugin", r.Plugins()[0].Name())
}

func TestRegistry_Register_Multiple(t *testing.T) {
	r := plugin.NewRegistry(log.NewNoopLogger())

	p1 := &basePlugin{name: "one"}
	p2 := &basePlugin{name: "two"}
	p3 := &basePlugin{name: "three"}

	r.Register(p1)
	r.Register(p2)
	r.Register(p3)

	assert.Len(t, r.Plugins(), 3)
}

func TestRegistry_TypeCaching_BeforeSignUp(t *testing.T) {
	r := plugin.NewRegistry(log.NewNoopLogger())
	ctx := context.Background()

	// Register a plugin that implements BeforeSignUp
	p := &fullPlugin{basePlugin: basePlugin{name: "signup-check"}}
	r.Register(p)

	// Also register one that doesn't implement BeforeSignUp
	r.Register(&basePlugin{name: "base-only"})

	req := &account.SignUpRequest{Email: "test@test.com"}
	err := r.EmitBeforeSignUp(ctx, req)

	require.NoError(t, err)
	assert.True(t, p.beforeSignUpCalled)
}

func TestRegistry_BeforeSignUp_AbortsOnError(t *testing.T) {
	r := plugin.NewRegistry(log.NewNoopLogger())
	ctx := context.Background()

	errBlocked := errors.New("signup blocked")
	p1 := &fullPlugin{basePlugin: basePlugin{name: "blocker"}, err: errBlocked}
	p2 := &fullPlugin{basePlugin: basePlugin{name: "after"}}

	r.Register(p1)
	r.Register(p2)

	err := r.EmitBeforeSignUp(ctx, &account.SignUpRequest{})
	assert.ErrorIs(t, err, errBlocked)

	// Second plugin's BeforeSignUp should NOT have been called
	assert.False(t, p2.beforeSignUpCalled)
}

func TestRegistry_AfterSignUp_LogsErrorButContinues(t *testing.T) {
	r := plugin.NewRegistry(log.NewNoopLogger())
	ctx := context.Background()

	p1 := &fullPlugin{basePlugin: basePlugin{name: "failing"}, err: errors.New("boom")}
	p2 := &fullPlugin{basePlugin: basePlugin{name: "succeeding"}}

	r.Register(p1)
	r.Register(p2)

	// AfterSignUp doesn't return errors — just logs them
	r.EmitAfterSignUp(ctx, &user.User{}, &session.Session{})

	assert.True(t, p1.afterSignUpCalled)
	assert.True(t, p2.afterSignUpCalled) // Should still be called
}

func TestRegistry_BeforeSignIn(t *testing.T) {
	r := plugin.NewRegistry(log.NewNoopLogger())
	ctx := context.Background()

	p := &fullPlugin{basePlugin: basePlugin{name: "signin"}}
	r.Register(p)

	err := r.EmitBeforeSignIn(ctx, &account.SignInRequest{})
	require.NoError(t, err)
	assert.True(t, p.beforeSignInCalled)
}

func TestRegistry_BeforeSignOut(t *testing.T) {
	r := plugin.NewRegistry(log.NewNoopLogger())
	ctx := context.Background()

	p := &fullPlugin{basePlugin: basePlugin{name: "signout"}}
	r.Register(p)

	sessID := id.NewSessionID()
	err := r.EmitBeforeSignOut(ctx, sessID)
	require.NoError(t, err)
	assert.True(t, p.beforeSignOutCalled)
}

func TestRegistry_BeforeUserCreate(t *testing.T) {
	r := plugin.NewRegistry(log.NewNoopLogger())
	ctx := context.Background()

	p := &fullPlugin{basePlugin: basePlugin{name: "user-create"}}
	r.Register(p)

	err := r.EmitBeforeUserCreate(ctx, &user.User{})
	require.NoError(t, err)
	assert.True(t, p.beforeUserCreateCalled)
}

func TestRegistry_BeforeSessionCreate(t *testing.T) {
	r := plugin.NewRegistry(log.NewNoopLogger())
	ctx := context.Background()

	p := &fullPlugin{basePlugin: basePlugin{name: "session-create"}}
	r.Register(p)

	err := r.EmitBeforeSessionCreate(ctx, &session.Session{})
	require.NoError(t, err)
	assert.True(t, p.beforeSessionCreateCalled)
}

func TestRegistry_RouteProviders(t *testing.T) {
	r := plugin.NewRegistry(log.NewNoopLogger())

	rp := &routePlugin{basePlugin: basePlugin{name: "route-plugin"}}
	r.Register(rp)

	// Also register non-route plugin
	r.Register(&basePlugin{name: "no-routes"})

	providers := r.RouteProviders()
	assert.Len(t, providers, 1)
}

func TestRegistry_CollectMigrationGroups(t *testing.T) {
	r := plugin.NewRegistry(log.NewNoopLogger())

	g := migrate.NewGroup("test-plugin", migrate.DependsOn("authsome"))
	mp := &migrationPlugin{
		basePlugin: basePlugin{name: "migration-plugin"},
		groups:     []*migrate.Group{g},
	}
	r.Register(mp)

	groups := r.CollectMigrationGroups("pg")
	require.Len(t, groups, 1)
	assert.Equal(t, "test-plugin", groups[0].Name())
}

func TestRegistry_CollectMigrationGroups_Empty(t *testing.T) {
	r := plugin.NewRegistry(log.NewNoopLogger())

	// Plugin with no migration groups
	mp := &migrationPlugin{
		basePlugin: basePlugin{name: "no-migrations"},
		groups:     nil,
	}
	r.Register(mp)

	groups := r.CollectMigrationGroups("pg")
	assert.Empty(t, groups)
}

func TestRegistry_EmptyRegistry(t *testing.T) {
	r := plugin.NewRegistry(log.NewNoopLogger())
	ctx := context.Background()

	// All emit calls should work fine on empty registry
	assert.NoError(t, r.EmitBeforeSignUp(ctx, &account.SignUpRequest{}))
	assert.NoError(t, r.EmitBeforeSignIn(ctx, &account.SignInRequest{}))
	assert.NoError(t, r.EmitBeforeSignOut(ctx, id.NewSessionID()))
	assert.NoError(t, r.EmitBeforeUserCreate(ctx, &user.User{}))
	assert.NoError(t, r.EmitBeforeSessionCreate(ctx, &session.Session{}))
	assert.Empty(t, r.CollectMigrationGroups("pg"))

	r.EmitAfterSignUp(ctx, &user.User{}, &session.Session{})
	r.EmitAfterSignIn(ctx, &user.User{}, &session.Session{})
	r.EmitAfterSignOut(ctx, id.NewSessionID())
	r.EmitAfterUserCreate(ctx, &user.User{})
	r.EmitAfterSessionCreate(ctx, &session.Session{})
	r.EmitAfterSessionRevoke(ctx, id.NewSessionID())

	assert.Empty(t, r.Plugins())
	assert.Empty(t, r.RouteProviders())
}

func TestRegistry_OnInitShutdown(_ *testing.T) {
	r := plugin.NewRegistry(log.NewNoopLogger())
	ctx := context.Background()

	// We can't easily test OnInit/OnShutdown with our mock approach since those
	// are separate interfaces. Instead just verify EmitOnInit/EmitOnShutdown
	// don't panic on an empty registry.
	r.EmitOnInit(ctx, nil)
	r.EmitOnShutdown(ctx)
}
