package subscription

// Config configures the subscription plugin.
type Config struct {
	// PathPrefix is the HTTP path prefix for billing routes.
	// Defaults to "/v1/billing".
	PathPrefix string

	// DefaultPlanSlug is the plan slug to auto-assign to new tenants.
	// If empty, no auto-subscription occurs unless overridden by settings.
	DefaultPlanSlug string

	// AutoSubscribeOnOrg creates a subscription when an organization is created.
	// Only applies when tenant mode is "organization".
	AutoSubscribeOnOrg bool

	// AutoSubscribeOnUser creates a subscription when a user signs up.
	// Only applies when tenant mode is "user".
	AutoSubscribeOnUser bool

	// TrialDays is the default trial period for new subscriptions.
	// 0 means no trial. Can be overridden by settings.
	TrialDays int

	// AllowSelfService lets users/orgs change their own plan.
	AllowSelfService bool

	// DisableRoutes mounts no HTTP routes. Use it when the host serves its
	// own billing API on top of the plugin's ledger.
	DisableRoutes bool

	// DisableAutoSubscribe turns off both the org-create and user-signup
	// auto-subscribe hooks, whatever the settings say. Use it when the host
	// creates subscriptions itself.
	DisableAutoSubscribe bool
}

func (c *Config) defaults() {
	if c.PathPrefix == "" {
		c.PathPrefix = "/v1/billing"
	}
}
