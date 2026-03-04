// Package consent provides GDPR-compliant consent tracking for AuthSome.
//
// The consent plugin records when users grant or revoke consent for specific
// purposes (e.g., marketing, analytics, essential). Each consent record is
// versioned against a policy version, and the full history is queryable.
//
// Usage:
//
//	eng, _ := authsome.NewEngine(
//	    authsome.WithStore(store),
//	    authsome.WithPlugin(consent.New()),
//	)
package consent
