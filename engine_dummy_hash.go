package authsome

import (
	"testing"

	"github.com/xraph/authsome/account"
)

// ConsumeDummyHash runs the configured password hash over password and
// discards the result, so a code path that has no stored hash to compare
// against (an unknown identifier at sign-in, a duplicate address at sign-up)
// takes as long as one that does. Without it the response time tells a
// caller whether the identifier exists. It returns the policy it hashed
// with so a test can check the cost matches a real check.
func (e *Engine) ConsumeDummyHash(password string) account.PasswordPolicy {
	if password == "" {
		// Spend the budget even for an empty guess: the shape of the
		// request must not change the timing.
		password = "x"
	}
	policy := e.passwordPolicy()
	_, _ = account.HashPasswordWithPolicy(password, policy) //nolint:errcheck // timing budget only
	if e.dummyHashObserver != nil {
		e.dummyHashObserver(policy)
	}
	return policy
}

// SetDummyHashObserver installs a test-only callback that fires on every
// dummy hash, so a test can prove the enumeration-resistant path paid the
// cost without timing it.
func (e *Engine) SetDummyHashObserver(fn func(account.PasswordPolicy)) {
	if !testing.Testing() {
		panic("authsome: SetDummyHashObserver is only available in tests")
	}
	e.dummyHashObserver = fn
}
