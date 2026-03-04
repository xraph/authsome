// Package email provides transactional email notifications for AuthSome.
//
// The email plugin listens to auth lifecycle events (sign-up, user creation,
// password reset, invitation) and sends contextual transactional emails via
// the bridge.Mailer interface. No SMTP logic lives here — the plugin delegates
// all delivery to whatever Mailer implementation the host app configures.
//
// Usage:
//
//	eng, _ := authsome.NewEngine(
//	    authsome.WithStore(store),
//	    authsome.WithMailer(myMailer),
//	    authsome.WithPlugin(email.New(email.Config{
//	        From:    "noreply@example.com",
//	        AppName: "My App",
//	        BaseURL: "https://example.com",
//	    })),
//	)
package email
