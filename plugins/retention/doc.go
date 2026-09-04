// Package retention mirrors authsome auth activity into a CRM. Hooks write to
// an outbox and a background worker delivers, so a slow or unavailable CRM
// never shows up as login latency.
//
// Each configured provider destination is either "generic" (a config-driven
// REST endpoint) or a named vendor implementation. Only "generic" ships
// today; a provider Type the plugin does not recognise fails at OnInit
// rather than dead-lettering every job later.
//
// The generic provider's retry-classification policy (classifyHTTPError in
// provider_generic.go) is decided and recorded in "Retry classification" in
// docs/superpowers/specs/2026-09-03-crm-retention-delivery-design.md: one
// rule generates the whole table, that a failure affecting every job
// retries and a failure affecting only this job dies now.
//
// Consent is optional. When the consent plugin is also registered and
// retention.require_consent is turned on, delivery is gated on an active
// grant for retention.consent_purpose; a user with no grant is never sent.
// Without the consent plugin, or with the setting left at its default, every
// user syncs.
//
// Usage:
//
//	eng, _ := authsome.NewEngine(
//	    authsome.WithStore(store),
//	    authsome.WithPlugin(retention.New(retention.Config{
//	        Providers: []retention.ProviderConfig{{
//	            Name:       "crm",
//	            Type:       "generic",
//	            ContactURL: "https://crm.example.com/api/contacts",
//	            Token:      os.Getenv("CRM_TOKEN"),
//	            FieldMap: map[string]string{
//	                "email":      "email_address",
//	                "first_name": "first_name",
//	                "last_name":  "last_name",
//	                "remote_id":  "id",
//	            },
//	        }},
//	        TickInterval: 15 * time.Second,
//	    })),
//	)
//
// A GDPR export includes every CRM ref the user has, across all apps and
// providers, through ExportUserData: see export.go.
package retention
