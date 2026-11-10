package scim

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/organization"
	"github.com/xraph/authsome/store"
)

// adminRoles are the organization roles that may manage SCIM for an org.
//
// Connecting an identity provider decides who gets an account and what they
// can reach, so it sits with the roles that already decide membership rather
// than with anyone who happens to be in the org.
var adminRoles = []organization.MemberRole{
	organization.RoleOwner,
	organization.RoleAdmin,
}

// requireOrgAdmin checks the caller is an owner or admin of orgID.
//
// A caller who is not a member at all gets 404 rather than 403, so an
// outsider probing organization IDs cannot tell which ones exist.
func (p *Plugin) requireOrgAdmin(ctx forge.Context, orgID id.OrgID) error {
	userID, ok := middleware.UserIDFrom(ctx.Context())
	if !ok || userID.Prefix() == "" {
		return forge.Unauthorized("authentication required")
	}
	if p.authStore == nil {
		return forge.Forbidden("organization membership cannot be checked")
	}
	member, err := p.authStore.GetMemberByUserAndOrg(ctx.Context(), userID, orgID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return forge.NotFound("organization not found")
		}
		return forge.NewHTTPError(http.StatusInternalServerError, "look up membership")
	}
	for _, r := range adminRoles {
		if member.Role == r {
			return nil
		}
	}
	return forge.Forbidden("insufficient organization role")
}

// configForCaller loads a config by ID and checks the caller may manage it.
//
// An app-level config (one with no OrgID) has no organization to check a role
// against, so it is not reachable through these routes at all. Managing one
// is an operator job, done where the app itself is configured. Returning 404
// rather than 403 keeps the two cases indistinguishable from outside.
func (p *Plugin) configForCaller(ctx forge.Context, raw string) (*SCIMConfig, error) {
	if p.service == nil {
		return nil, forge.NewHTTPError(http.StatusInternalServerError, "scim service unavailable")
	}
	configID, err := id.ParseSCIMConfigID(raw)
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid config id: %v", err))
	}
	cfg, err := p.service.GetConfig(ctx.Context(), configID)
	if err != nil || cfg == nil {
		return nil, forge.NotFound("scim configuration not found")
	}
	if cfg.OrgID.IsNil() {
		return nil, forge.NotFound("scim configuration not found")
	}
	if err := p.requireOrgAdmin(ctx, cfg.OrgID); err != nil {
		return nil, err
	}
	return cfg, nil
}

// scimBaseURL is what gets pasted into an identity provider.
func (p *Plugin) scimBaseURL() string {
	return strings.TrimRight(p.config.BasePath, "/")
}

func (p *Plugin) handleListConfigs(ctx forge.Context, req *ListSCIMConfigsRequest) (*ConfigListResponse, error) {
	// Distinct names on purpose. Reusing err makes gocritic call the second
	// one a sloppy reassignment, and shadowing it makes govet call it a
	// shadow, so neither linter is happy with the other's fix.
	orgID, parseErr := id.ParseOrgID(req.OrgID)
	if parseErr != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid org id: %v", parseErr))
	}
	if err := p.requireOrgAdmin(ctx, orgID); err != nil {
		return nil, err
	}
	if p.service == nil {
		return nil, forge.NewHTTPError(http.StatusInternalServerError, "scim service unavailable")
	}
	configs, err := p.service.ListConfigsByOrg(ctx.Context(), orgID)
	if err != nil {
		return nil, forge.NewHTTPError(http.StatusInternalServerError, "list scim configurations")
	}
	if configs == nil {
		configs = []*SCIMConfig{}
	}
	return nil, ctx.JSON(http.StatusOK, &ConfigListResponse{Configs: configs})
}

