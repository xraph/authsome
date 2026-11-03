package contract

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"

	ledgerid "github.com/xraph/ledger/id"
	"github.com/xraph/ledger/invoice"
	"github.com/xraph/ledger/plan"
	"github.com/xraph/ledger/subscription"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/id"

	dashauth "github.com/xraph/forge/extensions/dashboard/auth"
	"github.com/xraph/forge/extensions/dashboard/contract"
	"github.com/xraph/forge/extensions/dashboard/contract/dispatcher"
	"github.com/xraph/forge/extensions/dashboard/contract/loader"
)

// billingService is the slice of SubscriptionService these tests reach, over
// one subscription and one invoice in appID. Every call is recorded, so a test
// can tell a refused intent from one that touched billing data. Methods it
// doesn't override panic on the nil embedded interface.
type billingService struct {
	SubscriptionService
	appID string
	sub   *subscription.Subscription
	inv   *invoice.Invoice
	calls []string
}

func newBillingService(appID string) *billingService {
	sub := &subscription.Subscription{ID: ledgerid.NewSubscriptionID(), TenantID: "org_other", PlanID: ledgerid.NewPlanID(), AppID: appID, Status: subscription.StatusActive}
	inv := &invoice.Invoice{ID: ledgerid.NewInvoiceID(), TenantID: "org_other", SubscriptionID: sub.ID, AppID: appID, Status: invoice.StatusPending}
	return &billingService{appID: appID, sub: sub, inv: inv}
}

func (s *billingService) ListPlans(context.Context, string) ([]*plan.Plan, error) {
	s.calls = append(s.calls, "ListPlans")
	return nil, nil
}

func (s *billingService) CreatePlan(_ context.Context, p *plan.Plan) error {
	s.calls = append(s.calls, "CreatePlan")
	p.ID = ledgerid.NewPlanID()
	return nil
}

func (s *billingService) ListSubscriptions(context.Context, string, string, subscription.ListOpts) ([]*subscription.Subscription, error) {
	s.calls = append(s.calls, "ListSubscriptions")
	return []*subscription.Subscription{s.sub}, nil
}

func (s *billingService) GetSubscription(context.Context, ledgerid.SubscriptionID) (*subscription.Subscription, error) {
	s.calls = append(s.calls, "GetSubscription")
	return s.sub, nil
}

func (s *billingService) GetPlan(context.Context, ledgerid.PlanID) (*plan.Plan, error) {
	s.calls = append(s.calls, "GetPlan")
	return &plan.Plan{ID: s.sub.PlanID, AppID: s.appID, Name: "Pro"}, nil
}

func (s *billingService) ListInvoices(context.Context, string, string) ([]*invoice.Invoice, error) {
	s.calls = append(s.calls, "ListInvoices")
	return []*invoice.Invoice{s.inv}, nil
}

func (s *billingService) ListAllInvoices(context.Context, string) ([]*invoice.Invoice, error) {
	s.calls = append(s.calls, "ListAllInvoices")
	return []*invoice.Invoice{s.inv}, nil
}

func (s *billingService) GetInvoice(context.Context, ledgerid.InvoiceID) (*invoice.Invoice, error) {
	s.calls = append(s.calls, "GetInvoice")
	return s.inv, nil
}

func (s *billingService) MarkInvoicePaid(context.Context, ledgerid.InvoiceID, string) error {
	s.calls = append(s.calls, "MarkInvoicePaid")
	return nil
}

// billingFixture is the subscription contributor registered on a dispatcher,
// with operator holding manage/billing and member holding nothing.
type billingFixture struct {
	disp     *dispatcher.Dispatcher
	svc      *billingService
	appID    string
	operator id.UserID
	member   id.UserID
	checked  []id.UserID
}

func newBillingFixture(t *testing.T, withChecker bool) *billingFixture {
	t.Helper()
	f := &billingFixture{appID: id.NewAppID().String(), operator: id.NewUserID(), member: id.NewUserID()}
	f.svc = newBillingService(f.appID)
	f.disp = dispatcher.New(dispatcher.NoopMetricsEmitter{})

	deps := Deps{Engine: &authsome.Engine{}, Service: f.svc}
	if withChecker {
		deps.CanManageBilling = func(_ context.Context, userID id.UserID) bool {
			f.checked = append(f.checked, userID)
			return userID == f.operator
		}
	}
	if err := Register(f.disp, contract.NewRegistry(), contract.NewWardenRegistry(), deps); err != nil {
		t.Fatalf("register: %v", err)
	}
	return f
}

func (f *billingFixture) principal(userID id.UserID) contract.Principal {
	return contract.Principal{User: &dashauth.UserInfo{Subject: userID.String()}, Claims: map[string]any{"app_id": f.appID}}
}

// dispatch sends one intent as p. A handler that panics on the fake service
// counts as having reached billing data.
func (f *billingFixture) dispatch(t *testing.T, kind contract.Kind, intent string, payload any, p contract.Principal) (data json.RawMessage, err error) {
	t.Helper()
	raw, mErr := json.Marshal(payload)
	if mErr != nil {
		t.Fatal(mErr)
	}
	defer func() {
		if r := recover(); r != nil {
			err = errors.New("handler reached the billing service")
		}
	}()
	data, _, err = f.disp.Dispatch(context.Background(), contract.Request{
		Kind: kind, Contributor: "subscription", Intent: intent, IntentVersion: 1, Payload: raw,
	}, p)
	return data, err
}

func errCode(err error) contract.ErrorCode {
	var ce *contract.Error
	if errors.As(err, &ce) {
		return ce.Code
	}
	return ""
}

