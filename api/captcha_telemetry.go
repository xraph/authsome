package api

import (
	"errors"
	"net/http"

	"github.com/xraph/authsome/captcha"
	"github.com/xraph/authsome/internal/telemetry"
)

// captchaVerifierFor is the captcha middleware's default verifier factory
// with each Turnstile call measured in the engine's provider latency
// histogram as well as traced.
func (a *API) captchaVerifierFor(provider, secret string) (captcha.Verifier, error) {
	switch provider {
	case "", "turnstile":
		if secret == "" {
			return nil, errors.New("captcha: empty secret for turnstile")
		}
		client := telemetry.InstrumentClient(&http.Client{Timeout: captcha.DefaultTurnstileTimeout},
			telemetry.Endpoint{Kind: telemetry.ProviderCaptcha}, a.engine.Telemetry())
		return captcha.NewTurnstileVerifier(secret, client), nil
	default:
		return nil, errors.New("captcha: unknown provider: " + provider)
	}
}
