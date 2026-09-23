package contract

import (
	"context"
	"errors"
	"fmt"
	"net/http/httptest"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/app"
	"github.com/xraph/authsome/environment"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/secutil"
	"github.com/xraph/authsome/rbac"
	"github.com/xraph/authsome/store"
	"github.com/xraph/authsome/store/memory"
	"github.com/xraph/authsome/user"

	dashcontract "github.com/xraph/forge/extensions/dashboard/contract"
	"github.com/xraph/warden"
	wardenmem "github.com/xraph/warden/store/memory"
	"golang.org/x/crypto/bcrypt"
)

type failingUserListStore struct {
	store.Store
	fail atomic.Bool
}

func (s *failingUserListStore) ListUsers(ctx context.Context, q *user.Query) (*user.List, error) {
	if s.fail.Load() {
		return nil, fmt.Errorf("forced user-list failure")
	}
	return s.Store.ListUsers(ctx, q)
}

func newSetupEngine(t *testing.T) *authsome.Engine {
	t.Helper()
	cfg := authsome.DefaultConfig()
	cfg.Password.BcryptCost = bcrypt.MinCost
	return secutil.NewTestEngine(t,
		authsome.WithConfig(cfg),
		authsome.WithBootstrap(),
	)
}

func newSetupEngineWithFailingUserList(t *testing.T) (*authsome.Engine, *failingUserListStore) {
	t.Helper()
	wrapped := &failingUserListStore{Store: memory.New()}
	w, err := warden.NewEngine(warden.WithStore(wardenmem.New()))
	require.NoError(t, err)
	cfg := authsome.DefaultConfig()
	cfg.Password.BcryptCost = bcrypt.MinCost
	eng, err := authsome.NewEngine(
		authsome.WithStore(wrapped),
		authsome.WithWarden(w),
		authsome.WithDisableMigrate(),
		authsome.WithConfig(cfg),
		authsome.WithBootstrap(),
	)
	require.NoError(t, err)
	require.NoError(t, eng.Start(context.Background()))
	t.Cleanup(func() { _ = eng.Stop(context.Background()) })
	secutil.RelaxAuthDefaults(t, eng)
	return eng, wrapped
}

func TestSetupStatusReturnsSafeBootstrapDefaults(t *testing.T) {
	eng := newSetupEngine(t)
	got, err := setupStatusHandler(Deps{Engine: eng})(
		context.Background(), struct{}{}, dashcontract.Principal{},
	)
	require.NoError(t, err)
	require.True(t, got.Pending)
	require.NotNil(t, got.Platform)
	require.Equal(t, "Platform", got.Platform.Name)
	require.Equal(t, "platform", got.Platform.Slug)
	require.NotNil(t, got.Environment)
	require.True(t, got.Environment.IsDefault)
	require.Equal(t, "development", got.Environment.Type)
}

func TestSetupStatusDefaultsCannotExposePrivateFields(t *testing.T) {
	for _, tc := range []struct {
		name string
		typ  reflect.Type
	}{
		{name: "platform", typ: reflect.TypeOf(SetupPlatformDefaults{})},
		{name: "environment", typ: reflect.TypeOf(SetupEnvironmentDefaults{})},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, field := range []string{"Metadata", "PublishableKey", "Settings", "Credentials"} {
				_, found := tc.typ.FieldByName(field)
				require.False(t, found, "%s response must not expose %s", tc.name, field)
			}
		})
	}
}

func TestSetupStatusUnavailableWithoutEngine(t *testing.T) {
	_, err := setupStatusHandler(Deps{})(
		context.Background(), struct{}{}, dashcontract.Principal{},
	)
	var contractErr *dashcontract.Error
	require.True(t, errors.As(err, &contractErr))
	require.Equal(t, dashcontract.CodeUnavailable, contractErr.Code)
}

func TestSetupStatusFailsClosedWhenUserCountFails(t *testing.T) {
	eng, wrapped := newSetupEngineWithFailingUserList(t)
	wrapped.fail.Store(true)

	got, err := setupStatusHandler(Deps{Engine: eng})(
		context.Background(), struct{}{}, dashcontract.Principal{},
	)
	require.NoError(t, err)
	require.False(t, got.Pending)
	require.Nil(t, got.Platform)
	require.Nil(t, got.Environment)
}

func TestSetupStatusOmitsDefaultsAfterFirstUser(t *testing.T) {
	eng := newSetupEngine(t)
	_, _, err := eng.SignUp(context.Background(), &account.SignUpRequest{
		AppID:    eng.PlatformAppID(),
		Email:    "owner@example.com",
		Password: "SecureP@ss1",
	})
	require.NoError(t, err)

	got, err := setupStatusHandler(Deps{Engine: eng})(
		context.Background(), struct{}{}, dashcontract.Principal{},
	)
	require.NoError(t, err)
	require.Equal(t, SetupStatusResponse{Pending: false}, got)
}

