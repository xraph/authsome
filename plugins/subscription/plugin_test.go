package subscription

import (
	"context"
	"encoding/json"
	"testing"

	log "github.com/xraph/go-utils/log"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/forge"
	"github.com/xraph/ledger"
	"github.com/xraph/ledger/plan"
	"github.com/xraph/ledger/store/memory"
	"github.com/xraph/ledger/subscription"
	"github.com/xraph/warden"
	wardenmem "github.com/xraph/warden/store/memory"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/organization"
	"github.com/xraph/authsome/settings"
	authmemory "github.com/xraph/authsome/store/memory"
	"github.com/xraph/authsome/user"
)

// modeStore is a settings store that pins subscription.tenant_mode and lets
// every other key fall through to its registered default, so
// auto_subscribe_org stays true the way a fresh install has it.
type modeStore struct {
	settings.NilStore
	mode string
}

func (s modeStore) ResolveSettings(_ context.Context, key string, _ settings.ResolveOpts) ([]*settings.Setting, error) {
	if key != SettingTenantMode.Def.Key {
		return nil, nil
	}
	raw, err := json.Marshal(s.mode)
	if err != nil {
		return nil, err
	}
	return []*settings.Setting{{Key: key, Value: raw, Scope: settings.ScopeGlobal}}, nil
}

func (s modeStore) BatchResolve(ctx context.Context, keys []string, opts settings.ResolveOpts) (map[string][]*settings.Setting, error) {
	out := make(map[string][]*settings.Setting, len(keys))
	for _, k := range keys {
		rows, err := s.ResolveSettings(ctx, k, opts)
		if err != nil {
			return nil, err
		}
		out[k] = rows
	}
	return out, nil
}

// newHookPlugin returns a plugin on an in-memory ledger that holds a "free"
// plan for appID, with tenant mode pinned to mode and "free" as the default.
func newHookPlugin(t *testing.T, cfg Config, mode string, appID id.AppID) (*Plugin, *memory.Store) {
	t.Helper()

	st := memory.New()
	l := ledger.New(st)
	require.NoError(t, l.CreatePlan(context.Background(), &plan.Plan{
		Name: "Free", Slug: "free", Currency: "usd", Status: plan.StatusActive, AppID: appID.String(),
	}))

	mgr := settings.NewManager(modeStore{mode: mode}, log.NewNoopLogger(), settings.WithCacheTTL(0))
	require.NoError(t, (&Plugin{}).DeclareSettings(mgr))

	cfg.DefaultPlanSlug = "free"
	p := New(cfg)
	p.settings = mgr
	p.SetLedger(l)
	p.SetLedgerStore(st)
	return p, st
}

func subscriptionsOf(t *testing.T, st *memory.Store, tenantID string) []*subscription.Subscription {
	t.Helper()
	subs, err := st.ListSubscriptions(context.Background(), tenantID, "", subscription.ListOpts{})
	require.NoError(t, err)
	return subs
}

func TestRegisterRoutes_DisabledMountsNothing(t *testing.T) {
	p := New(Config{DisableRoutes: true})
	router := forge.NewRouter()

	require.NoError(t, p.RegisterRoutes(router))
	assert.Empty(t, router.Routes())
}

func TestOnAfterOrgCreate_DisableAutoSubscribeWins(t *testing.T) {
	appID := id.NewAppID()
	p, st := newHookPlugin(t, Config{DisableAutoSubscribe: true, AutoSubscribeOnOrg: true}, "organization", appID)
	org := &organization.Organization{ID: id.NewOrgID(), AppID: appID}

	require.NoError(t, p.OnAfterOrgCreate(context.Background(), org))
	assert.Empty(t, subscriptionsOf(t, st, org.ID.String()))
}

func TestOnAfterOrgCreate_DefaultStillSubscribes(t *testing.T) {
	appID := id.NewAppID()
	p, st := newHookPlugin(t, Config{}, "organization", appID)
	org := &organization.Organization{ID: id.NewOrgID(), AppID: appID}

	require.NoError(t, p.OnAfterOrgCreate(context.Background(), org))
	assert.Len(t, subscriptionsOf(t, st, org.ID.String()), 1)
}

func TestOnAfterSignUp_DisableAutoSubscribeWins(t *testing.T) {
	appID := id.NewAppID()
	p, st := newHookPlugin(t, Config{DisableAutoSubscribe: true, AutoSubscribeOnUser: true}, "user", appID)
	u := &user.User{ID: id.NewUserID(), AppID: appID}

	require.NoError(t, p.OnAfterSignUp(context.Background(), u, nil))
	assert.Empty(t, subscriptionsOf(t, st, u.ID.String()))
}

func TestOnAfterSignUp_DefaultStillSubscribes(t *testing.T) {
	appID := id.NewAppID()
	p, st := newHookPlugin(t, Config{AutoSubscribeOnUser: true}, "user", appID)
	u := &user.User{ID: id.NewUserID(), AppID: appID}

	require.NoError(t, p.OnAfterSignUp(context.Background(), u, nil))
	assert.Len(t, subscriptionsOf(t, st, u.ID.String()), 1)
}

// TestOnInit_FindsLedgerFromEngine builds the engine with WithLedgerEngine and
// never calls RebindLedgerOnPlugins. The plugin has to find the ledger and its
// store on its own, because *authsome.Engine returns them with concrete types.
func TestOnInit_FindsLedgerFromEngine(t *testing.T) {
	ctx := context.Background()
	st := memory.New()
	l := ledger.New(st)
	pl := &plan.Plan{
		Name: "Free", Slug: "free", Currency: "usd", Status: plan.StatusActive,
		AppID: "aapp_01jf0000000000000000000000",
	}
	require.NoError(t, l.CreatePlan(ctx, pl))

	w, err := warden.NewEngine(warden.WithStore(wardenmem.New()))
	require.NoError(t, err)

	p := New(Config{})
	eng, err := authsome.NewEngine(
		authsome.WithStore(authmemory.New()),
		authsome.WithWarden(w),
		authsome.WithChronicle(bridge.NewMemoryChronicle()),
		authsome.WithDisableMigrate(),
		authsome.WithAppID("aapp_01jf0000000000000000000000"),
		authsome.WithLedgerEngine(l),
		authsome.WithPlugin(p),
	)
	require.NoError(t, err)
	require.NoError(t, eng.InitPlugins(ctx))

	sub, err := p.Service().Subscribe(ctx, "tenant-1", pl.ID, "aapp_01jf0000000000000000000000", nil)
	require.NoError(t, err)
	assert.Equal(t, "tenant-1", sub.TenantID)
	assert.NotNil(t, p.ledgerStore)
}
