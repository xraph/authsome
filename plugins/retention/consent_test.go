package retention

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/xraph/authsome/id"
)

// policy builds a fixed consentPolicy for tests.
func policy(require bool, purpose string) func(context.Context, id.AppID) (bool, string) {
	return func(context.Context, id.AppID) (bool, string) { return require, purpose }
}

type stubConsent struct {
	granted bool
	err     error
	calls   int
}

func (s *stubConsent) HasConsent(context.Context, id.UserID, id.AppID, string) (bool, error) {
	s.calls++
	return s.granted, s.err
}

func TestAllowSendPassesWhenGateDisabled(t *testing.T) {
	p := New()
	p.consentPolicy = policy(false, "marketing")
	p.consent = &stubConsent{granted: false}

	ok, _ := p.allowSend(context.Background(), &Job{})
	assert.True(t, ok, "with the gate off, consent is not consulted")
	assert.Zero(t, p.consent.(*stubConsent).calls)
}

func TestAllowSendBlocksWithoutGrant(t *testing.T) {
	p := New()
	p.consentPolicy = policy(true, "marketing")
	p.consent = &stubConsent{granted: false}

	ok, reason := p.allowSend(context.Background(), &Job{})
	assert.False(t, ok)
	assert.Contains(t, reason, "marketing")
}

func TestAllowSendPassesWithGrant(t *testing.T) {
	p := New()
	p.consentPolicy = policy(true, "marketing")
	p.consent = &stubConsent{granted: true}

	ok, _ := p.allowSend(context.Background(), &Job{})
	assert.True(t, ok)
}

func TestAllowSendBlocksWhenGateOnButConsentUnavailable(t *testing.T) {
	p := New()
	p.consentPolicy = policy(true, "marketing")
	p.consent = nil // consent plugin not registered

	ok, reason := p.allowSend(context.Background(), &Job{})
	assert.False(t, ok, "asking for a gate you cannot evaluate must not send")
	assert.Contains(t, reason, "unavailable")
}

func TestAllowSendBlocksOnLookupError(t *testing.T) {
	p := New()
	p.consentPolicy = policy(true, "marketing")
	p.consent = &stubConsent{err: assert.AnError}

	ok, _ := p.allowSend(context.Background(), &Job{})
	assert.False(t, ok, "a failed lookup must not be read as consent")
}
