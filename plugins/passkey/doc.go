// Package passkey provides WebAuthn/passkey authentication for AuthSome.
//
// The passkey plugin supports FIDO2 WebAuthn registration and authentication
// ceremonies. It owns its own credential store (like the MFA plugin) and
// provides HTTP routes via the RouteProvider interface.
//
// Usage:
//
//	eng, _ := authsome.NewEngine(
//	    authsome.WithStore(store),
//	    authsome.WithPlugin(passkey.New(passkey.Config{
//	        RPDisplayName: "My App",
//	        RPID:          "example.com",
//	        RPOrigins:     []string{"https://example.com"},
//	    })),
//	)
package passkey