func completeSetupInput() SetupInput {
	return SetupInput{
		Email:    "owner@example.com",
		Password: "SecureP@ss1",
		Name:     "Ada Lovelace",
		Platform: &SetupPlatformInput{
			Name:     "TwinOS Office",
			Slug:     "twinos-office",
			Logo:     "https://example.test/forge.svg",
			Metadata: map[string]string{"region": "us-central"},
		},
		Environment: &SetupEnvironmentInput{
			Name:        "Local Development",
			Slug:        "local-development",
			Type:        "development",
			Color:       "#2563eb",
			Description: "Local plugin development",
			Metadata:    map[string]string{"purpose": "plugins"},
		},
	}
}

func TestSetupHandlerAppliesPlatformEnvironmentAndOwner(t *testing.T) {
	eng := newSetupEngine(t)
	ctx := context.Background()

	platform, err := eng.GetApp(ctx, eng.PlatformAppID())
	require.NoError(t, err)
	platform.Metadata = app.Metadata{"existing": "kept"}
	require.NoError(t, eng.UpdateApp(ctx, platform))
	defaultEnv, err := eng.GetDefaultEnvironment(ctx, eng.PlatformAppID())
	require.NoError(t, err)
	defaultEnv.Metadata = environment.Metadata{"seeded": "kept"}
	require.NoError(t, eng.UpdateEnvironment(ctx, defaultEnv))

	h := setupHandler(Deps{Engine: eng})
	httpCtx, writer, _ := withHTTPCtx(t)
	got, err := h(httpCtx, completeSetupInput(), dashcontract.Principal{})
	require.NoError(t, err)
	require.True(t, got.OK)
	require.NotEmpty(t, got.Subject)

	updatedApp, err := eng.GetApp(ctx, eng.PlatformAppID())
	require.NoError(t, err)
	require.Equal(t, "TwinOS Office", updatedApp.Name)
	require.Equal(t, "twinos-office", updatedApp.Slug)
	require.Equal(t, "https://example.test/forge.svg", updatedApp.Logo)
	require.Equal(t, "kept", updatedApp.Metadata["existing"])
	require.Equal(t, "us-central", updatedApp.Metadata["region"])

	updatedEnv, err := eng.GetDefaultEnvironment(ctx, eng.PlatformAppID())
	require.NoError(t, err)
	require.Equal(t, defaultEnv.ID, updatedEnv.ID)
	require.Equal(t, "Local Development", updatedEnv.Name)
	require.Equal(t, "local-development", updatedEnv.Slug)
	require.Equal(t, environment.TypeDevelopment, updatedEnv.Type)
	require.Equal(t, "#2563eb", updatedEnv.Color)
	require.Equal(t, "Local plugin development", updatedEnv.Description)
	require.Equal(t, "kept", updatedEnv.Metadata["seeded"])
	require.Equal(t, "plugins", updatedEnv.Metadata["purpose"])

	cookies := writer.(*httptest.ResponseRecorder).Result().Cookies()
	require.Condition(t, func() bool {
		for _, cookie := range cookies {
			if cookie.Name == dashboardCookieName && cookie.Value != "" {
				return true
			}
		}
		return false
	}, "setup must write the dashboard session cookie")

	userID, err := id.ParseUserID(got.Subject)
	require.NoError(t, err)
	roles, err := eng.ListUserRoles(ctx, userID)
	require.NoError(t, err)
	require.Condition(t, func() bool {
		for _, role := range roles {
			if role.Slug == rbac.PlatformOwnerSlug {
				return true
			}
		}
		return false
	}, "first setup user must receive platform-owner")

	httpCtx, _, _ = withHTTPCtx(t)
	_, err = h(httpCtx, completeSetupInput(), dashcontract.Principal{})
	var contractErr *dashcontract.Error
	require.True(t, errors.As(err, &contractErr))
	require.Equal(t, dashcontract.CodePermissionDenied, contractErr.Code)
}

func TestSetupHandlerLegacyPayloadPreservesBootstrapConfiguration(t *testing.T) {
	eng := newSetupEngine(t)
	ctx := context.Background()
	beforeApp, err := eng.GetApp(ctx, eng.PlatformAppID())
	require.NoError(t, err)
	beforeEnv, err := eng.GetDefaultEnvironment(ctx, eng.PlatformAppID())
	require.NoError(t, err)

	httpCtx, _, _ := withHTTPCtx(t)
	got, err := setupHandler(Deps{Engine: eng})(httpCtx, SetupInput{
		Email: "legacy@example.com", Password: "SecureP@ss1",
	}, dashcontract.Principal{})
	require.NoError(t, err)
	require.True(t, got.OK)

	afterApp, err := eng.GetApp(ctx, eng.PlatformAppID())
	require.NoError(t, err)
	afterEnv, err := eng.GetDefaultEnvironment(ctx, eng.PlatformAppID())
	require.NoError(t, err)
	require.Equal(t, beforeApp.Name, afterApp.Name)
	require.Equal(t, beforeApp.Slug, afterApp.Slug)
	require.Equal(t, beforeEnv.ID, afterEnv.ID)
	require.Equal(t, beforeEnv.Name, afterEnv.Name)
	require.Equal(t, beforeEnv.Slug, afterEnv.Slug)
}
