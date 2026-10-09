package authsome_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/warden"
	wardenmem "github.com/xraph/warden/store/memory"

	"github.com/xraph/authsome"
	"github.com/xraph/authsome/app"
	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/store"
	"github.com/xraph/authsome/store/memory"
	"github.com/xraph/authsome/webhook"
)

// receiver is a webhook receiver that records what it was sent.
type receiver struct {
	mu       sync.Mutex
	status   int
	requests []recorded
}

type recorded struct {
	body      []byte
	timestamp string
	signature string
	eventType string
}

func newReceiver(t *testing.T, status int) (*receiver, *httptest.Server) {
	t.Helper()
	r := &receiver{status: status}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		body, _ := io.ReadAll(req.Body)
		r.mu.Lock()
		r.requests = append(r.requests, recorded{
			body: body, timestamp: req.Header.Get(webhook.RelayTimestampHeader),
			signature: req.Header.Get(webhook.RelaySignatureHeader), eventType: req.Header.Get("X-Relay-Event-Type"),
		})
		r.mu.Unlock()
		if r.status == http.StatusFound {
			http.Redirect(w, req, "/elsewhere", http.StatusFound)
			return
		}
		w.WriteHeader(r.status)
	}))
	t.Cleanup(srv.Close)
	return r, srv
}

func (r *receiver) last() recorded {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.requests[len(r.requests)-1]
}

func webhookTestConfig() authsome.Config {
	cfg := testEngineConfig()
	cfg.Webhooks.AllowInsecureURLs = true // the receivers live on loopback
	return cfg
}

func TestCreateWebhook_RegistersRelayEndpointAndShowsSecretOnce(t *testing.T) {
	relay := bridge.NewMemoryRelay()
	eng, st := newTestEngine(t, authsome.WithConfig(webhookTestConfig()), authsome.WithEventRelay(relay))
	ctx := context.Background()
	appID, err := id.ParseAppID("aapp_01jf0000000000000000000000")
	require.NoError(t, err)
	rcv, srv := newReceiver(t, http.StatusNoContent)

	w := &webhook.Webhook{AppID: appID, URL: srv.URL + "/hook", Events: []string{"user.created", "auth.*"}, Active: true}
	require.NoError(t, eng.CreateWebhook(ctx, w))

	require.NotEmpty(t, w.Secret, "the secret is shown on the way out")
	assert.Equal(t, store.HashToken(w.Secret), w.SecretHash)
	require.NotEmpty(t, w.RelayEndpointID)
	ep, ok := relay.Endpoint(w.RelayEndpointID)
	require.True(t, ok, "a relay endpoint was created")
	assert.Equal(t, w.Secret, ep.Input.Secret, "relay signs with the secret the caller was shown")
	assert.Equal(t, appID.String(), ep.Input.TenantID, "the relay tenant is the app")
	assert.Equal(t, srv.URL+"/hook", ep.Input.URL)
	assert.Equal(t, []string{"user.created", "auth.*"}, ep.Input.EventTypes)
	assert.Equal(t, w.ID.String(), ep.Input.Metadata["authsome_webhook_id"])
	assert.True(t, ep.Enabled)

	got, err := st.GetWebhook(ctx, w.ID)
	require.NoError(t, err)
	assert.Empty(t, got.Secret, "the row holds no secret")
	assert.Equal(t, w.SecretHash, got.SecretHash)
	assert.Equal(t, w.RelayEndpointID, got.RelayEndpointID)

	delivered := rcv.last()
	assert.Equal(t, "webhook.test", delivered.eventType, "registration made one signed test delivery")
	assert.NoError(t, webhook.VerifyRelay(delivered.body, w.Secret, delivered.timestamp, delivered.signature, 0),
		"the test delivery is signed the way relay will sign, with the shown secret")

	listed, err := eng.ListWebhooks(ctx, appID)
	require.NoError(t, err)
	require.Len(t, listed, 1)
	assert.Empty(t, listed[0].Secret)
}

