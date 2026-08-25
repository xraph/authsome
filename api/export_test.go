package api

import "github.com/xraph/authsome/account"

// SetHashBudgetObserver installs fn as the duplicate-signup hash observer on
// a. fn is called, with the policy that was used, every time the
// duplicate-email branch of handleSignUp spends its password-hash time
// budget.
//
// Test-only: it exists so tests can assert the enumeration-resistance dummy
// hash actually runs (and runs with the engine's real cost parameters)
// instead of trying to detect it by timing the HTTP response, which is not a
// reliable signal. Call it before serving any request through a.
func SetHashBudgetObserver(a *API, fn func(account.PasswordPolicy)) {
	a.hashBudgetObserver = fn
}
