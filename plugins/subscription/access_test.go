package subscription

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	log "github.com/xraph/go-utils/log"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/forge"
	"github.com/xraph/forge/extensions/auth"
	"github.com/xraph/ledger"
	ledgerid "github.com/xraph/ledger/id"
	"github.com/xraph/ledger/invoice"
	"github.com/xraph/ledger/plan"
	"github.com/xraph/ledger/store/memory"
	"github.com/xraph/ledger/subscription"
	"github.com/xraph/ledger/types"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/organization"
	"github.com/xraph/authsome/plugin"
	"github.com/xraph/authsome/settings"
	authmemory "github.com/xraph/authsome/store/memory"
)

// accessEngine is the slice of plugin.Engine the billing routes touch: no
// auth registry (so the group session guard is skipped and the test sets the
// user on the context itself) and an RBAC check that grants manage/billing to
// the listed users only.
type accessEngine struct {
	plugin.Engine
	managers map[string]bool
}

func (e *accessEngine) AuthRegistry() auth.Registry { return nil }

func (e *accessEngine) HasPermission(_ context.Context, userID id.UserID, action, resource string) (bool, error) {
	return action == "manage" && resource == "billing" && e.managers[userID.String()], nil
}

// billingFixture is a plugin with routes mounted over an in-memory authsome
// store and ledger, holding one tenant with a subscription and an invoice.
type billingFixture struct {
	handler   http.Handler
	authStore *authmemory.Store
	engine    *accessEngine
	appID     string
	tenantID  string
	invoiceID string
}

// newBillingFixture builds the fixture in the given tenant mode. tenantID is
// the tenant that owns the seeded subscription and invoice.
func newBillingFixture(t *testing.T, mode, tenantID string) *billingFixture {
	t.Helper()
	ctx := context.Background()
	appID := id.NewAppID().String()

	st := memory.New()
	l := ledger.New(st)
	pl := &plan.Plan{
		Name: "Pro", Slug: "pro", Currency: "usd", Status: plan.StatusActive, AppID: appID,
		Features: []plan.Feature{{ID: ledgerid.NewFeatureID(), Key: "sso", Name: "SSO", Type: plan.FeatureBoolean, Limit: 1}},
	}
	require.NoError(t, l.CreatePlan(ctx, pl))

	now := time.Now()
	sub := &subscription.Subscription{
		TenantID: tenantID, PlanID: pl.ID, Status: subscription.StatusActive,
		CurrentPeriodStart: now, CurrentPeriodEnd: now.AddDate(0, 1, 0), AppID: appID,
	}
	require.NoError(t, l.CreateSubscription(ctx, sub))

	inv := &invoice.Invoice{
		ID: ledgerid.NewInvoiceID(), TenantID: tenantID, SubscriptionID: sub.ID,
		Status: invoice.StatusPending, Currency: "usd", Total: types.Money{Amount: 1000, Currency: "usd"},
		PeriodStart: now, PeriodEnd: now.AddDate(0, 1, 0), AppID: appID,
	}
	require.NoError(t, st.CreateInvoice(ctx, inv))

	mgr := settings.NewManager(modeStore{mode: mode}, log.NewNoopLogger(), settings.WithCacheTTL(0))
	require.NoError(t, (&Plugin{}).DeclareSettings(mgr))

	eng := &accessEngine{managers: map[string]bool{}}
	authStore := authmemory.New()

	p := New()
	p.engine = eng
	p.authStore = authStore
	p.settings = mgr
	p.defaultAppID = appID
	p.service = &Service{settings: mgr}
	p.SetLedger(l)
	p.SetLedgerStore(st)

	router := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(router))

	return &billingFixture{
		handler: router.Handler(), authStore: authStore, engine: eng,
		appID: appID, tenantID: tenantID, invoiceID: inv.ID.String(),
	}
}