func (p *Plugin) handleCreateConfig(ctx forge.Context, req *CreateSCIMConfigRequest) (*SCIMConfig, error) {
	// Distinct names on purpose. Reusing err makes gocritic call the second
	// one a sloppy reassignment, and shadowing it makes govet call it a
	// shadow, so neither linter is happy with the other's fix.
	orgID, parseErr := id.ParseOrgID(req.OrgID)
	if parseErr != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid org id: %v", parseErr))
	}
	if err := p.requireOrgAdmin(ctx, orgID); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.Name) == "" {
		return nil, forge.BadRequest("name is required")
	}
	if p.service == nil {
		return nil, forge.NewHTTPError(http.StatusInternalServerError, "scim service unavailable")
	}

	// A connection that quietly does nothing is worse than no connection, so
	// a new one is enabled and creates users unless the caller says otherwise.
	cfg := &SCIMConfig{
		ID:          id.NewSCIMConfigID(),
		AppID:       p.configAppID(),
		OrgID:       orgID,
		Name:        strings.TrimSpace(req.Name),
		Enabled:     boolOr(req.Enabled, true),
		AutoCreate:  boolOr(req.AutoCreate, true),
		AutoSuspend: boolOr(req.AutoSuspend, true),
		GroupSync:   boolOr(req.GroupSync, false),
		DefaultRole: req.DefaultRole,
		Metadata:    req.Metadata,
	}
	if err := p.service.CreateConfig(ctx.Context(), cfg); err != nil {
		return nil, forge.NewHTTPError(http.StatusInternalServerError, "create scim configuration")
	}
	return nil, ctx.JSON(http.StatusCreated, cfg)
}

func (p *Plugin) handleGetConfig(ctx forge.Context, req *GetSCIMConfigRequest) (*SCIMConfig, error) {
	cfg, err := p.configForCaller(ctx, req.ConfigID)
	if err != nil {
		return nil, err
	}
	return nil, ctx.JSON(http.StatusOK, cfg)
}

func (p *Plugin) handleUpdateConfig(ctx forge.Context, req *UpdateSCIMConfigRequest) (*SCIMConfig, error) {
	cfg, err := p.configForCaller(ctx, req.ConfigID)
	if err != nil {
		return nil, err
	}
	if req.Name != nil {
		if strings.TrimSpace(*req.Name) == "" {
			return nil, forge.BadRequest("name cannot be empty")
		}
		cfg.Name = strings.TrimSpace(*req.Name)
	}
	if req.Enabled != nil {
		cfg.Enabled = *req.Enabled
	}
	if req.AutoCreate != nil {
		cfg.AutoCreate = *req.AutoCreate
	}
	if req.AutoSuspend != nil {
		cfg.AutoSuspend = *req.AutoSuspend
	}
	if req.GroupSync != nil {
		cfg.GroupSync = *req.GroupSync
	}
	if req.DefaultRole != nil {
		cfg.DefaultRole = *req.DefaultRole
	}
	if req.Metadata != nil {
		cfg.Metadata = req.Metadata
	}
	if err := p.service.UpdateConfig(ctx.Context(), cfg); err != nil {
		return nil, forge.NewHTTPError(http.StatusInternalServerError, "update scim configuration")
	}
	return nil, ctx.JSON(http.StatusOK, cfg)
}

func (p *Plugin) handleDeleteConfig(ctx forge.Context, req *GetSCIMConfigRequest) (*StatusResponse, error) {
	cfg, err := p.configForCaller(ctx, req.ConfigID)
	if err != nil {
		return nil, err
	}
	if err := p.service.DeleteConfig(ctx.Context(), cfg.ID); err != nil {
		return nil, forge.NewHTTPError(http.StatusInternalServerError, "delete scim configuration")
	}
	return nil, ctx.JSON(http.StatusOK, &StatusResponse{Status: "deleted"})
}

func (p *Plugin) handleListTokens(ctx forge.Context, req *ListSCIMTokensRequest) (*TokenListResponse, error) {
	cfg, err := p.configForCaller(ctx, req.ConfigID)
	if err != nil {
		return nil, err
	}
	tokens, err := p.service.ListTokens(ctx.Context(), cfg.ID)
	if err != nil {
		return nil, forge.NewHTTPError(http.StatusInternalServerError, "list scim tokens")
	}
	if tokens == nil {
		tokens = []*Token{}
	}
	return nil, ctx.JSON(http.StatusOK, &TokenListResponse{Tokens: tokens})
}

