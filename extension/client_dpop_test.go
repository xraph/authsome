package extension

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/dpop"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/dpoptest"
	authclient "github.com/xraph/authsome/sdk/go"
)

// newClientModeRouter builds a client-mode extension pointed at a stub
// identity server that reports token as active and bound to jkt, then applies
// whatever Middlewares() returns to a single route. It exercises the wiring
// an operator actually gets, not a hand-assembled middleware.
func newClientModeRouter(t *testing.T, token, jkt string) forge.Router {
	t.Helper()

	userID := id.NewUserID().String()

	identity := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Token string `json:"token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode introspect request: %v", err)
			return
		}

		resp := authclient.IntrospectResponse{}
		if body.Token == token {
			resp.Active = true
			resp.UserID = userID
			resp.AppID = id.NewAppID().String()
			resp.SessionID = id.NewSessionID().String()
			resp.Cnf = &authclient.IntrospectConfirmation{Jkt: jkt}
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("encode introspect response: %v", err)
		}
	}))
	t.Cleanup(identity.Close)

	e := newTestExtension(identity.URL)
	e.client = authclient.NewClient(identity.URL)

	router := forge.NewRouter()
	for _, mw := range e.Middlewares() {
		router.Use(mw)
	}
	if err := router.GET("/test", func(ctx forge.Context) error {
		return ctx.NoContent(http.StatusOK)
	}); err != nil {
		t.Fatalf("register route: %v", err)
	}
	return router
}

// TestClientMode_BoundTokenWithoutProofIs401 pins the rule an operator gets
// from the extension itself: a DPoP-bound token presented to a client-mode
// service with no proof is refused.
func TestClientMode_BoundTokenWithoutProofIs401(t *testing.T) {
	t.Parallel()

	key := dpoptest.Key(t)
	router := newClientModeRouter(t, "bound-token", dpoptest.Thumbprint(t, key))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.com/test", nil)
	req.Header.Set("Authorization", "DPoP bound-token")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 for a bound token with no proof", rec.Code)
	}
}

// TestClientMode_BoundTokenWithValidProofPasses is the other half: the
// extension has to hand the middleware a validator, or client mode would
// refuse every bound token including the legitimate client's.
func TestClientMode_BoundTokenWithValidProofPasses(t *testing.T) {
	t.Parallel()

	key := dpoptest.Key(t)
	router := newClientModeRouter(t, "bound-token", dpoptest.Thumbprint(t, key))

	claims := dpoptest.ValidClaims(http.MethodGet, "http://example.com/test")
	claims["ath"] = dpop.AccessTokenHash("bound-token")

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.com/test", nil)
	req.Header.Set("Authorization", "DPoP bound-token")
	req.Header.Set("DPoP", dpoptest.MintProof(t, key, "ES256", claims))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 for a bound token with a valid proof", rec.Code)
	}
}
