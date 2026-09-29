package authsome

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"time"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/hook"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/safeurl"
	"github.com/xraph/authsome/store"
	"github.com/xraph/authsome/webhook"
)

// ErrWebhooksUnavailable is returned by every webhook operation when the
// engine's relay cannot manage delivery endpoints (no relay, or a
// send-only one). A webhook that cannot deliver is not registered at all,
// so an operator never holds a subscription that silently does nothing.
var ErrWebhooksUnavailable = errors.New("authsome: webhooks unavailable: the relay cannot manage endpoints")

// ErrWebhookURLRejected wraps a URL the safety check refused.
var ErrWebhookURLRejected = errors.New("authsome: webhook url rejected")

// ErrWebhookEvents is returned when a subscription names no event, or an
// event that is not shaped like one.
var ErrWebhookEvents = errors.New("authsome: webhook events invalid")

// webhookEventName is the shape of an event type: dotted lowercase words,
// with * allowed as a segment or suffix so a subscription can say user.*.
// Plugins register their own event types with the relay at start, so the
// engine validates the shape rather than a fixed list.
var webhookEventName = regexp.MustCompile(`^[a-z0-9_]+(\.[a-z0-9_*]+)+$|^\*$`)

// endpointRelay returns the relay as an endpoint manager, or false when
// webhooks are unavailable.
func (e *Engine) endpointRelay() (bridge.EndpointRelay, bool) {
	r, ok := e.relay.(bridge.EndpointRelay)
	return r, ok && r != nil
}

func (e *Engine) webhookURLOptions() safeurl.Options {
	return safeurl.Options{AllowInsecure: e.config.Webhooks.AllowInsecureURLs}
}

func (e *Engine) webhookVerifyTimeout() time.Duration {
	if e.config.Webhooks.VerifyTimeout > 0 {
		return e.config.Webhooks.VerifyTimeout
	}
	return 10 * time.Second
}

func validateWebhookEvents(events []string) error {
	if len(events) == 0 {
		return fmt.Errorf("%w: at least one event type is required", ErrWebhookEvents)
	}
	for _, ev := range events {
		if !webhookEventName.MatchString(ev) {
			return fmt.Errorf("%w: %q is not an event type", ErrWebhookEvents, ev)
		}
	}
	return nil
}

// generateWebhookSecret generates a random hex secret for webhook signing.
func generateWebhookSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "whsec_" + hex.EncodeToString(b), nil
}

// verifyWebhookURL checks the URL may be called and then makes one test
// delivery to it, signed with the secret the receiver will hold, through a
// client that pins the resolved address and refuses redirects. Anything but
// a 2xx answer refuses the URL: a redirect in particular, since a receiver
// that redirects would carry every delivery, signature and all, somewhere
// the check never looked.
func (e *Engine) verifyWebhookURL(ctx context.Context, w *webhook.Webhook, secret string) error {
	opts := e.webhookURLOptions()
	if _, err := safeurl.Validate(ctx, w.URL, opts); err != nil {
		return fmt.Errorf("%w: %w", ErrWebhookURLRejected, err)
	}
	ts := time.Now()
	body, err := json.Marshal(map[string]string{
		"type":       "webhook.test",
		"webhook_id": w.ID.String(),
		"app_id":     w.AppID.String(),
		"env_id":     w.EnvID.String(),
		"sent_at":    ts.UTC().Format(time.RFC3339),
	})
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, e.webhookVerifyTimeout())
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.URL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("%w: %w", ErrWebhookURLRejected, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "authsome-webhook-verify")
	req.Header.Set("X-Relay-Event-Type", "webhook.test")
	req.Header.Set(webhook.RelayTimestampHeader, strconv.FormatInt(ts.Unix(), 10))
	req.Header.Set(webhook.RelaySignatureHeader, webhook.SignRelay(secret, ts, body))
	resp, err := safeurl.Client(e.webhookVerifyTimeout(), opts).Do(req)
	if err != nil {
		return fmt.Errorf("%w: test delivery failed: %w", ErrWebhookURLRejected, err)
	}
	resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("%w: test delivery answered %d, want 2xx (redirects are refused)", ErrWebhookURLRejected, resp.StatusCode)
	}
	return nil
}

func (e *Engine) webhookEndpointInput(w *webhook.Webhook, secret string) bridge.EndpointInput {
	return bridge.EndpointInput{
		TenantID:    w.AppID.String(),
		URL:         w.URL,
		Description: "authsome webhook " + w.ID.String(),
		Secret:      secret,
		EventTypes:  w.Events,
		Metadata: map[string]string{
			"authsome_webhook_id": w.ID.String(),
			"authsome_app_id":     w.AppID.String(),
			"authsome_env_id":     w.EnvID.String(),
		},
	}
}

