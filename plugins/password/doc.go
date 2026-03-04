// Package password provides a built-in password authentication plugin for AuthSome.
//
// The password plugin wraps the core email/password authentication flow as a
// named plugin with configurable validation rules. It hooks into BeforeSignUp
// to enforce password policy, and provides a StrategyProvider for password-based
// authentication.
package password
