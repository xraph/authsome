package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/apikey"
)

// ──────────────────────────────────────────────────
// Token introspection (RFC 7662)
// ──────────────────────────────────────────────────

func (a *API) registerIntrospectRoutes(router forge.Router) error {
	g := router.Group("/v1", forge.WithGroupTags("introspection"))
	rlCfg := a.engine.Config().RateLimit

	introspectOpts := make([]forge.RouteOption, 0, 7) //nolint:mnd // base options + rate limit
	introspectOpts = append(introspectOpts,
		forge.WithSummary("Introspect token"),
		forge.WithDescription("Validates a token and returns the associated identity. Follows RFC 7662 semantics: invalid tokens return {active: false} with 200 status."),
		forge.WithOperationID("introspectToken"),
		forge.WithRequestSchema(IntrospectRequest{}),
		forge.WithResponseSchema(http.StatusOK, "Token introspection result", IntrospectResponse{}),
		forge.WithErrorResponses(),
	)
	introspectOpts = append(introspectOpts, a.rateLimitOpt(rlCfg.IntrospectLimit)...)
	return g.POST("/introspect", a.handleIntrospect, introspectOpts...)
}

func (a *API) handleIntrospect(ctx forge.Context, req *IntrospectRequest) (*IntrospectResponse, error) {
	if req.Token == "" {
		return nil, forge.BadRequest("token is required")
	}

	inactive := &IntrospectResponse{Active: false}

	// Try API key (machine-to-machine auth) before session/JWT paths.
	// Identified by their marker prefix (ask_, sk_*). Public keys (pk_*) are
	// rejected — they are not for authentication.
	if apikey.IsAPIKey(req.Token) {
		if apikey.IsPublicKey(req.Token) {
			return inactive, nil
		}
		return a.introspectAPIKey(ctx, req.Token, inactive)
	}

	// Try JWT first (stateless validation)
	if isJWT(req.Token) {
		claims, err := a.engine.ValidateJWT(req.Token)
		if err != nil {
			return inactive, nil //nolint:nilerr // RFC 7662: invalid token → active=false
		}

		resp := &IntrospectResponse{
			Active:    true,
			UserID:    claims.UserID,
			AppID:     claims.AppID,
			OrgID:     claims.OrgID,
			EnvID:     claims.EnvID,
			SessionID: claims.SessionID,
			ExpiresAt: claims.ExpiresAt.Format(time.RFC3339),
		}

		// Optionally resolve user details
		if claims.UserID != "" {
			if u, err := a.engine.ResolveUser(claims.UserID); err == nil {
				resp.User = &IntrospectUser{
					ID:        u.ID.String(),
					Email:     u.Email,
					FirstName: u.FirstName,
					LastName:  u.LastName,
					Username:  u.Username,
				}
			}
		}

		return resp, nil
	}

	// Opaque session token
	sess, err := a.engine.ResolveSessionByToken(req.Token)
	if err != nil {
		return inactive, nil //nolint:nilerr // RFC 7662: invalid token → active=false
	}

	resp := &IntrospectResponse{
		Active:    true,
		UserID:    sess.UserID.String(),
		AppID:     sess.AppID.String(),
		SessionID: sess.ID.String(),
		ExpiresAt: sess.ExpiresAt.Format(time.RFC3339),
	}

	if sess.OrgID.String() != "" {
		resp.OrgID = sess.OrgID.String()
	}
	if sess.EnvID.String() != "" {
		resp.EnvID = sess.EnvID.String()
	}

	// Resolve user details
	if u, resolveErr := a.engine.ResolveUser(sess.UserID.String()); resolveErr == nil {
		resp.User = &IntrospectUser{
			ID:        u.ID.String(),
			Email:     u.Email,
			FirstName: u.FirstName,
			LastName:  u.LastName,
			Username:  u.Username,
		}
	}

	return resp, nil
}

// isJWT detects JWT tokens by the presence of two dots (header.payload.signature).
func isJWT(token string) bool {
	return strings.Count(token, ".") == 2
}

// introspectAPIKey validates a secret API key against the API key store and
// builds an IntrospectResponse mirroring the strategy-based middleware path.
//
// Lookup is appID-agnostic: the keysmith-backed store ignores its appID
// parameter and resolves by prefix alone, which is what introspection needs
// since the caller doesn't know the appID up-front. The verified key carries
// its own AppID, EnvID, and UserID, which we surface in the response.
func (a *API) introspectAPIKey(ctx forge.Context, token string, inactive *IntrospectResponse) (*IntrospectResponse, error) {
	store := a.engine.APIKeyStore()
	if store == nil {
		return inactive, nil
	}

	prefix := extractAPIKeyPrefix(token)
	if prefix == "" {
		return inactive, nil
	}

	key, err := store.FindByPrefix(ctx.Context(), prefix)
	if err != nil || key == nil {
		return inactive, nil //nolint:nilerr // RFC 7662: invalid token → active=false
	}
	if !apikey.VerifyKey(token, key.KeyHash) {
		return inactive, nil
	}
	if !key.IsValid() {
		return inactive, nil
	}

	// Best-effort touch of last-used so dashboards can surface activity. We
	// deliberately ignore the error — introspection must remain fast and a
	// store hiccup shouldn't fail a valid auth.
	now := time.Now()
	key.LastUsedAt = &now
	_ = store.UpdateAPIKey(ctx.Context(), key) //nolint:errcheck // best-effort

	resp := &IntrospectResponse{
		Active:    true,
		AppID:     key.AppID.String(),
		UserID:    key.UserID.String(),
		EnvID:     key.EnvID.String(),
		ExpiresAt: "",
	}
	if key.ExpiresAt != nil {
		resp.ExpiresAt = key.ExpiresAt.Format(time.RFC3339)
	}

	if key.UserID.String() != "" {
		if u, resolveErr := a.engine.ResolveUser(key.UserID.String()); resolveErr == nil {
			resp.User = &IntrospectUser{
				ID:        u.ID.String(),
				Email:     u.Email,
				FirstName: u.FirstName,
				LastName:  u.LastName,
				Username:  u.Username,
			}
		}
	}

	return resp, nil
}

// extractAPIKeyPrefix returns the lookup prefix for a raw API key. Mirrors
// plugins/apikey extractPrefix but lives here so the introspect handler
// doesn't depend on the plugin (which may not be registered).
func extractAPIKeyPrefix(raw string) string {
	const prefixHexLen = 8
	markers := []string{
		"sk_test_", "sk_stg_", "sk_live_",
		"pk_test_", "pk_stg_", "pk_live_",
		"ask_",
	}
	for _, m := range markers {
		if strings.HasPrefix(raw, m) {
			need := len(m) + prefixHexLen
			if len(raw) < need {
				return ""
			}
			return raw[:need]
		}
	}
	return ""
}
