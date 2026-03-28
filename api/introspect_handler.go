package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/xraph/forge"
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

func (a *API) handleIntrospect(_ forge.Context, req *IntrospectRequest) (*IntrospectResponse, error) {
	if req.Token == "" {
		return nil, forge.BadRequest("token is required")
	}

	inactive := &IntrospectResponse{Active: false}

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