func TestContract_ReadsRequireBillingManage(t *testing.T) {
	f := newBillingFixture(t, true)
	member := f.principal(f.member)

	reads := []struct {
		intent  string
		payload any
	}{
		{"subscriptions.list", SubscriptionsListInput{TenantID: "org_other"}},
		{"subscriptions.all", struct{}{}},
		{"subscriptions.detail", detailInput{ID: f.svc.sub.ID.String()}},
		{"invoices.list", struct{}{}},
		{"invoices.detail", detailInput{ID: f.svc.inv.ID.String()}},
	}
	for _, r := range reads {
		_, err := f.dispatch(t, contract.KindQuery, r.intent, r.payload, member)
		if got := errCode(err); got != contract.CodePermissionDenied {
			t.Errorf("%s: code = %q (%v), want %q", r.intent, got, err, contract.CodePermissionDenied)
		}
	}
	if len(f.svc.calls) != 0 {
		t.Errorf("refused reads touched the service: %v", f.svc.calls)
	}
}

func TestContract_WritesRequireBillingManage(t *testing.T) {
	f := newBillingFixture(t, true)
	member := f.principal(f.member)

	_, err := f.dispatch(t, contract.KindCommand, "invoices.markPaid", invoiceWriteInput{ID: f.svc.inv.ID.String(), PaymentRef: "ref_1"}, member)
	if got := errCode(err); got != contract.CodePermissionDenied {
		t.Errorf("invoices.markPaid: code = %q (%v), want %q", got, err, contract.CodePermissionDenied)
	}
	_, err = f.dispatch(t, contract.KindCommand, "plans.create", planWriteInput{Name: "Free", Slug: "free", Currency: "usd"}, member)
	if got := errCode(err); got != contract.CodePermissionDenied {
		t.Errorf("plans.create: code = %q (%v), want %q", got, err, contract.CodePermissionDenied)
	}
	if len(f.svc.calls) != 0 {
		t.Errorf("refused writes touched the service: %v", f.svc.calls)
	}
}

// Every intent the manifest declares is behind the gate, so a new intent
// can't be registered without it.
func TestContract_EveryIntentRequiresBillingManage(t *testing.T) {
	m, err := loader.Load(bytes.NewReader(manifestYAML), "subscription/contract/manifest.yaml")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	f := newBillingFixture(t, true)
	member := f.principal(f.member)

	for _, in := range m.Intents {
		kind := contract.KindQuery
		if in.Kind == contract.IntentKindCommand {
			kind = contract.KindCommand
		}
		_, err := f.dispatch(t, kind, in.Name, struct{}{}, member)
		if got := errCode(err); got != contract.CodePermissionDenied {
			t.Errorf("%s: code = %q (%v), want %q", in.Name, got, err, contract.CodePermissionDenied)
		}
	}
	if len(f.svc.calls) != 0 {
		t.Errorf("refused intents touched the service: %v", f.svc.calls)
	}
}

func TestContract_BillingManagerAllowed(t *testing.T) {
	f := newBillingFixture(t, true)
	operator := f.principal(f.operator)

	data, err := f.dispatch(t, contract.KindQuery, "invoices.list", struct{}{}, operator)
	if err != nil {
		t.Fatalf("invoices.list: %v", err)
	}
	var list invoicesResponse
	if err := json.Unmarshal(data, &list); err != nil {
		t.Fatal(err)
	}
	if len(list.Invoices) != 1 || list.Invoices[0].ID != f.svc.inv.ID.String() {
		t.Errorf("invoices = %+v, want the seeded invoice", list.Invoices)
	}

	if _, err := f.dispatch(t, contract.KindQuery, "subscriptions.list", SubscriptionsListInput{TenantID: "org_other"}, operator); err != nil {
		t.Fatalf("subscriptions.list: %v", err)
	}
	if _, err := f.dispatch(t, contract.KindCommand, "invoices.markPaid", invoiceWriteInput{ID: f.svc.inv.ID.String(), PaymentRef: "ref_1"}, operator); err != nil {
		t.Fatalf("invoices.markPaid: %v", err)
	}
	if f.svc.calls[len(f.svc.calls)-1] != "MarkInvoicePaid" {
		t.Errorf("calls = %v, want MarkInvoicePaid last", f.svc.calls)
	}
	for _, u := range f.checked {
		if u != f.operator {
			t.Errorf("checked %s, want only the principal %s", u, f.operator)
		}
	}
}

// Without a permission checker nobody holds billing:manage, not even the
// operator.
func TestContract_NoRBACEngineRefuses(t *testing.T) {
	f := newBillingFixture(t, false)

	_, err := f.dispatch(t, contract.KindQuery, "invoices.list", struct{}{}, f.principal(f.operator))
	if got := errCode(err); got != contract.CodePermissionDenied {
		t.Errorf("code = %q (%v), want %q", got, err, contract.CodePermissionDenied)
	}
	if len(f.svc.calls) != 0 {
		t.Errorf("refused intent touched the service: %v", f.svc.calls)
	}
}

func TestContract_AnonymousRefused(t *testing.T) {
	f := newBillingFixture(t, true)

	for _, p := range []contract.Principal{{}, {User: &dashauth.UserInfo{Subject: "not-a-user-id"}}} {
		_, err := f.dispatch(t, contract.KindQuery, "invoices.list", struct{}{}, p)
		if got := errCode(err); got != contract.CodeUnauthenticated {
			t.Errorf("principal %+v: code = %q (%v), want %q", p.User, got, err, contract.CodeUnauthenticated)
		}
	}
	if len(f.checked) != 0 || len(f.svc.calls) != 0 {
		t.Errorf("anonymous caller reached the check %v or the service %v", f.checked, f.svc.calls)
	}
}
