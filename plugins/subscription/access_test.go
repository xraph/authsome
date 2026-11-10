package subscription

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	log "github.com/xraph/go-utils/log"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/forge"
	"github.com/xraph/forge/extensions/auth"
	"github.com/xraph/ledger"
	"github.com/xraph/ledger/coupon"
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
	"github.com/xraph/authsome/principal"
	"github.com/xraph/authsome/settings"
	"github.com/xraph/authsome/store"
	authmemory "github.com/xraph/authsome/store/memory"
)

// accessEngine is the slice of plugin.Engine the billing routes touch: no
// auth registry (so the group session guard is skipped and the test sets the
// user on the context itself) and an RBAC check that grants manage/billing to
// the listed users only. permErr makes every check fail; chainAllows is the
// answer for a delegated caller, whose actor chain lands in chainActors.
type accessEngine struct {
	plugin.Engine
	managers    map[string]bool
	permErr     error
	chainAllows bool
	chainActors principal.Chain
}

func (e *accessEngine) AuthRegistry() auth.Registry { return nil }

// Logger serves RequirePermission, which logs a denial through it.
func (e *accessEngine) Logger() log.Logger { return log.NewNoopLogger() }

func (e *accessEngine) HasPermission(_ context.Context, userID id.UserID, action, resource string) (bool, error) {
	if e.permErr != nil {
		return false, e.permErr
	}
	return action == "manage" && resource == "billing" && e.managers[userID.String()], nil
}

func (e *accessEngine) Can(_ context.Context, _ principal.Ref, actors principal.Chain, _, _ string) (bool, error) {
	e.chainActors = actors
	return e.chainAllows, e.permErr
}

// bareEngine has no RBAC at all: it isn't a plugin.PermissionChecker.
type bareEngine struct{ plugin.Engine }

func (bareEngine) AuthRegistry() auth.Registry { return nil }

// failingMembers is an authsome store whose membership lookup is down.
type failingMembers struct{ store.Store }

func (failingMembers) GetMemberByUserAndOrg(context.Context, id.UserID, id.OrgID) (*organization.Member, error) {
	return nil, errors.New("database unavailable")
}

