package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/xraph/forge"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/apitypes"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/principal"
)

// issuedTokenTypeAccessToken is the RFC 8693 §3 URN for an OAuth 2.0 access
// token, the only token type this engine issues from an exchange.
const issuedTokenTypeAccessToken = "urn:ietf:params:oauth:token-type:access_token"

// ──────────────────────────────────────────────────
// Principal / delegation route registration
// ──────────────────────────────────────────────────

func (a *API) registerPrincipalRoutes(router forge.Router) error {
	g := router.Group("/v1",
		forge.WithGroupTags("principals"),
		forge.WithGroupMiddleware(middleware.RequireAuth()),
		// Declared so the requirement the middleware above enforces reaches
		// the OpenAPI document and the generated clients. Declaring does not
		// enforce: the middleware is still what refuses the request.
		forge.WithGroupAuth("session", "session-cookie", "api-key", "jwt"),
	)

	if err := g.POST("/token/exchange", a.handleTokenExchange,
		forge.WithSummary("Exchange a credential for a delegated session"),
		forge.WithDescription("Mints a session in which the calling principal acts on behalf of another, against an existing delegation grant (RFC 8693 shaped token exchange). Returns 403 when no live grant matches the caller and the requested subject, or when the requested scopes are not all inside the grant."),
		forge.WithOperationID("exchangeToken"),
		forge.WithResponseSchema(http.StatusOK, "Delegated session", TokenExchangeResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.GET("/principals/me/delegations", a.handleListMyDelegations,
		forge.WithSummary("List what may act on your behalf"),
		forge.WithDescription("Lists the live delegation grants naming the calling principal as subject, so it can see the agents holding authority over it. There is no revoke route yet; Engine.RevokeDelegation is reachable only from engine-embedding code today."),
		forge.WithOperationID("listMyDelegations"),
		forge.WithResponseSchema(http.StatusOK, "Delegations", DelegationListResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	return g.POST("/principals/:id/children", a.handleMintChild,
		forge.WithSummary("Mint an ephemeral child principal"),
		forge.WithDescription("Creates a short-lived principal under the calling parent, with its own credential. Scopes must be a subset of the parent's and the TTL is capped by the parent's expiry."),
		forge.WithOperationID("mintChildPrincipal"),
		forge.WithResponseSchema(http.StatusCreated, "Child principal", MintChildResponse{}),
		forge.WithErrorResponses(),
	)
}

// ──────────────────────────────────────────────────
// Handlers
// ──────────────────────────────────────────────────

func (a *API) handleTokenExchange(ctx forge.Context, req *TokenExchangeRequest) (*TokenExchangeResponse, error) {
	caller, ok := middleware.PrincipalFrom(ctx.Context())
	if !ok {
		return nil, forge.Unauthorized("authentication required")
	}

	appID, ok := a.callerAppID(ctx)
	if !ok {
		return nil, forge.Unauthorized("authentication required")
	}

	if req.Subject == "" {
		return nil, forge.BadRequest("subject is required")
	}
	subjectUserID, err := id.ParseUserID(req.Subject)
	if err != nil {
		return nil, forge.BadRequest("invalid subject: must be a user id")
	}

	// A caller presenting an already-delegated or already-impersonated
	// session must not be allowed to exchange again: that would let it
	// launder the authority of whoever it is currently acting for into a
	// grant it holds against some third party, escalating past what either
	// grant alone would permit. ExchangeToken refuses this outright, but it
	// can only see it if the calling session's own actor chain reaches it.
	var callerActors principal.Chain
	var callerDelegationID id.DelegationID
	if callerSession, ok := middleware.SessionFrom(ctx.Context()); ok && callerSession != nil {
		callerActors = callerSession.Actors
		callerDelegationID = callerSession.DelegationID
	}

	httpReq := ctx.Request()
	sess, err := a.engine.ExchangeToken(ctx.Context(), &authsome.ExchangeRequest{
		AppID:              appID,
		Actor:              caller.Ref,
		RequestedSubject:   principal.UserRef(subjectUserID),
		Scopes:             req.Scopes,
		IPAddress:          clientIPFromRequest(httpReq),
		UserAgent:          httpReq.UserAgent(),
		CallerActors:       callerActors,
		CallerDelegationID: callerDelegationID,
	})
	if err != nil {
		// This endpoint never creates authority: every reason ExchangeToken
		// declines (no live grant, wrong actor, disabled actor, an
		// out-of-grant scope, a chained exchange) is the same refusal from
		// the caller's point of view, and reads as 403 here rather than 404
		// or 400: the grant might exist for a different actor or subject,
		// so "not found" would overclaim, and the request itself was
		// well-formed.
		if errors.Is(err, authsome.ErrExchangeRefused) {
			return nil, forge.Forbidden(err.Error())
		}
		return nil, mapError(err)
	}

	expiresIn := int64(0)
	if remaining := time.Until(sess.ExpiresAt); remaining > 0 {
		expiresIn = int64(remaining.Seconds())
	}

	// The minted token goes in the response body only.
	return &TokenExchangeResponse{
		AccessToken:     sess.Token,
		IssuedTokenType: issuedTokenTypeAccessToken,
		TokenType:       "Bearer",
		ExpiresIn:       expiresIn,
		Scopes:          sess.Scopes,
		Subject:         req.Subject,
		Actor:           caller.Ref.String(),
	}, nil
}

func (a *API) handleListMyDelegations(ctx forge.Context, _ *apitypes.Empty) (*DelegationListResponse, error) {
	caller, ok := middleware.PrincipalFrom(ctx.Context())
	if !ok {
		return nil, forge.Unauthorized("authentication required")
	}

	appID, ok := a.callerAppID(ctx)
	if !ok {
		return nil, forge.Unauthorized("authentication required")
	}

	delegations, err := a.engine.ListDelegationsForSubject(ctx.Context(), appID, caller.Ref)
	if err != nil {
		return nil, mapError(err)
	}

	resp := &DelegationListResponse{Delegations: make([]DelegationResponse, 0, len(delegations))}
	for _, d := range delegations {
		item := DelegationResponse{
			ID:        d.ID.String(),
			Actor:     d.Actor.String(),
			Subject:   d.Subject.String(),
			GrantKind: string(d.GrantKind),
			Scopes:    d.Scopes,
			GrantedBy: d.GrantedBy.String(),
			CreatedAt: d.CreatedAt.Format(time.RFC3339),
		}
		if d.ExpiresAt != nil {
			item.ExpiresAt = d.ExpiresAt.Format(time.RFC3339)
		}
		resp.Delegations = append(resp.Delegations, item)
	}
	return resp, nil
}

// handleMintChild mints an ephemeral child principal under the calling
// parent.
//
// The path id names whose children are being created, but the parent comes
// from the request context, never from the body: req.ID is checked against
// the authenticated caller and refused on any mismatch, so a principal can
// only ever mint children under itself, not under some other id it happens
// to know.
func (a *API) handleMintChild(ctx forge.Context, req *MintChildRequest) (*MintChildResponse, error) {
	caller, ok := middleware.PrincipalFrom(ctx.Context())
	if !ok {
		return nil, forge.Unauthorized("authentication required")
	}
	if req.ID != caller.ID {
		return nil, forge.Forbidden("can only mint children under your own principal")
	}
	if req.Name == "" {
		return nil, forge.BadRequest("name is required")
	}
	if req.TTLSeconds <= 0 {
		return nil, forge.BadRequest("ttl_seconds must be positive")
	}

	parentID, err := id.ParseServiceAccountID(req.ID)
	if err != nil {
		return nil, forge.BadRequest("invalid principal id")
	}

	child, key, secret, err := a.engine.MintChildPrincipal(ctx.Context(), parentID,
		req.Name, req.Scopes, time.Duration(req.TTLSeconds)*time.Second)
	if err != nil {
		return nil, mapError(err)
	}

	return &MintChildResponse{
		ID:        child.ID.String(),
		ParentID:  child.ParentID.String(),
		Name:      child.Name,
		Scopes:    child.Scopes,
		ExpiresAt: child.ExpiresAt,
		Key:       secret,
		KeyPrefix: key.KeyPrefix,
		CreatedAt: child.CreatedAt,
	}, nil
}
