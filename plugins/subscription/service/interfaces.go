// Package service provides business logic services for the subscription plugin.
package service

import (
	"context"

	"github.com/rs/xid"
	"github.com/xraph/authsome/plugins/subscription/core"
)

// PlanServiceInterface defines the plan service interface
type PlanServiceInterface interface {
	Create(ctx context.Context, appID xid.ID, req *core.CreatePlanRequest) (*core.Plan, error)
	Update(ctx context.Context, id xid.ID, req *core.UpdatePlanRequest) (*core.Plan, error)
	Delete(ctx context.Context, id xid.ID) error
	GetByID(ctx context.Context, id xid.ID) (*core.Plan, error)
	GetBySlug(ctx context.Context, appID xid.ID, slug string) (*core.Plan, error)
	List(ctx context.Context, appID xid.ID, activeOnly, publicOnly bool, page, pageSize int) ([]*core.Plan, int, error)
	SetActive(ctx context.Context, id xid.ID, active bool) error
	SetPublic(ctx context.Context, id xid.ID, public bool) error
	SyncToProvider(ctx context.Context, id xid.ID) error
	SyncFromProvider(ctx context.Context, providerPlanID string) (*core.Plan, error)
	SyncAllFromProvider(ctx context.Context, appID xid.ID) ([]*core.Plan, error)
}

// SubscriptionServiceInterface defines the subscription service interface
type SubscriptionServiceInterface interface {
	Create(ctx context.Context, req *core.CreateSubscriptionRequest) (*core.Subscription, error)
	Update(ctx context.Context, id xid.ID, req *core.UpdateSubscriptionRequest) (*core.Subscription, error)
	Cancel(ctx context.Context, id xid.ID, req *core.CancelSubscriptionRequest) error
	Pause(ctx context.Context, id xid.ID, req *core.PauseSubscriptionRequest) error
	Resume(ctx context.Context, id xid.ID) error
	GetByID(ctx context.Context, id xid.ID) (*core.Subscription, error)
	GetByOrganizationID(ctx context.Context, orgID xid.ID) (*core.Subscription, error)
	List(ctx context.Context, appID, orgID, planID *xid.ID, status string, page, pageSize int) ([]*core.Subscription, int, error)
	ChangePlan(ctx context.Context, id, newPlanID xid.ID) (*core.Subscription, error)
	UpdateQuantity(ctx context.Context, id xid.ID, quantity int) (*core.Subscription, error)
	AttachAddOn(ctx context.Context, subID, addOnID xid.ID, quantity int) error
	DetachAddOn(ctx context.Context, subID, addOnID xid.ID) error
	SyncFromProvider(ctx context.Context, providerSubID string) (*core.Subscription, error)
}

// AddOnServiceInterface defines the add-on service interface
type AddOnServiceInterface interface {
	Create(ctx context.Context, appID xid.ID, req *core.CreateAddOnRequest) (*core.AddOn, error)
	Update(ctx context.Context, id xid.ID, req *core.UpdateAddOnRequest) (*core.AddOn, error)
	Delete(ctx context.Context, id xid.ID) error
	GetByID(ctx context.Context, id xid.ID) (*core.AddOn, error)
	GetBySlug(ctx context.Context, appID xid.ID, slug string) (*core.AddOn, error)
	List(ctx context.Context, appID xid.ID, activeOnly, publicOnly bool, page, pageSize int) ([]*core.AddOn, int, error)
	GetAvailableForPlan(ctx context.Context, planID xid.ID) ([]*core.AddOn, error)
}

// InvoiceServiceInterface defines the invoice service interface
type InvoiceServiceInterface interface {
	GetByID(ctx context.Context, id xid.ID) (*core.Invoice, error)
	GetByNumber(ctx context.Context, number string) (*core.Invoice, error)
	List(ctx context.Context, orgID, subID *xid.ID, status string, page, pageSize int) ([]*core.Invoice, int, error)
	GetPDFURL(ctx context.Context, id xid.ID) (string, error)
	Void(ctx context.Context, id xid.ID) error
	SyncFromProvider(ctx context.Context, providerInvoiceID string) (*core.Invoice, error)
}

// UsageServiceInterface defines the usage service interface
type UsageServiceInterface interface {
	RecordUsage(ctx context.Context, req *core.RecordUsageRequest) (*core.UsageRecord, error)
	GetSummary(ctx context.Context, req *core.GetUsageSummaryRequest) (*core.UsageSummary, error)
	GetUsageLimit(ctx context.Context, orgID xid.ID, metricKey string) (*core.UsageLimit, error)
	ReportToProvider(ctx context.Context, batchSize int) error
}

// PaymentServiceInterface defines the payment service interface
type PaymentServiceInterface interface {
	CreateSetupIntent(ctx context.Context, orgID xid.ID) (*core.SetupIntentResult, error)
	AddPaymentMethod(ctx context.Context, req *core.AddPaymentMethodRequest) (*core.PaymentMethod, error)
	RemovePaymentMethod(ctx context.Context, id xid.ID) error
	SetDefaultPaymentMethod(ctx context.Context, orgID, paymentMethodID xid.ID) error
	ListPaymentMethods(ctx context.Context, orgID xid.ID) ([]*core.PaymentMethod, error)
	GetDefaultPaymentMethod(ctx context.Context, orgID xid.ID) (*core.PaymentMethod, error)
}

// CustomerServiceInterface defines the customer service interface
type CustomerServiceInterface interface {
	Create(ctx context.Context, req *core.CreateCustomerRequest) (*core.Customer, error)
	Update(ctx context.Context, id xid.ID, req *core.UpdateCustomerRequest) (*core.Customer, error)
	GetByOrganizationID(ctx context.Context, orgID xid.ID) (*core.Customer, error)
	GetOrCreate(ctx context.Context, orgID xid.ID, email, name string) (*core.Customer, error)
	SyncToProvider(ctx context.Context, id xid.ID) error
}

// EnforcementServiceInterface defines the enforcement service interface
type EnforcementServiceInterface interface {
	CheckFeatureAccess(ctx context.Context, orgID xid.ID, feature string) (bool, error)
	GetRemainingSeats(ctx context.Context, orgID xid.ID) (int, error)
	GetFeatureLimit(ctx context.Context, orgID xid.ID, feature string) (int64, error)
	GetAllLimits(ctx context.Context, orgID xid.ID) (map[string]*core.UsageLimit, error)
	EnforceSubscriptionRequired(ctx context.Context, req interface{}) error
	EnforceSeatLimit(ctx context.Context, orgID string, userID xid.ID) error
}
