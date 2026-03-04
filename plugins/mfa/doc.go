// Package mfa provides multi-factor authentication for AuthSome.
//
// The MFA plugin supports TOTP (Time-based One-Time Password) enrollment
// and verification. It implements AfterSignIn to inject MFA challenges
// when users have MFA enabled, and provides HTTP routes for enrollment
// and verification.
//
// Usage:
//
//	eng, _ := authsome.NewEngine(
//	    authsome.WithStore(store),
//	    authsome.WithPlugin(mfa.New(mfa.Config{
//	        Issuer: "My App",
//	    })),
//	)
package mfa