func (p *Plugin) handleCreateToken(ctx forge.Context, req *CreateSCIMTokenRequest) (*TokenCreatedResponse, error) {
	cfg, err := p.configForCaller(ctx, req.ConfigID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.Name) == "" {
		return nil, forge.BadRequest("name is required")
	}
	if req.ExpiresAt != nil && req.ExpiresAt.Before(time.Now()) {
		return nil, forge.BadRequest("expires_at is in the past")
	}

	plaintext, token, err := p.service.GenerateToken(
		ctx.Context(), cfg.ID, strings.TrimSpace(req.Name), req.ExpiresAt,
	)
	if err != nil {
		return nil, forge.NewHTTPError(http.StatusInternalServerError, "generate scim token")
	}
	return nil, ctx.JSON(http.StatusCreated, &TokenCreatedResponse{
		Token:     token,
		Plaintext: plaintext,
		BaseURL:   p.scimBaseURL(),
	})
}

func (p *Plugin) handleRotateToken(ctx forge.Context, req *RotateSCIMTokenRequest) (*TokenCreatedResponse, error) {
	tokenID, err := p.tokenForCaller(ctx, req.TokenID)
	if err != nil {
		return nil, err
	}
	if req.GraceSeconds < 0 {
		return nil, forge.BadRequest("grace_seconds cannot be negative")
	}

	plaintext, token, err := p.service.RotateToken(
		ctx.Context(), tokenID, strings.TrimSpace(req.Name),
		time.Duration(req.GraceSeconds)*time.Second,
	)
	if err != nil {
		return nil, forge.NewHTTPError(http.StatusInternalServerError, "rotate scim token")
	}
	return nil, ctx.JSON(http.StatusCreated, &TokenCreatedResponse{
		Token:     token,
		Plaintext: plaintext,
		BaseURL:   p.scimBaseURL(),
	})
}

func (p *Plugin) handleRevokeToken(ctx forge.Context, req *RevokeSCIMTokenRequest) (*StatusResponse, error) {
	tokenID, err := p.tokenForCaller(ctx, req.TokenID)
	if err != nil {
		return nil, err
	}
	if err := p.service.RevokeToken(ctx.Context(), tokenID); err != nil {
		return nil, forge.NewHTTPError(http.StatusInternalServerError, "revoke scim token")
	}
	return nil, ctx.JSON(http.StatusOK, &StatusResponse{Status: "revoked"})
}

// tokenForCaller resolves a token ID and checks the caller may manage the
// configuration it belongs to. Without this a token ID alone would be enough
// to rotate or revoke someone else's connection.
func (p *Plugin) tokenForCaller(ctx forge.Context, raw string) (id.SCIMTokenID, error) {
	if p.service == nil {
		return id.SCIMTokenID{}, forge.NewHTTPError(http.StatusInternalServerError, "scim service unavailable")
	}
	tokenID, err := id.ParseSCIMTokenID(raw)
	if err != nil {
		return id.SCIMTokenID{}, forge.BadRequest(fmt.Sprintf("invalid token id: %v", err))
	}
	tok, err := p.scimStore.GetToken(ctx.Context(), tokenID)
	if err != nil || tok == nil {
		return id.SCIMTokenID{}, forge.NotFound("scim token not found")
	}
	// The configuration itself is not needed here, only the check that the
	// caller may manage it.
	if _, err = p.configForCaller(ctx, tok.ConfigID.String()); err != nil {
		return id.SCIMTokenID{}, err
	}
	return tokenID, nil
}

// configAppID is the app a new configuration belongs to. SCIM configs are
// per-organization, and an organization belongs to exactly one app, so the
// engine's default is the right answer for every deployment that has one.
func (p *Plugin) configAppID() id.AppID {
	appID, err := id.ParseAppID(p.defaultAppID)
	if err != nil {
		return id.AppID{}
	}
	return appID
}

func boolOr(v *bool, fallback bool) bool {
	if v == nil {
		return fallback
	}
	return *v
}