// CreateWebhook registers w as a delivering Relay endpoint. The URL must
// be public and answer a signed test delivery; the events must be shaped
// like event types. On return w.Secret holds the signing secret, the only
// time it is shown; the row keeps its hash and the Relay endpoint id.
func (e *Engine) CreateWebhook(ctx context.Context, w *webhook.Webhook) error {
	r, ok := e.endpointRelay()
	if !ok {
		return ErrWebhooksUnavailable
	}
	if err := validateWebhookEvents(w.Events); err != nil {
		return err
	}
	if w.ID.String() == "" {
		w.ID = id.NewWebhookID()
	}
	secret, err := generateWebhookSecret()
	if err != nil {
		return fmt.Errorf("authsome: create webhook: generate secret: %w", err)
	}
	if verifyErr := e.verifyWebhookURL(ctx, w, secret); verifyErr != nil {
		return verifyErr
	}
	now := time.Now()
	if w.CreatedAt.IsZero() {
		w.CreatedAt = now
		w.UpdatedAt = now
	}

	endpointID, err := r.CreateEndpoint(ctx, e.webhookEndpointInput(w, secret))
	if err != nil {
		return fmt.Errorf("authsome: create webhook: %w", err)
	}
	w.RelayEndpointID = endpointID
	w.SecretHash = store.HashToken(secret)
	w.Secret = secret
	if !w.Active {
		if err := r.SetEndpointEnabled(ctx, endpointID, false); err != nil {
			_ = r.DeleteEndpoint(ctx, endpointID) //nolint:errcheck // best-effort undo
			return fmt.Errorf("authsome: create webhook: %w", err)
		}
	}
	if err := e.store.CreateWebhook(ctx, w); err != nil {
		_ = r.DeleteEndpoint(ctx, endpointID) //nolint:errcheck // best-effort undo: no row, no endpoint
		return fmt.Errorf("authsome: create webhook: %w", err)
	}

	e.hooks.Emit(ctx, &hook.Event{
		Action:     hook.ActionWebhookCreate,
		Resource:   hook.ResourceWebhook,
		ResourceID: w.ID.String(),
		Tenant:     w.AppID.String(),
	})
	e.relayEvent(ctx, "webhook.created", w.AppID.String(), map[string]string{
		"webhook_id": w.ID.String(),
		"url":        w.URL,
	})
	return nil
}

// GetWebhook returns a webhook by ID. The secret is never part of it.
func (e *Engine) GetWebhook(ctx context.Context, webhookID id.WebhookID) (*webhook.Webhook, error) {
	return e.store.GetWebhook(ctx, webhookID)
}

// UpdateWebhook mirrors w's URL, events and active state to its Relay
// endpoint and saves the row. A changed URL is checked and test-delivered
// the way a new one is; the secret is unchanged (see RotateWebhookSecret).
func (e *Engine) UpdateWebhook(ctx context.Context, w *webhook.Webhook) error {
	r, ok := e.endpointRelay()
	if !ok {
		return ErrWebhooksUnavailable
	}
	if err := validateWebhookEvents(w.Events); err != nil {
		return err
	}
	current, err := e.store.GetWebhook(ctx, w.ID)
	if err != nil {
		return fmt.Errorf("authsome: update webhook: %w", err)
	}
	if current.RelayEndpointID == "" {
		return fmt.Errorf("authsome: update webhook: %w: not adopted yet", ErrWebhooksUnavailable)
	}
	w.RelayEndpointID, w.SecretHash, w.Secret = current.RelayEndpointID, current.SecretHash, ""
	if w.URL != current.URL {
		// The test delivery is signed with a throwaway secret: the real one
		// is not held here, and the receiver only needs to prove it answers.
		probe, err := generateWebhookSecret()
		if err != nil {
			return fmt.Errorf("authsome: update webhook: %w", err)
		}
		if err := e.verifyWebhookURL(ctx, w, probe); err != nil {
			return err
		}
	}
	// An empty secret on the input keeps Relay's; see bridge.EndpointInput.
	if err := r.UpdateEndpoint(ctx, w.RelayEndpointID, e.webhookEndpointInput(w, "")); err != nil {
		return fmt.Errorf("authsome: update webhook: %w", err)
	}
	if err := r.SetEndpointEnabled(ctx, w.RelayEndpointID, w.Active); err != nil {
		return fmt.Errorf("authsome: update webhook: %w", err)
	}
	w.UpdatedAt = time.Now()
	if err := e.store.UpdateWebhook(ctx, w); err != nil {
		return fmt.Errorf("authsome: update webhook: %w", err)
	}
	e.hooks.Emit(ctx, &hook.Event{
		Action:     hook.ActionWebhookUpdate,
		Resource:   hook.ResourceWebhook,
		ResourceID: w.ID.String(),
		Tenant:     w.AppID.String(),
	})
	return nil
}

