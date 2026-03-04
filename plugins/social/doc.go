// Package social provides OAuth2-based social login authentication for AuthSome.
//
// The social plugin enables users to sign in with third-party OAuth2 providers
// such as Google, GitHub, Apple, and Microsoft. It manages the full OAuth2 flow
// including authorization redirect, callback handling, token exchange, and user
// profile fetching. Social connections are stored per-user and linked to the
// core user account.
//
// Usage:
//
//	eng, _ := authsome.NewEngine(
//	    authsome.WithStore(store),
//	    authsome.WithPlugin(social.New(social.Config{
//	        Providers: []social.Provider{
//	            social.Google("client-id", "client-secret", "https://example.com/callback"),
//	        },
//	    })),
//	)
package social