// billingFixture is a plugin with routes mounted over an in-memory authsome
// store and ledger, holding one tenant with a subscription and an invoice.
type billingFixture struct {
	plugin    *Plugin
	handler   http.Handler
	authStore *authmemory.Store
	engine    *accessEngine
	// settings holds setting overrides by key, then by org ID ("" = app-wide).
	settings  map[string]map[string]any
	appID     string
	tenantID  string
	planID    string
	subID     string
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

	overrides := map[string]map[string]any{}
	mgr := settings.NewManager(modeStore{mode: mode, overrides: overrides}, log.NewNoopLogger(), settings.WithCacheTTL(0))
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
		plugin: p, handler: router.Handler(), authStore: authStore, engine: eng, settings: overrides,
		appID: appID, tenantID: tenantID, planID: pl.ID.String(), subID: sub.ID.String(), invoiceID: inv.ID.String(),
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

// send makes a request as userID scoped to appScope, the way an API key or
// an app-bound session scopes one. An empty appScope leaves it unscoped.
func (f *billingFixture) send(t *testing.T, method, path string, userID id.UserID, appScope, body string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequestWithContext(context.Background(), method, path, strings.NewReader(body))
	if body != "" {
		r.Header.Set("Content-Type", "application/json")
	}
	ctx := middleware.WithUserID(r.Context(), userID)
	if appScope != "" {
		appID, err := id.ParseAppID(appScope)
		require.NoError(t, err)
		ctx = middleware.WithAppID(ctx, appID)
	}
	rec := httptest.NewRecorder()
	f.handler.ServeHTTP(rec, r.WithContext(ctx))
	return rec
}

// addCoupon seeds a coupon in the fixture's app and returns its ID.
func (f *billingFixture) addCoupon(t *testing.T) string {
	t.Helper()
	c := &coupon.Coupon{
		ID: ledgerid.NewCouponID(), Code: "LAUNCH", Name: "Launch", Type: coupon.CouponTypePercentage, Percentage: 10, Currency: "usd", AppID: f.appID,
	}
	require.NoError(t, f.plugin.service.CreateCoupon(context.Background(), c))
	return c.ID.String()
}

// writes lists every by-ID billing write aimed at the fixture's records.
func (f *billingFixture) writes(couponID string) [][3]string {
	return [][3]string{
		{http.MethodPost, "/v1/billing/subscriptions/" + f.subID + "/change-plan", `{"new_plan_id":"` + f.planID + `"}`},
		{http.MethodPost, "/v1/billing/subscriptions/" + f.subID + "/pause", ""},
		{http.MethodPost, "/v1/billing/subscriptions/" + f.subID + "/resume", ""},
		{http.MethodPost, "/v1/billing/subscriptions/" + f.subID + "/cancel", `{"immediately":true}`},
		{http.MethodPost, "/v1/billing/invoices/" + f.invoiceID + "/pay", `{"payment_ref":"x"}`},
		{http.MethodPost, "/v1/billing/invoices/" + f.invoiceID + "/void", `{"reason":"x"}`},
		{http.MethodDelete, "/v1/billing/coupons/" + couponID, ""},
		{http.MethodPost, "/v1/billing/plans/" + f.planID + "/archive", ""},
		{http.MethodPost, "/v1/billing/plans/" + f.planID + "/activate", ""},
	}
}

func TestCrossApp_ManagerInOtherAppReadsNotFound(t *testing.T) {
	org := id.NewOrgID()
	f := newBillingFixture(t, "organization", org.String())
	f.addOrg(t, org, id.NewUserID())
	operator := id.NewUserID()
	f.engine.managers[operator.String()] = true
	f.addCoupon(t)
	otherApp := id.NewAppID().String()

	paths := append(f.readPaths(org.String()), "/v1/billing/coupons?app_id="+f.appID)
	for _, path := range paths {
		rec := f.send(t, http.MethodGet, path, operator, otherApp, "")
		assert.Equal(t, http.StatusNotFound, rec.Code, "GET %s: %s", path, rec.Body.String())
	}
}

func TestCrossApp_ManagerInOtherAppWritesNotFound(t *testing.T) {
	org := id.NewOrgID()
	f := newBillingFixture(t, "organization", org.String())
	operator := id.NewUserID()
	f.engine.managers[operator.String()] = true
	couponID := f.addCoupon(t)
	otherApp := id.NewAppID().String()

	for _, w := range f.writes(couponID) {
		rec := f.send(t, w[0], w[1], operator, otherApp, w[2])
		assert.Equal(t, http.StatusNotFound, rec.Code, "%s %s: %s", w[0], w[1], rec.Body.String())
	}

	ctx := context.Background()
	subID, err := ledgerid.ParseSubscriptionID(f.subID)
	require.NoError(t, err)
	sub, err := f.plugin.service.GetSubscription(ctx, subID)
	require.NoError(t, err)
	assert.Equal(t, subscription.StatusActive, sub.Status)
	invID, err := ledgerid.ParseInvoiceID(f.invoiceID)
	require.NoError(t, err)
	inv, err := f.plugin.service.GetInvoice(ctx, invID)
	require.NoError(t, err)
	assert.Equal(t, invoice.StatusPending, inv.Status)
	coupons, err := f.plugin.service.ListCoupons(ctx, f.appID)
	require.NoError(t, err)
	assert.Len(t, coupons, 1)
}

func TestCrossApp_SameAppManagerStillWorks(t *testing.T) {
	org := id.NewOrgID()
	f := newBillingFixture(t, "organization", org.String())
	f.addOrg(t, org, id.NewUserID())
	operator := id.NewUserID()
	f.engine.managers[operator.String()] = true
	couponID := f.addCoupon(t)

	paths := append(f.readPaths(org.String()), "/v1/billing/coupons?app_id="+f.appID)
	for _, path := range paths {
		rec := f.send(t, http.MethodGet, path, operator, f.appID, "")
		assert.Equal(t, http.StatusOK, rec.Code, "GET %s: %s", path, rec.Body.String())
	}

	rec := f.send(t, http.MethodPost, "/v1/billing/subscriptions/"+f.subID+"/cancel", operator, f.appID, `{"immediately":true}`)
	assert.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
	rec = f.send(t, http.MethodDelete, "/v1/billing/coupons/"+couponID, operator, f.appID, "")
	assert.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
}

func TestCrossApp_ScopedRequestDefaultsToItsApp(t *testing.T) {
	org := id.NewOrgID()
	f := newBillingFixture(t, "organization", org.String())
	operator := id.NewUserID()
	f.engine.managers[operator.String()] = true
	path := "/v1/billing/entitlements/sso?tenant_id=" + org.String()

	// No app_id: a request scoped to another app checks that app, not the
	// plugin's default one, so the fixture's plan grants it nothing.
	var got EntitlementResponse
	rec := f.send(t, http.MethodGet, path, operator, id.NewAppID().String(), "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.False(t, got.Allowed)

	rec = f.send(t, http.MethodGet, path, operator, f.appID, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.True(t, got.Allowed)
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

func TestListCoupons_RequiresBillingManage(t *testing.T) {
	org := id.NewOrgID()
	f := newBillingFixture(t, "organization", org.String())
	alice := id.NewUserID()
	f.addOrg(t, org, alice)
	operator := id.NewUserID()
	f.engine.managers[operator.String()] = true
	require.NoError(t, f.plugin.service.CreateCoupon(context.Background(), &coupon.Coupon{
		Code: "LAUNCH", Name: "Launch", Type: coupon.CouponTypePercentage, Percentage: 10, Currency: "usd", AppID: f.appID,
	}))
	path := "/v1/billing/coupons?app_id=" + f.appID

	assert.Equal(t, http.StatusForbidden, f.get(t, path, alice).Code)

	rec := f.get(t, path, operator)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var list ListCouponsResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &list))
	assert.Equal(t, 1, list.Total)
}

func TestHoldsBillingManage_NoRBACIsNotManager(t *testing.T) {
	p := New()
	p.engine = bareEngine{}

	assert.False(t, p.holdsBillingManage(context.Background(), id.NewUserID()))
}

func TestHoldsBillingManage_CheckErrorIsNotManager(t *testing.T) {
	operator := id.NewUserID()
	eng := &accessEngine{managers: map[string]bool{operator.String(): true}, permErr: errors.New("warden down")}
	p := New()
	p.engine = eng

	assert.False(t, p.holdsBillingManage(context.Background(), operator))
}

func TestHoldsBillingManage_DelegatedChecksChain(t *testing.T) {
	operator := id.NewUserID()
	eng := &accessEngine{managers: map[string]bool{operator.String(): true}}
	p := New()
	p.engine = eng
	agent := principal.Chain{principal.UserRef(id.NewUserID())}
	ctx := middleware.WithActors(middleware.WithUserID(context.Background(), operator), agent)

	// The user holds manage/billing, but the chain doesn't, so the agent
	// doesn't get it on the user's authority.
	assert.False(t, p.holdsBillingManage(ctx, operator))
	assert.Equal(t, agent, eng.chainActors)

	eng.chainAllows = true
	assert.True(t, p.holdsBillingManage(ctx, operator))
}

func TestRead_MembershipLookupErrorIs500(t *testing.T) {
	org := id.NewOrgID()
	f := newBillingFixture(t, "organization", org.String())
	f.plugin.authStore = failingMembers{Store: f.authStore}

	rec := f.get(t, "/v1/billing/subscriptions?tenant_id="+org.String()+"&app_id="+f.appID, id.NewUserID())
	assert.Equal(t, http.StatusInternalServerError, rec.Code, rec.Body.String())
}