// addOrg creates an org whose ID is orgID with userID as its only member.
func (f *billingFixture) addOrg(t *testing.T, orgID id.OrgID, userID id.UserID) {
	t.Helper()
	ctx := context.Background()
	appID, err := id.ParseAppID(f.appID)
	require.NoError(t, err)
	require.NoError(t, f.authStore.CreateOrganization(ctx, &organization.Organization{
		ID: orgID, AppID: appID, Name: "Org", Slug: orgID.String(), CreatedBy: userID,
	}))
	require.NoError(t, f.authStore.CreateMember(ctx, &organization.Member{
		ID: id.NewMemberID(), OrgID: orgID, UserID: userID, Role: organization.RoleMember,
	}))
}

// readPaths lists every tenant-scoped read route, aimed at tenantID.
func (f *billingFixture) readPaths(tenantID string) []string {
	q := "?tenant_id=" + tenantID + "&app_id=" + f.appID
	return []string{
		"/v1/billing/subscriptions" + q,
		"/v1/billing/subscriptions/active" + q,
		"/v1/billing/invoices" + q,
		"/v1/billing/invoices/" + f.invoiceID,
		"/v1/billing/entitlements/sso" + q,
		"/v1/billing/usage" + q,
	}
}

func (f *billingFixture) get(t *testing.T, path string, userID id.UserID) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequestWithContext(context.Background(), http.MethodGet, path, nil)
	r = r.WithContext(middleware.WithUserID(r.Context(), userID))
	rec := httptest.NewRecorder()
	f.handler.ServeHTTP(rec, r)
	return rec
}

func TestRead_OtherOrgIsNotFound(t *testing.T) {
	victim, attacker := id.NewOrgID(), id.NewOrgID()
	f := newBillingFixture(t, "organization", victim.String())
	f.addOrg(t, victim, id.NewUserID())
	mallory := id.NewUserID()
	f.addOrg(t, attacker, mallory)

	for _, path := range f.readPaths(victim.String()) {
		rec := f.get(t, path, mallory)
		assert.Equal(t, http.StatusNotFound, rec.Code, "GET %s: %s", path, rec.Body.String())
	}
}

func TestRead_MemberCanRead(t *testing.T) {
	org := id.NewOrgID()
	f := newBillingFixture(t, "organization", org.String())
	alice := id.NewUserID()
	f.addOrg(t, org, alice)

	for _, path := range f.readPaths(org.String()) {
		rec := f.get(t, path, alice)
		assert.Equal(t, http.StatusOK, rec.Code, "GET %s: %s", path, rec.Body.String())
	}

	var list ListInvoicesResponse
	rec := f.get(t, "/v1/billing/invoices?tenant_id="+org.String()+"&app_id="+f.appID, alice)
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &list))
	assert.Equal(t, 1, list.Total)
}

func TestRead_UserModeOtherUserNotFound(t *testing.T) {
	owner, other := id.NewUserID(), id.NewUserID()
	f := newBillingFixture(t, "user", owner.String())

	for _, path := range f.readPaths(owner.String()) {
		assert.Equal(t, http.StatusNotFound, f.get(t, path, other).Code, "GET %s as another user", path)
		assert.Equal(t, http.StatusOK, f.get(t, path, owner).Code, "GET %s as the owner", path)
	}
}

func TestRead_BillingManagerCanReadAny(t *testing.T) {
	org := id.NewOrgID()
	f := newBillingFixture(t, "organization", org.String())
	f.addOrg(t, org, id.NewUserID())
	operator := id.NewUserID()
	f.engine.managers[operator.String()] = true

	for _, path := range f.readPaths(org.String()) {
		rec := f.get(t, path, operator)
		assert.Equal(t, http.StatusOK, rec.Code, "GET %s: %s", path, rec.Body.String())
	}
}

func TestCheckEntitlement_SetsTenantContext(t *testing.T) {
	org := id.NewOrgID()
	f := newBillingFixture(t, "organization", org.String())
	alice := id.NewUserID()
	f.addOrg(t, org, alice)

	rec := f.get(t, "/v1/billing/entitlements/sso?tenant_id="+org.String()+"&app_id="+f.appID, alice)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var got EntitlementResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.NotEqual(t, "missing tenant or app context", got.Reason)
	assert.True(t, got.Allowed)
}
