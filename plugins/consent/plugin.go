package consent

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/authprovider"
	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/hook"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/plugin"

	"github.com/xraph/grove/migrate"
)

// Compile-time interface checks.
var (
	_ plugin.Plugin                = (*Plugin)(nil)
	_ plugin.OnInit                = (*Plugin)(nil)
	_ plugin.RouteProvider         = (*Plugin)(nil)
	_ plugin.DataExportContributor = (*Plugin)(nil)
	_ plugin.MigrationProvider     = (*Plugin)(nil)
)

// Plugin is the consent tracking plugin.
type Plugin struct {
	engine      plugin.Engine
	store       Store
	hooks       *hook.Bus
	relay       bridge.EventRelay
	chronicle   bridge.Chronicle
	logger      log.Logger
	basePath    string
	permChecker plugin.PermissionChecker
}

// New creates a new consent plugin. An in-memory consent store is used by
// default. Use SetConsentStore to inject a persistent store.
func New() *Plugin {
	return &Plugin{
		store: NewMemoryStore(),
	}
}

// Name returns the plugin name.
func (p *Plugin) Name() string { return "consent" }

// SetConsentStore allows direct consent store injection.
func (p *Plugin) SetConsentStore(s Store) { p.store = s }

// MigrationGroups returns the consent migration groups for the given driver.
func (p *Plugin) MigrationGroups(driverName string) []*migrate.Group {
	switch driverName {
	case "pg", "postgres":
		return []*migrate.Group{PostgresMigrations}
	case "sqlite", "sqlite3":
		return []*migrate.Group{SqliteMigrations}
	default:
		return nil
	}
}

// OnInit captures engine capabilities.
func (p *Plugin) OnInit(_ context.Context, engine plugin.Engine) error {
	p.engine = engine
	p.hooks = engine.Hooks()
	p.relay = engine.Relay()
	p.chronicle = engine.Chronicle()
	p.logger = engine.Logger()

	p.basePath = "/v1"

	if pc, ok := engine.(plugin.PermissionChecker); ok {
		p.permChecker = pc
	}

	return nil
}