func TestCreateWebhook_RefusesBadURLsAndEvents(t *testing.T) {
	relay := bridge.NewMemoryRelay()
	eng, _ := newTestEngine(t, authsome.WithConfig(webhookTestConfig()), authsome.WithEventRelay(relay))
	ctx := context.Background()
	appID, err := id.ParseAppID("aapp_01jf0000000000000000000000")
	require.NoError(t, err)

	_, redirecting := newReceiver(t, http.StatusFound)
	err = eng.CreateWebhook(ctx, &webhook.Webhook{AppID: appID, URL: redirecting.URL, Events: []string{"user.created"}})
	assert.ErrorIs(t, err, authsome.ErrWebhookURLRejected, "a receiver that redirects is refused")

	_, failing := newReceiver(t, http.StatusInternalServerError)
	err = eng.CreateWebhook(ctx, &webhook.Webhook{AppID: appID, URL: failing.URL, Events: []string{"user.created"}})
	assert.ErrorIs(t, err, authsome.ErrWebhookURLRejected, "a receiver that fails the test delivery is refused")

	_, ok := newReceiver(t, http.StatusOK)
	err = eng.CreateWebhook(ctx, &webhook.Webhook{AppID: appID, URL: ok.URL, Events: nil})
	assert.ErrorIs(t, err, authsome.ErrWebhookEvents)
	err = eng.CreateWebhook(ctx, &webhook.Webhook{AppID: appID, URL: ok.URL, Events: []string{"not an event"}})
	assert.ErrorIs(t, err, authsome.ErrWebhookEvents)
	err = eng.CreateWebhook(ctx, &webhook.Webhook{AppID: appID, URL: "ftp://x.example/", Events: []string{"user.created"}})
	assert.ErrorIs(t, err, authsome.ErrWebhookURLRejected)
	assert.Zero(t, relay.EndpointCount(), "nothing reached relay")

	strict := bridge.NewMemoryRelay()
	strictEng, _ := newTestEngine(t, authsome.WithEventRelay(strict))
	err = strictEng.CreateWebhook(ctx, &webhook.Webhook{AppID: appID, URL: ok.URL, Events: []string{"user.created"}})
	assert.ErrorIs(t, err, authsome.ErrWebhookURLRejected, "outside insecure mode a loopback receiver is refused")
	err = strictEng.CreateWebhook(ctx, &webhook.Webhook{AppID: appID, URL: "https://169.254.169.254/latest", Events: []string{"user.created"}})
	assert.ErrorIs(t, err, authsome.ErrWebhookURLRejected)
}

func TestWebhooks_UnavailableWithoutEndpointRelay(t *testing.T) {
	eng, _ := newTestEngine(t)
	ctx := context.Background()
	appID, err := id.ParseAppID("aapp_01jf0000000000000000000000")
	require.NoError(t, err)
	err = eng.CreateWebhook(ctx, &webhook.Webhook{AppID: appID, URL: "https://receiver.example/hook", Events: []string{"user.created"}})
	assert.ErrorIs(t, err, authsome.ErrWebhooksUnavailable)
	_, err = eng.RotateWebhookSecret(ctx, id.NewWebhookID())
	assert.ErrorIs(t, err, authsome.ErrWebhooksUnavailable)
}

func TestUpdateDeleteRotateWebhook_MirrorToRelay(t *testing.T) {
	relay := bridge.NewMemoryRelay()
	eng, st := newTestEngine(t, authsome.WithConfig(webhookTestConfig()), authsome.WithEventRelay(relay))
	ctx := context.Background()
	appID, err := id.ParseAppID("aapp_01jf0000000000000000000000")
	require.NoError(t, err)
	_, first := newReceiver(t, http.StatusOK)
	second, secondSrv := newReceiver(t, http.StatusOK)

	w := &webhook.Webhook{AppID: appID, URL: first.URL, Events: []string{"user.created"}, Active: true}
	require.NoError(t, eng.CreateWebhook(ctx, w))
	original := w.Secret

	w.URL, w.Events, w.Active = secondSrv.URL+"/v2", []string{"user.*"}, false
	require.NoError(t, eng.UpdateWebhook(ctx, w))
	ep, _ := relay.Endpoint(w.RelayEndpointID)
	assert.Equal(t, secondSrv.URL+"/v2", ep.Input.URL)
	assert.Equal(t, []string{"user.*"}, ep.Input.EventTypes)
	assert.Equal(t, original, ep.Input.Secret, "an update keeps the secret")
	assert.False(t, ep.Enabled, "active mirrors to the endpoint")
	assert.Len(t, second.requests, 1, "a changed URL is test-delivered")
	got, err := st.GetWebhook(ctx, w.ID)
	require.NoError(t, err)
	assert.Equal(t, store.HashToken(original), got.SecretHash)
	assert.Empty(t, got.Secret)

	_, redirecting := newReceiver(t, http.StatusFound)
	w.URL = redirecting.URL
	assert.ErrorIs(t, eng.UpdateWebhook(ctx, w), authsome.ErrWebhookURLRejected)

	rotated, err := eng.RotateWebhookSecret(ctx, w.ID)
	require.NoError(t, err)
	assert.NotEqual(t, original, rotated)
	ep, _ = relay.Endpoint(got.RelayEndpointID)
	assert.Equal(t, rotated, ep.Input.Secret)
	got, err = st.GetWebhook(ctx, w.ID)
	require.NoError(t, err)
	assert.Equal(t, store.HashToken(rotated), got.SecretHash)
	assert.Empty(t, got.Secret)

	require.NoError(t, eng.DeleteWebhook(ctx, w.ID))
	_, ok := relay.Endpoint(got.RelayEndpointID)
	assert.False(t, ok, "the relay endpoint goes with the row")
	_, err = st.GetWebhook(ctx, w.ID)
	assert.ErrorIs(t, err, store.ErrNotFound)
}

