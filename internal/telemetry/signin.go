package telemetry

import (
	"errors"
	"net/http"
)

// SignInAttempt counts one sign-in through a plugin's verify or callback
// handler exactly once. Start it at the top of the handler and defer Done
// with the handler's error. Call HandOff right before IssueSession, which
// counts the attempt itself from then on; call Refuse where the handler
// answers a refusal without returning an error (a redirect carrying the
// error, say), so Done doesn't read the nil error as success.
//
// A nil *SignInAttempt (from a nil *Metrics) does nothing.
type SignInAttempt struct {
	m         *Metrics
	method    string
	result    string
	handedOff bool
}

// StartSignIn begins counting an attempt under method, one of the Method*
// constants.
func (t *Metrics) StartSignIn(method string) *SignInAttempt {
	if t == nil {
		return nil
	}
	return &SignInAttempt{m: t, method: method}
}

// HandOff leaves the count to IssueSession.
func (a *SignInAttempt) HandOff() {
	if a != nil {
		a.handedOff = true
	}
}

// Refuse sets the result Done records, whatever error it is given.
func (a *SignInAttempt) Refuse(result string) {
	if a != nil {
		a.result = result
	}
}

// Done records the attempt unless it was handed off. Without a Refuse it
// reads the result off err: nil is ok, and an HTTP error's status sorts it
// into invalid (400, 401, 404), denied (403), rate_limited (429) or error.
func (a *SignInAttempt) Done(err error) {
	if a == nil || a.handedOff {
		return
	}
	result := a.result
	if result == "" {
		result = refusalResult(err)
	}
	a.m.SignIn(a.method, result)
}

func refusalResult(err error) string {
	if err == nil {
		return SignInOK
	}
	var coded interface{ StatusCode() int }
	if !errors.As(err, &coded) {
		return SignInError
	}
	switch code := coded.StatusCode(); {
	case code == http.StatusForbidden:
		return SignInDenied
	case code == http.StatusTooManyRequests:
		return SignInRateLimited
	case code >= 400 && code < 500:
		return SignInInvalid
	default:
		return SignInError
	}
}
