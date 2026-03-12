package subscription

import (
	"context"
	"time"

	log "github.com/xraph/go-utils/log"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/formconfig"
	"github.com/xraph/authsome/hook"
	"github.com/xraph/authsome/organization"
	"github.com/xraph/authsome/plugin"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/settings"
	"github.com/xraph/authsome/store"
	"github.com/xraph/authsome/user"

	"github.com/xraph/ledger"
	ledgerstore "github.com/xraph/ledger/store"
	"github.com/xraph/ledger/subscription"
)

// Compile-time interface checks.
var (
	_ plugin.Plugin           = (*Plugin)(nil)
	_ plugin.OnInit           = (*Plugin)(nil)
	_ plugin.RouteProvider    = (*Plugin)(nil)
	_ plugin.SettingsProvider = (*Plugin)(nil)
	_ plugin.AfterOrgCreate   = (*Plugin)(nil)
	_ plugin.AfterSignUp      = (*Plugin)(nil)
	_ plugin.AfterMemberAdd   = (*Plugin)(nil)
)

// ──────────────────────────────────────────────────
// Dynamic setting definitions
// ──────────────────────────────────────────────────

var (
	// SettingDefaultPlan is the plan slug to auto-assign to new tenants.
	SettingDefaultPlan = settings.Define("subscription.default_plan", "",
		settings.WithDisplayName("Default Plan"),
		settings.WithDescription("Plan slug to auto-assign to new tenants"),
		settings.WithCategory("Subscription"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithInputType(formconfig.FieldText),
		settings.WithPlaceholder("e.g. free, starter"),
		settings.WithHelpText("Leave empty to disable auto-subscription"),
		settings.WithOrder(10),
	)

	// SettingTenantMode controls whether subscriptions are org-level or user-level.
	SettingTenantMode = settings.Define("subscription.tenant_mode", "organization",
		settings.WithDisplayName("Tenant Mode"),
		settings.WithDescription("Whether subscriptions are scoped to organizations or individual users"),
		settings.WithCategory("Subscription"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithInputType(formconfig.FieldSelect),
		settings.WithOptions(
			formconfig.SelectOption{Label: "Organization", Value: "organization"},
			formconfig.SelectOption{Label: "User", Value: "user"},
		),
		settings.WithHelpText("Determines whether billing is org-level or user-level"),
		settings.WithOrder(15),
	)

	// SettingAutoSubscribeOrg controls auto-subscription for new organizations.
	SettingAutoSubscribeOrg = settings.Define("subscription.auto_subscribe_org", true,
		settings.WithDisplayName("Auto-Subscribe Organizations"),
		settings.WithDescription("Automatically create a subscription when an organization is created"),
		settings.WithCategory("Subscription"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithVisibleWhen("subscription.tenant_mode", "organization", "eq"),
		settings.WithOrder(20),
	)

	// SettingAutoSubscribeUser controls auto-subscription for new users.
	SettingAutoSubscribeUser = settings.Define("subscription.auto_subscribe_user", false,
		settings.WithDisplayName("Auto-Subscribe Users"),
		settings.WithDescription("Automatically create a subscription when a user signs up"),
		settings.WithCategory("Subscription"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithVisibleWhen("subscription.tenant_mode", "user", "eq"),
		settings.WithOrder(30),
	)

	// SettingTrialDays controls the default trial period for new subscriptions.
	SettingTrialDays = settings.Define("subscription.trial_days", 14,
		settings.WithDisplayName("Trial Period (days)"),
		settings.WithDescription("Number of trial days for new subscriptions (0 = no trial)"),
		settings.WithCategory("Subscription"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithEnforceable(),
		settings.WithInputType(formconfig.FieldNumber),
		settings.WithUIValidation(formconfig.Validation{Min: intPtr(0), Max: intPtr(365)}),
		settings.WithHelpText("Set to 0 to disable trials. Default: 14"),
		settings.WithOrder(40),
	)

	// SettingSelfServiceUpgrade controls whether tenants can change their own plan.
	SettingSelfServiceUpgrade = settings.Define("subscription.self_service_upgrade", true,
		settings.WithDisplayName("Allow Self-Service Plan Changes"),
		settings.WithDescription("Allow users/orgs to upgrade or downgrade their own plan"),
		settings.WithCategory("Subscription"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithOrder(50),
	)

	// SettingGracePeriodDays controls the grace period after billing failure.
	SettingGracePeriodDays = settings.Define("subscription.grace_period_days", 3,
		settings.WithDisplayName("Grace Period (days)"),
		settings.WithDescription("Days after billing failure before restricting access"),
		settings.WithCategory("Subscription"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithInputType(formconfig.FieldNumber),
		settings.WithUIValidation(formconfig.Validation{Min: intPtr(0), Max: intPtr(30)}),
		settings.WithHelpText("Set to 0 for immediate restriction. Default: 3"),
		settings.WithOrder(60),
	)
)

func intPtr(v int) *int { return &v }

// DeclareSettings implements plugin.SettingsProvider.
func (p *Plugin) DeclareSettings(m *settings.Manager) error {
	if err := settings.RegisterTyped(m, "subscription", SettingDefaultPlan); err != nil {
		return err
	}
	if err := settings.RegisterTyped(m, "subscription", SettingTenantMode); err != nil {
		return err
	}
	if err := settings.RegisterTyped(m, "subscription", SettingAutoSubscribeOrg); err != nil {
		return err
	}
	if err := settings.RegisterTyped(m, "subscription", SettingAutoSubscribeUser); err != nil {
		return err
	}
	if err := settings.RegisterTyped(m, "subscription", SettingTrialDays); err != nil {
		return err
	}
	if err := settings.RegisterTyped(m, "subscription", SettingSelfServiceUpgrade); err != nil {
		return err
	}
	return settings.RegisterTyped(m, "subscription", SettingGracePeriodDays)
}

// Plugin is the subscription management plugin for authsome.
type Plugin struct {
	config  Config
	service *Service

	// Ledger references
	ledger      *ledger.Ledger
	ledgerStore ledgerstore.Store
	ledgerBrdg  bridge.Ledger

	// AuthSome references
	authStore    store.Store
	chronicle    bridge.Chronicle
	relay        bridge.EventRelay
	hooks        *hook.Bus
	logger       log.Logger
	settings     *settings.Manager
	defaultAppID string
}

// New creates a new subscription plugin with the given configuration.
func New(cfg ...Config) *Plugin {
	var c Config
	if len(cfg) > 0 {
		c = cfg[0]
	}
	c.defaults()
	return &Plugin{config: c}
}

// Name returns the plugin name.
func (p *Plugin) Name() string { return "subscription" }

// OnInit captures bridge and engine references.
func (p *Plugin) OnInit(_ context.Context, engine any) error {
	// Discover ledger engine.
	type ledgerEngineGetter interface {
		Ledger() *ledger.Ledger
	}
	if lg, ok := engine.(ledgerEngineGetter); ok {
		p.ledger = lg.Ledger()
	}

	// Discover ledger store (for list operations).
	type ledgerStoreGetter interface {
		LedgerStore() ledgerstore.Store
	}
	if lsg, ok := engine.(ledgerStoreGetter); ok {
		p.ledgerStore = lsg.LedgerStore()
	}

	// Discover ledger bridge.
	type ledgerBridgeGetter interface {
		Ledger() bridge.Ledger
	}
	if lbg, ok := engine.(ledgerBridgeGetter); ok {
		p.ledgerBrdg = lbg.Ledger()
	}

	// Discover AuthSome store.
	type storeGetter interface {
		Store() store.Store
	}
	if sg, ok := engine.(storeGetter); ok {
		p.authStore = sg.Store()
	}

	// Discover chronicle bridge.
	type chronicleGetter interface {
		Chronicle() bridge.Chronicle
	}
	if cg, ok := engine.(chronicleGetter); ok {
		p.chronicle = cg.Chronicle()
	}

	// Discover relay bridge.
	type relayGetter interface {
		Relay() bridge.EventRelay
	}
	if rg, ok := engine.(relayGetter); ok {
		p.relay = rg.Relay()
	}

	// Discover hook bus.
	type hooksGetter interface {
		Hooks() *hook.Bus
	}
	if hg, ok := engine.(hooksGetter); ok {
		p.hooks = hg.Hooks()
	}

	// Discover logger.
	type loggerGetter interface {
		Logger() log.Logger
	}
	if lg, ok := engine.(loggerGetter); ok {
		p.logger = lg.Logger()
	}

	// Discover settings manager.
	type settingsGetter interface {
		Settings() *settings.Manager
	}
	if sg, ok := engine.(settingsGetter); ok {
		p.settings = sg.Settings()
	}

	// Discover default app ID.
	type configGetter interface {
		Config() authsome.Config
	}
	if cg, ok := engine.(configGetter); ok {
		p.defaultAppID = cg.Config().AppID
	}

	// Initialize the service layer.
	p.service = &Service{
		ledger:      p.ledger,
		ledgerStore: p.ledgerStore,
		ledgerBrdg:  p.ledgerBrdg,
		authStore:   p.authStore,
		settings:    p.settings,
		logger:      p.logger,
	}

	return nil
}

// ──────────────────────────────────────────────────
// Lifecycle hooks
// ──────────────────────────────────────────────────

// OnAfterOrgCreate auto-creates a subscription for new organizations
// when tenant mode is "organization" and auto-subscribe is enabled.
func (p *Plugin) OnAfterOrgCreate(ctx context.Context, o *organization.Organization) error {
	if p.ledger == nil {
		return nil
	}

	appID := o.AppID.String()

	// Resolve tenant mode setting.
	tenantMode, _ := settings.Get(ctx, p.settings, SettingTenantMode, settings.ResolveOpts{AppID: appID})
	if tenantMode != "organization" {
		return nil
	}

	// Check auto-subscribe setting.
	autoSub, _ := settings.Get(ctx, p.settings, SettingAutoSubscribeOrg, settings.ResolveOpts{AppID: appID})
	if !autoSub && !p.config.AutoSubscribeOnOrg {
		return nil
	}

	return p.autoSubscribe(ctx, o.ID.String(), appID)
}

// OnAfterSignUp auto-creates a subscription for new users
// when tenant mode is "user" and auto-subscribe is enabled.
func (p *Plugin) OnAfterSignUp(ctx context.Context, u *user.User, s *session.Session) error {
	if p.ledger == nil || u == nil || s == nil {
		return nil
	}

	appID := s.AppID.String()

	// Resolve tenant mode setting.
	tenantMode, _ := settings.Get(ctx, p.settings, SettingTenantMode, settings.ResolveOpts{AppID: appID})
	if tenantMode != "user" {
		return nil
	}

	// Check auto-subscribe setting.
	autoSub, _ := settings.Get(ctx, p.settings, SettingAutoSubscribeUser, settings.ResolveOpts{AppID: appID})
	if !autoSub && !p.config.AutoSubscribeOnUser {
		return nil
	}

	return p.autoSubscribe(ctx, u.ID.String(), appID)
}

// OnAfterMemberAdd records seat usage when a member is added to an organization.
func (p *Plugin) OnAfterMemberAdd(ctx context.Context, m *organization.Member) error {
	if p.ledgerBrdg == nil {
		return nil
	}
	_ = p.ledgerBrdg.RecordUsage(ctx, "authsome.orgs.members", 1) //nolint:errcheck
	return nil
}

// autoSubscribe creates a subscription for the given tenant using the default plan.
func (p *Plugin) autoSubscribe(ctx context.Context, tenantID, appID string) error {
	// Resolve the default plan slug.
	planSlug, _ := settings.Get(ctx, p.settings, SettingDefaultPlan, settings.ResolveOpts{AppID: appID})
	if planSlug == "" {
		planSlug = p.config.DefaultPlanSlug
	}
	if planSlug == "" {
		return nil // No default plan configured.
	}

	plan, err := p.ledger.GetPlanBySlug(ctx, planSlug, appID)
	if err != nil {
		if p.logger != nil {
			p.logger.Warn("subscription: auto-subscribe failed to find plan",
				log.String("plan_slug", planSlug),
				log.String("tenant_id", tenantID),
				log.Error(err),
			)
		}
		return nil // Non-fatal — don't block org/user creation.
	}

	// Resolve trial days.
	trialDays, _ := settings.Get(ctx, p.settings, SettingTrialDays, settings.ResolveOpts{AppID: appID})
	if trialDays == 0 {
		trialDays = p.config.TrialDays
	}

	now := time.Now()
	sub := &subscription.Subscription{
		TenantID:           tenantID,
		PlanID:             plan.ID,
		Status:             subscription.StatusActive,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		AppID:              appID,
	}

	if trialDays > 0 {
		sub.Status = subscription.StatusTrialing
		trialEnd := now.AddDate(0, 0, trialDays)
		sub.TrialStart = &now
		sub.TrialEnd = &trialEnd
		sub.CurrentPeriodEnd = trialEnd
	}

	if err := p.ledger.CreateSubscription(ctx, sub); err != nil {
		if p.logger != nil {
			p.logger.Warn("subscription: auto-subscribe failed",
				log.String("tenant_id", tenantID),
				log.String("plan", planSlug),
				log.Error(err),
			)
		}
		return nil // Non-fatal.
	}

	p.audit(ctx, "subscription.create", "subscription", sub.ID.String(), tenantID, tenantID, bridge.OutcomeSuccess)
	p.relayEvent(ctx, "subscription.created", tenantID, map[string]string{
		"plan_slug": planSlug,
		"trial":     boolStr(trialDays > 0),
	})

	return nil
}

func boolStr(v bool) string {
	if v {
		return "true"
	}
	return "false"
}