// DeleteWebhook removes the Relay endpoint, then the row. A row with no
// endpoint (not adopted yet) is simply removed.
func (e *Engine) DeleteWebhook(ctx context.Context, webhookID id.WebhookID) error {
	current, err := e.store.GetWebhook(ctx, webhookID)
	if err != nil {
		return fmt.Errorf("authsome: delete webhook: %w", err)
	}
	if current.RelayEndpointID != "" {
		r, ok := e.endpointRelay()
		if !ok {
			return ErrWebhooksUnavailable
		}
		if err := r.DeleteEndpoint(ctx, current.RelayEndpointID); err != nil {
			return fmt.Errorf("authsome: delete webhook: %w", err)
		}
	}
	if err := e.store.DeleteWebhook(ctx, webhookID); err != nil {
		return fmt.Errorf("authsome: delete webhook: %w", err)
	}
	e.hooks.Emit(ctx, &hook.Event{
		Action:     hook.ActionWebhookDelete,
		Resource:   hook.ResourceWebhook,
		ResourceID: webhookID.String(),
		Tenant:     current.AppID.String(),
	})
	return nil
}

// ListWebhooks returns all webhooks for an app, without secrets.
func (e *Engine) ListWebhooks(ctx context.Context, appID id.AppID) ([]*webhook.Webhook, error) {
	hooks, err := e.store.ListWebhooks(ctx, appID)
	if err != nil {
		return nil, err
	}
	for _, w := range hooks {
		w.Secret = ""
	}
	return hooks, nil
}

// RotateWebhookSecret gives the webhook's Relay endpoint a new secret and
// returns it, the only time it is shown. Deliveries from this point are
// signed with the new secret; the receiver must switch before the old one
// stops verifying, which is at once.
func (e *Engine) RotateWebhookSecret(ctx context.Context, webhookID id.WebhookID) (string, error) {
	r, ok := e.endpointRelay()
	if !ok {
		return "", ErrWebhooksUnavailable
	}
	w, err := e.store.GetWebhook(ctx, webhookID)
	if err != nil {
		return "", fmt.Errorf("authsome: rotate webhook secret: %w", err)
	}
	if w.RelayEndpointID == "" {
		return "", fmt.Errorf("authsome: rotate webhook secret: %w: not adopted yet", ErrWebhooksUnavailable)
	}
	secret, err := r.RotateEndpointSecret(ctx, w.RelayEndpointID)
	if err != nil {
		return "", fmt.Errorf("authsome: rotate webhook secret: %w", err)
	}
	w.SecretHash = store.HashToken(secret)
	w.Secret = ""
	w.UpdatedAt = time.Now()
	if err := e.store.UpdateWebhook(ctx, w); err != nil {
		return "", fmt.Errorf("authsome: rotate webhook secret: %w", err)
	}
	e.hooks.Emit(ctx, &hook.Event{
		Action:     hook.ActionWebhookUpdate,
		Resource:   hook.ResourceWebhook,
		ResourceID: w.ID.String(),
		Tenant:     w.AppID.String(),
		Metadata:   map[string]string{"secret_rotated": "true"},
	})
	return secret, nil
}

// adoptLegacyWebhooks gives every webhook row written before webhooks
// became Relay endpoints an endpoint of its own. A row that still carries
// its plaintext secret hands that same secret to Relay, so the receiver
// keeps verifying without anyone re-sharing anything; a row without one
// gets a fresh secret and starts disabled, since the secret has nowhere to
// be shown. Rows are rewritten with the hash and no plaintext. The URL is
// not re-verified: it was accepted once, and refusing it now would drop a
// subscription an operator relies on; the safety rules apply from the next
// change to it.
func (e *Engine) adoptLegacyWebhooks(ctx context.Context) {
	r, ok := e.endpointRelay()
	if !ok {
		return
	}
	apps, err := e.store.ListApps(ctx)
	if err != nil {
		e.logger.Warn("authsome: adopting legacy webhooks: list apps", log.String("error", err.Error()))
		return
	}
	var adopted int
	for _, a := range apps {
		hooks, err := e.store.ListWebhooks(ctx, a.ID)
		if err != nil {
			e.logger.Warn("authsome: adopting legacy webhooks: list", log.String("app", a.ID.String()), log.String("error", err.Error()))
			continue
		}
		for _, w := range hooks {
			if w.RelayEndpointID != "" {
				continue
			}
			if err := e.adoptWebhook(ctx, r, w); err != nil {
				e.logger.Warn("authsome: adopting legacy webhook failed", log.String("webhook", w.ID.String()), log.String("error", err.Error()))
				continue
			}
			adopted++
		}
	}
	if adopted > 0 {
		e.logger.Info("authsome: adopted legacy webhooks as relay endpoints", log.Int("adopted", adopted))
	}
}

func (e *Engine) adoptWebhook(ctx context.Context, r bridge.EndpointRelay, w *webhook.Webhook) error {
	secret := w.Secret
	active := w.Active
	if secret == "" {
		fresh, err := generateWebhookSecret()
		if err != nil {
			return err
		}
		secret, active = fresh, false
	}
	endpointID, err := r.CreateEndpoint(ctx, e.webhookEndpointInput(w, secret))
	if err != nil {
		return err
	}
	if !active {
		if err := r.SetEndpointEnabled(ctx, endpointID, false); err != nil {
			return err
		}
	}
	w.RelayEndpointID = endpointID
	w.SecretHash = store.HashToken(secret)
	w.Secret = ""
	w.Active = active
	return e.store.UpdateWebhook(ctx, w)
}
