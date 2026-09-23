package contract

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/internal/secutil"
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