// RegisterRoutes registers consent HTTP routes.
func (p *Plugin) RegisterRoutes(router forge.Router) error {
	g := router.Group(p.basePath+"/consent",
		forge.WithGroupTags("consent"),
		forge.WithGroupAuth("session"),
		forge.WithGroupMiddleware(authprovider.RegistryMiddleware(p.engine.AuthRegistry(), "session")),
	)

	if err := g.POST("/grant", p.handleGrant,
		forge.WithSummary("Grant consent"),
		forge.WithDescription("Records consent for a specific purpose and policy version."),
		forge.WithOperationID("grantConsent"),
		forge.WithRequestSchema(GrantConsentRequest{}),
		forge.WithResponseSchema(http.StatusOK, "Consent granted", Consent{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.POST("/revoke", p.handleRevoke,
		forge.WithSummary("Revoke consent"),
		forge.WithDescription("Revokes previously granted consent for a specific purpose."),
		forge.WithOperationID("revokeConsent"),
		forge.WithRequestSchema(RevokeConsentRequest{}),
		forge.WithResponseSchema(http.StatusOK, "Consent revoked", StatusResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	return g.GET("", p.handleList,
		forge.WithSummary("List consents"),
		forge.WithDescription("Returns all consent records for the authenticated user."),
		forge.WithOperationID("listConsents"),
		forge.WithResponseSchema(http.StatusOK, "Consent list", ListResponse{}),
		forge.WithErrorResponses(),
	)
}

// ExportUserData returns the user's consent records for GDPR data export.
func (p *Plugin) ExportUserData(ctx context.Context, userID id.UserID) (category string, data any, err error) {
	consents, _, err := p.store.ListConsents(ctx, &Query{
		UserID: userID,
		Limit:  1000,
	})
	if err != nil {
		return "", nil, fmt.Errorf("consent: export user data: %w", err)
	}
	if len(consents) == 0 {
		return "consents", nil, nil
	}
	return "consents", consents, nil
}

// ──────────────────────────────────────────────────
// Handlers
// ──────────────────────────────────────────────────

func (p *Plugin) handleGrant(ctx forge.Context, req *GrantConsentRequest) (*Consent, error) {
	userID, ok := middleware.UserIDFrom(ctx.Context())
	if !ok {
		return nil, forge.Unauthorized("authentication required")
	}

	if req.Purpose == "" {
		return nil, forge.BadRequest("purpose is required")
	}

	appID := id.Nil
	if req.AppID != "" {
		parsed, err := id.ParseAppID(req.AppID)
		if err != nil {
			return nil, forge.BadRequest("invalid app_id")
		}
		appID = parsed
	}

	now := time.Now()
	c := &Consent{
		ID:        id.NewConsentID(),
		UserID:    userID,
		AppID:     appID,
		Purpose:   req.Purpose,
		Granted:   true,
		Version:   req.Version,
		IPAddress: clientIPFromRequest(ctx),
		GrantedAt: now,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := p.store.GrantConsent(ctx.Context(), c); err != nil {
		return nil, forge.InternalError(fmt.Errorf("failed to record consent"))
	}

	p.audit(ctx.Context(), "consent.grant", "consent", c.ID.String(), userID.String(), appID.String(), map[string]string{
		"purpose": req.Purpose,
		"version": req.Version,
	})

	p.emitHook(ctx.Context(), "consent.grant", "consent", c.ID.String(), userID.String(), appID.String())

	p.relayEvent(ctx.Context(), "consent.granted", appID.String(), map[string]string{
		"user_id": userID.String(),
		"purpose": req.Purpose,
		"version": req.Version,
	})

	return nil, ctx.JSON(http.StatusOK, c)
}

func (p *Plugin) handleRevoke(ctx forge.Context, req *RevokeConsentRequest) (*StatusResponse, error) {
	userID, ok := middleware.UserIDFrom(ctx.Context())
	if !ok {
		return nil, forge.Unauthorized("authentication required")
	}

	if req.Purpose == "" {
		return nil, forge.BadRequest("purpose is required")
	}

	appID := id.Nil
	if req.AppID != "" {
		parsed, err := id.ParseAppID(req.AppID)
		if err != nil {
			return nil, forge.BadRequest("invalid app_id")
		}
		appID = parsed
	}

	if err := p.store.RevokeConsent(ctx.Context(), userID, appID, req.Purpose); err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, forge.NotFound("consent record not found")
		}
		return nil, forge.InternalError(fmt.Errorf("failed to revoke consent"))
	}

	p.audit(ctx.Context(), "consent.revoke", "consent", "", userID.String(), appID.String(), map[string]string{
		"purpose": req.Purpose,
	})

	p.emitHook(ctx.Context(), "consent.revoke", "consent", "", userID.String(), appID.String())

	p.relayEvent(ctx.Context(), "consent.revoked", appID.String(), map[string]string{
		"user_id": userID.String(),
		"purpose": req.Purpose,
	})

	resp := &StatusResponse{Status: "revoked"}
	return nil, ctx.JSON(http.StatusOK, resp)
}

func (p *Plugin) handleList(ctx forge.Context, req *ListConsentsRequest) (*ListResponse, error) {
	userID, ok := middleware.UserIDFrom(ctx.Context())
	if !ok {
		return nil, forge.Unauthorized("authentication required")
	}

	q := &Query{
		UserID:  userID,
		Purpose: req.Purpose,
		Cursor:  req.Cursor,
		Limit:   req.Limit,
	}

	if q.Limit <= 0 {
		q.Limit = 50
	}
	if q.Limit > 200 {
		q.Limit = 200
	}

	consents, cursor, err := p.store.ListConsents(ctx.Context(), q)
	if err != nil {
		return nil, forge.InternalError(fmt.Errorf("failed to list consents"))
	}

	resp := &ListResponse{
		Consents:   consents,
		NextCursor: cursor,
	}
	return nil, ctx.JSON(http.StatusOK, resp)
}

// ──────────────────────────────────────────────────
// Request / response types
// ──────────────────────────────────────────────────

// GrantConsentRequest binds the body for POST /consent/grant.
type GrantConsentRequest struct {
	AppID   string `json:"app_id,omitempty" description:"Application ID"`
	Purpose string `json:"purpose" description:"Consent purpose (e.g. marketing, analytics, essential)"`
	Version string `json:"version,omitempty" description:"Policy version this consent applies to"`
}

// RevokeConsentRequest binds the body for POST /consent/revoke.
type RevokeConsentRequest struct {
	AppID   string `json:"app_id,omitempty" description:"Application ID"`
	Purpose string `json:"purpose" description:"Consent purpose to revoke"`
}

// ListConsentsRequest binds query params for GET /consent.
type ListConsentsRequest struct {
	Purpose string `query:"purpose" description:"Filter by consent purpose"`
	Cursor  string `query:"cursor" description:"Pagination cursor"`
	Limit   int    `query:"limit" description:"Maximum number of results (default 50, max 200)"`
}

// ListResponse wraps a paginated list of consent records.
type ListResponse struct {
	Consents   []*Consent `json:"consents" description:"List of consent records"`
	NextCursor string     `json:"next_cursor,omitempty" description:"Pagination cursor for next page"`
}

// StatusResponse is a generic status response.
type StatusResponse struct {
	Status string `json:"status" description:"Operation status"`
}

// ──────────────────────────────────────────────────
// Observability helpers
// ──────────────────────────────────────────────────

func (p *Plugin) audit(ctx context.Context, action, resource, resourceID, actorID, tenant string, metadata map[string]string) {
	if p.chronicle == nil {
		return
	}
	_ = p.chronicle.Record(ctx, &bridge.AuditEvent{ //nolint:errcheck // best-effort audit
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		ActorID:    actorID,
		Tenant:     tenant,
		Outcome:    bridge.OutcomeSuccess,
		Severity:   bridge.SeverityInfo,
		Category:   "consent",
		Metadata:   metadata,
	})
}

func (p *Plugin) relayEvent(ctx context.Context, eventType, tenantID string, data map[string]string) {
	if p.relay == nil {
		return
	}
	_ = p.relay.Send(ctx, &bridge.WebhookEvent{ //nolint:errcheck // best-effort webhook
		Type:     eventType,
		TenantID: tenantID,
		Data:     data,
	})
}

func (p *Plugin) emitHook(ctx context.Context, action, resource, resourceID, actorID, tenant string) {
	if p.hooks == nil {
		return
	}
	p.hooks.Emit(ctx, &hook.Event{
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		ActorID:    actorID,
		Tenant:     tenant,
	})
}

func clientIPFromRequest(ctx forge.Context) string {
	r := ctx.Request()
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	return r.RemoteAddr
}