// A webhook row written before webhooks became relay endpoints gets one at
// start, signed with the secret its receiver already holds; a row with no
// secret gets a fresh one and starts disabled.
func TestStart_AdoptsLegacyWebhooks(t *testing.T) {
	s := memory.New()
	appID, err := id.ParseAppID("aapp_01jf0000000000000000000000")
	require.NoError(t, err)
	now := time.Now()
	require.NoError(t, s.CreateApp(context.Background(), &app.App{
		ID: appID, Name: "Platform", Slug: "platform", PublishableKey: "pk_test_authsome_root_default", IsPlatform: true,
		CreatedAt: now, UpdatedAt: now,
	}))
	ctx := context.Background()
	withSecret := &webhook.Webhook{
		ID: id.NewWebhookID(), AppID: appID, URL: "https://receiver.example/hook", Events: []string{"user.created"},
		Secret: "whsec_legacy", Active: true, CreatedAt: now, UpdatedAt: now,
	}
	require.NoError(t, s.CreateWebhook(ctx, withSecret))
	withoutSecret := &webhook.Webhook{
		ID: id.NewWebhookID(), AppID: appID, URL: "https://receiver.example/other", Events: []string{"user.created"},
		Active: true, CreatedAt: now, UpdatedAt: now,
	}
	require.NoError(t, s.CreateWebhook(ctx, withoutSecret))

	wardenEng, err := warden.NewEngine(warden.WithStore(wardenmem.New()))
	require.NoError(t, err)
	relay := bridge.NewMemoryRelay()
	eng, err := authsome.NewEngine(
		authsome.WithStore(s), authsome.WithWarden(wardenEng), authsome.WithChronicle(bridge.NewMemoryChronicle()),
		authsome.WithDisableMigrate(), authsome.WithConfig(testEngineConfig()), authsome.WithAppID(appID.String()),
		authsome.WithEventRelay(relay),
	)
	require.NoError(t, err)
	require.NoError(t, eng.Start(ctx))
	t.Cleanup(func() { _ = eng.Stop(ctx) })

	require.Eventually(t, func() bool {
		a, errA := s.GetWebhook(ctx, withSecret.ID)
		b, errB := s.GetWebhook(ctx, withoutSecret.ID)
		return errA == nil && errB == nil && a.RelayEndpointID != "" && b.RelayEndpointID != ""
	}, 5*time.Second, 10*time.Millisecond, "adoption runs in the background after Start")

	a, err := s.GetWebhook(ctx, withSecret.ID)
	require.NoError(t, err)
	ep, ok := relay.Endpoint(a.RelayEndpointID)
	require.True(t, ok)
	assert.Equal(t, "whsec_legacy", ep.Input.Secret, "the receiver's existing secret is handed to relay")
	assert.True(t, ep.Enabled)
	assert.Empty(t, a.Secret, "the plaintext is gone from the row")
	assert.Equal(t, store.HashToken("whsec_legacy"), a.SecretHash)

	b, err := s.GetWebhook(ctx, withoutSecret.ID)
	require.NoError(t, err)
	ep, ok = relay.Endpoint(b.RelayEndpointID)
	require.True(t, ok)
	assert.NotEmpty(t, ep.Input.Secret)
	assert.False(t, ep.Enabled, "no secret to show means the webhook starts disabled")
	assert.False(t, b.Active)
}
