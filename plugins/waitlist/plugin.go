package waitlist

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/forge"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/hook"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/plugin"
	"github.com/xraph/authsome/settings"

	"github.com/xraph/grove/migrate"
)

// Compile-time interface checks.
var (
	_ plugin.Plugin                       = (*Plugin)(nil)
	_ plugin.OnInit                       = (*Plugin)(nil)
	_ plugin.RouteProvider                = (*Plugin)(nil)
	_ plugin.MigrationProvider            = (*Plugin)(nil)
	_ plugin.NotificationMappingContributor = (*Plugin)(nil)
	_ plugin.BeforeSignUp                 = (*Plugin)(nil)
)

// Config configures the waitlist plugin.
type Config struct {
	// Enabled controls whether the waitlist gate is active.
	// When true, sign-ups require a prior approved waitlist entry.
	Enabled bool
}

// Plugin is the waitlist management plugin.
type Plugin struct {
	config      Config
	store       Store
	hooks       *hook.Bus
	relay       bridge.EventRelay
	chronicle   bridge.Chronicle
	herald      bridge.Herald
	logger      log.Logger
	permChecker middleware.PermissionChecker
	basePath    string
	defaultAppID string
	settingsMgr *settings.Manager
}

// New creates a new waitlist plugin. An in-memory store is used by
// default. Use SetWaitlistStore to inject a persistent store.
func New(cfg ...Config) *Plugin {
	var c Config
	if len(cfg) > 0 {
		c = cfg[0]
	}
	return &Plugin{
		config: c,
		store:  NewMemoryStore(),
	}
}

// Name returns the plugin name.
func (p *Plugin) Name() string { return "waitlist" }

// SetWaitlistStore allows direct waitlist store injection.
func (p *Plugin) SetWaitlistStore(s Store) { p.store = s }

// MigrationGroups returns the waitlist migration groups for the given driver.
func (p *Plugin) MigrationGroups(driverName string) []*migrate.Group {
	switch driverName {
	case "pg", "postgres":
		return []*migrate.Group{PostgresMigrations}
	case "sqlite", "sqlite3":
		return []*migrate.Group{SqliteMigrations}
	case "mongo", "mongodb":
		return []*migrate.Group{MongoMigrations}
	default:
		return nil
	}
}

// OnInit captures engine capabilities.
func (p *Plugin) OnInit(_ context.Context, engine any) error {
	type hooksGetter interface {
		Hooks() *hook.Bus
	}
	if hg, ok := engine.(hooksGetter); ok {
		p.hooks = hg.Hooks()
	}

	type relayGetter interface {
		Relay() bridge.EventRelay
	}
	if rg, ok := engine.(relayGetter); ok {
		p.relay = rg.Relay()
	}

	type chronicleGetter interface {
		Chronicle() bridge.Chronicle
	}
	if cg, ok := engine.(chronicleGetter); ok {
		p.chronicle = cg.Chronicle()
	}

	type heraldGetter interface {
		Herald() bridge.Herald
	}
	if hg, ok := engine.(heraldGetter); ok {
		p.herald = hg.Herald()
	}

	type loggerGetter interface {
		Logger() log.Logger
	}
	if lg, ok := engine.(loggerGetter); ok {
		p.logger = lg.Logger()
	}

	if pc, ok := engine.(middleware.PermissionChecker); ok {
		p.permChecker = pc
	}

	type settingsGetter interface {
		Settings() *settings.Manager
	}
	if sg, ok := engine.(settingsGetter); ok {
		p.settingsMgr = sg.Settings()
	}

	type configGetter interface {
		Config() authsome.Config
	}
	if cg, ok := engine.(configGetter); ok {
		cfg := cg.Config()
		p.defaultAppID = cfg.AppID
		if cfg.BasePath != "" {
			p.basePath = cfg.BasePath
		}
	}
	if p.basePath == "" {
		p.basePath = "/v1"
	}

	return nil
}

// NotificationMappings contributes waitlist notification templates.
func (p *Plugin) NotificationMappings() map[string]*plugin.NotificationMapping {
	return map[string]*plugin.NotificationMapping{
		hook.ActionWaitlistJoin: {
			Template: "waitlist_join",
			Channels: []string{"email"},
			Enabled:  true,
		},
		hook.ActionWaitlistApprove: {
			Template: "waitlist_approve",
			Channels: []string{"email"},
			Enabled:  true,
		},
		hook.ActionWaitlistReject: {
			Template: "waitlist_reject",
			Channels: []string{"email"},
			Enabled:  true,
		},
	}
}

// OnBeforeSignUp implements plugin.BeforeSignUp.
// When the waitlist is enabled, it blocks sign-ups unless the email has
// an approved entry on the waitlist.
func (p *Plugin) OnBeforeSignUp(ctx context.Context, req *account.SignUpRequest) error {
	if !p.config.Enabled {
		return nil
	}

	if req.Email == "" {
		return fmt.Errorf("waitlist: signup requires waitlist approval — join the waitlist first")
	}

	appID := req.AppID
	if appID.IsNil() && p.defaultAppID != "" {
		parsed, err := id.ParseAppID(p.defaultAppID)
		if err == nil {
			appID = parsed
		}
	}

	entry, err := p.store.GetEntryByEmail(ctx, appID, strings.ToLower(req.Email))
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return fmt.Errorf("waitlist: signup requires waitlist approval — join the waitlist first")
		}
		return fmt.Errorf("waitlist: failed to check waitlist status: %w", err)
	}

	if entry.Status != StatusApproved {
		return fmt.Errorf("waitlist: signup requires waitlist approval — join the waitlist first")
	}

	return nil
}

// RegisterRoutes registers waitlist HTTP routes.
func (p *Plugin) RegisterRoutes(router any) error {
	r, ok := router.(forge.Router)
	if !ok {
		return fmt.Errorf("waitlist: expected forge.Router, got %T", router)
	}

	// ──────────────────────────────────────────────────
	// Public routes (no auth required)
	// ──────────────────────────────────────────────────

	pub := r.Group(p.basePath+"/waitlist", forge.WithGroupTags("Waitlist"))

	if err := pub.POST("/join", p.handleJoin,
		forge.WithSummary("Join waitlist"),
		forge.WithDescription("Submit an email to join the waitlist. Idempotent — returns existing entry if already on the list."),
		forge.WithOperationID("joinWaitlist"),
		forge.WithRequestSchema(JoinRequest{}),
		forge.WithResponseSchema(http.StatusOK, "Waitlist entry", WaitlistEntry{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := pub.GET("/status", p.handleStatus,
		forge.WithSummary("Check waitlist status"),
		forge.WithDescription("Returns the waitlist status for a given email."),
		forge.WithOperationID("waitlistStatus"),
		forge.WithResponseSchema(http.StatusOK, "Waitlist status", StatusResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	// ──────────────────────────────────────────────────
	// Admin routes (auth + permission required)
	// ──────────────────────────────────────────────────

	adminMiddlewares := []forge.Middleware{middleware.RequireAuth()}
	if p.permChecker != nil {
		adminMiddlewares = append(adminMiddlewares, middleware.RequirePermission(p.permChecker, "manage", "waitlist"))
	}

	admin := r.Group(p.basePath+"/admin/waitlist",
		forge.WithGroupTags("Waitlist Admin"),
		forge.WithGroupMiddleware(adminMiddlewares...),
	)

	if err := admin.GET("", p.handleList,
		forge.WithSummary("List waitlist entries"),
		forge.WithDescription("Returns a paginated list of waitlist entries with optional filters."),
		forge.WithOperationID("listWaitlistEntries"),
		forge.WithResponseSchema(http.StatusOK, "Waitlist list", WaitlistList{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := admin.GET("/stats", p.handleStats,
		forge.WithSummary("Waitlist statistics"),
		forge.WithDescription("Returns counts of pending, approved, and rejected waitlist entries."),
		forge.WithOperationID("waitlistStats"),
		forge.WithResponseSchema(http.StatusOK, "Waitlist stats", StatsResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := admin.POST("/:entryId/approve", p.handleApprove,
		forge.WithSummary("Approve waitlist entry"),
		forge.WithDescription("Approves a pending waitlist entry, allowing the user to sign up."),
		forge.WithOperationID("approveWaitlistEntry"),
		forge.WithResponseSchema(http.StatusOK, "Approved entry", WaitlistEntry{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := admin.POST("/:entryId/reject", p.handleReject,
		forge.WithSummary("Reject waitlist entry"),
		forge.WithDescription("Rejects a waitlist entry."),
		forge.WithOperationID("rejectWaitlistEntry"),
		forge.WithResponseSchema(http.StatusOK, "Rejected entry", WaitlistEntry{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	return admin.DELETE("/:entryId", p.handleDelete,
		forge.WithSummary("Delete waitlist entry"),
		forge.WithDescription("Permanently removes a waitlist entry."),
		forge.WithOperationID("deleteWaitlistEntry"),
		forge.WithErrorResponses(),
	)
}

// ──────────────────────────────────────────────────
// Request / response types
// ──────────────────────────────────────────────────

// JoinRequest binds the body for POST /waitlist/join.
type JoinRequest struct {
	Email string `json:"email" description:"Email address to add to the waitlist"`
	Name  string `json:"name,omitempty" description:"Display name (optional)"`
	AppID string `json:"app_id,omitempty" description:"Application ID (optional, resolved from context if absent)"`
}

// StatusRequest binds query params for GET /waitlist/status.
type StatusRequest struct {
	Email string `query:"email" description:"Email address to check"`
	AppID string `query:"app_id" description:"Application ID (optional)"`
}

// StatusResponse is returned when checking waitlist status.
type StatusResponse struct {
	Email  string         `json:"email" description:"Email address"`
	Status WaitlistStatus `json:"status" description:"Waitlist status (pending, approved, rejected)"`
}

// ListRequest binds query params for GET /admin/waitlist.
type ListRequest struct {
	AppID  string `query:"app_id" description:"Filter by application ID"`
	Email  string `query:"email" description:"Filter by email address"`
	Status string `query:"status" description:"Filter by status (pending, approved, rejected)"`
	Cursor string `query:"cursor" description:"Pagination cursor"`
	Limit  int    `query:"limit" description:"Maximum number of results (default 50, max 200)"`
}

// StatsRequest binds query params for GET /admin/waitlist/stats.
type StatsRequest struct {
	AppID string `query:"app_id" description:"Application ID"`
}

// StatsResponse holds waitlist entry counts by status.
type StatsResponse struct {
	Pending  int `json:"pending" description:"Number of pending entries"`
	Approved int `json:"approved" description:"Number of approved entries"`
	Rejected int `json:"rejected" description:"Number of rejected entries"`
	Total    int `json:"total" description:"Total number of entries"`
}

// ApproveRequest binds the path + body for POST /admin/waitlist/:entryId/approve.
type ApproveRequest struct {
	EntryID string `path:"entryId" description:"Waitlist entry ID"`
	Note    string `json:"note,omitempty" description:"Optional admin note"`
}

// RejectRequest binds the path + body for POST /admin/waitlist/:entryId/reject.
type RejectRequest struct {
	EntryID string `path:"entryId" description:"Waitlist entry ID"`
	Note    string `json:"note,omitempty" description:"Optional admin note"`
}

// DeleteRequest binds the path param for DELETE /admin/waitlist/:entryId.
type DeleteRequest struct {
	EntryID string `path:"entryId" description:"Waitlist entry ID"`
}

// ──────────────────────────────────────────────────
// Handlers
// ──────────────────────────────────────────────────

func (p *Plugin) handleJoin(ctx forge.Context, req *JoinRequest) (*WaitlistEntry, error) {
	if req.Email == "" {
		return nil, forge.BadRequest("email is required")
	}

	appID, err := p.resolveAppID(ctx, req.AppID)
	if err != nil {
		return nil, forge.BadRequest("unable to determine app_id")
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))

	// Idempotent: return existing entry if already on the list.
	existing, err := p.store.GetEntryByEmail(ctx.Context(), appID, email)
	if err == nil {
		return nil, ctx.JSON(http.StatusOK, existing)
	}
	if !errors.Is(err, ErrNotFound) {
		return nil, forge.InternalError(fmt.Errorf("failed to check waitlist"))
	}

	now := time.Now()
	entry := &WaitlistEntry{
		ID:        id.NewWaitlistID(),
		AppID:     appID,
		Email:     email,
		Name:      strings.TrimSpace(req.Name),
		Status:    StatusPending,
		IPAddress: clientIPFromRequest(ctx),
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := p.store.CreateEntry(ctx.Context(), entry); err != nil {
		if errors.Is(err, ErrDuplicateEmail) {
			// Race condition: entry was created between check and insert.
			existing, lookupErr := p.store.GetEntryByEmail(ctx.Context(), appID, email)
			if lookupErr == nil {
				return nil, ctx.JSON(http.StatusOK, existing)
			}
		}
		return nil, forge.InternalError(fmt.Errorf("failed to create waitlist entry"))
	}

	p.emitHook(ctx.Context(), hook.ActionWaitlistJoin, hook.ResourceWaitlist, entry.ID.String(), "", appID.String())

	p.relayEvent(ctx.Context(), "waitlist.join", appID.String(), map[string]string{
		"email": entry.Email,
		"name":  entry.Name,
	})

	p.audit(ctx.Context(), "waitlist.join", "waitlist", entry.ID.String(), "", appID.String(), map[string]string{
		"email": entry.Email,
		"name":  entry.Name,
	})

	return nil, ctx.JSON(http.StatusOK, entry)
}

func (p *Plugin) handleStatus(ctx forge.Context, req *StatusRequest) (*StatusResponse, error) {
	if req.Email == "" {
		return nil, forge.BadRequest("email query parameter is required")
	}

	appID, err := p.resolveAppID(ctx, req.AppID)
	if err != nil {
		return nil, forge.BadRequest("unable to determine app_id")
	}

	entry, err := p.store.GetEntryByEmail(ctx.Context(), appID, strings.ToLower(req.Email))
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, forge.NotFound("email not found on waitlist")
		}
		return nil, forge.InternalError(fmt.Errorf("failed to check waitlist status"))
	}

	resp := &StatusResponse{
		Email:  entry.Email,
		Status: entry.Status,
	}
	return nil, ctx.JSON(http.StatusOK, resp)
}

func (p *Plugin) handleList(ctx forge.Context, req *ListRequest) (*WaitlistList, error) {
	q := &WaitlistQuery{
		Email:  req.Email,
		Status: WaitlistStatus(req.Status),
		Cursor: req.Cursor,
		Limit:  req.Limit,
	}

	if req.AppID != "" {
		parsed, err := id.ParseAppID(req.AppID)
		if err != nil {
			return nil, forge.BadRequest("invalid app_id")
		}
		q.AppID = parsed
	}

	if q.Limit <= 0 {
		q.Limit = 50
	}
	if q.Limit > 200 {
		q.Limit = 200
	}

	list, err := p.store.ListEntries(ctx.Context(), q)
	if err != nil {
		return nil, forge.InternalError(fmt.Errorf("failed to list waitlist entries"))
	}

	return nil, ctx.JSON(http.StatusOK, list)
}

func (p *Plugin) handleStats(ctx forge.Context, req *StatsRequest) (*StatsResponse, error) {
	appID, err := p.resolveAppID(ctx, req.AppID)
	if err != nil {
		return nil, forge.BadRequest("unable to determine app_id")
	}

	pending, approved, rejected, err := p.store.CountByStatus(ctx.Context(), appID)
	if err != nil {
		return nil, forge.InternalError(fmt.Errorf("failed to get waitlist stats"))
	}

	resp := &StatsResponse{
		Pending:  pending,
		Approved: approved,
		Rejected: rejected,
		Total:    pending + approved + rejected,
	}
	return nil, ctx.JSON(http.StatusOK, resp)
}

func (p *Plugin) handleApprove(ctx forge.Context, req *ApproveRequest) (*WaitlistEntry, error) {
	entryID, err := id.ParseWaitlistID(req.EntryID)
	if err != nil {
		return nil, forge.BadRequest("invalid entry ID")
	}

	if err := p.store.UpdateEntryStatus(ctx.Context(), entryID, StatusApproved, req.Note); err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, forge.NotFound("waitlist entry not found")
		}
		return nil, forge.InternalError(fmt.Errorf("failed to approve entry"))
	}

	entry, err := p.store.GetEntry(ctx.Context(), entryID)
	if err != nil {
		return nil, forge.InternalError(fmt.Errorf("failed to retrieve updated entry"))
	}

	actorID := ""
	if uid, ok := middleware.UserIDFrom(ctx.Context()); ok {
		actorID = uid.String()
	}

	p.emitHook(ctx.Context(), hook.ActionWaitlistApprove, hook.ResourceWaitlist, entry.ID.String(), actorID, entry.AppID.String())

	p.relayEvent(ctx.Context(), "waitlist.approved", entry.AppID.String(), map[string]string{
		"email":  entry.Email,
		"name":   entry.Name,
		"status": string(entry.Status),
	})

	p.audit(ctx.Context(), "waitlist.approve", "waitlist", entry.ID.String(), actorID, entry.AppID.String(), map[string]string{
		"email":  entry.Email,
		"name":   entry.Name,
		"status": string(entry.Status),
	})

	return nil, ctx.JSON(http.StatusOK, entry)
}

func (p *Plugin) handleReject(ctx forge.Context, req *RejectRequest) (*WaitlistEntry, error) {
	entryID, err := id.ParseWaitlistID(req.EntryID)
	if err != nil {
		return nil, forge.BadRequest("invalid entry ID")
	}

	if err := p.store.UpdateEntryStatus(ctx.Context(), entryID, StatusRejected, req.Note); err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, forge.NotFound("waitlist entry not found")
		}
		return nil, forge.InternalError(fmt.Errorf("failed to reject entry"))
	}

	entry, err := p.store.GetEntry(ctx.Context(), entryID)
	if err != nil {
		return nil, forge.InternalError(fmt.Errorf("failed to retrieve updated entry"))
	}

	actorID := ""
	if uid, ok := middleware.UserIDFrom(ctx.Context()); ok {
		actorID = uid.String()
	}

	p.emitHook(ctx.Context(), hook.ActionWaitlistReject, hook.ResourceWaitlist, entry.ID.String(), actorID, entry.AppID.String())

	p.relayEvent(ctx.Context(), "waitlist.rejected", entry.AppID.String(), map[string]string{
		"email":  entry.Email,
		"name":   entry.Name,
		"status": string(entry.Status),
	})

	p.audit(ctx.Context(), "waitlist.reject", "waitlist", entry.ID.String(), actorID, entry.AppID.String(), map[string]string{
		"email":  entry.Email,
		"name":   entry.Name,
		"status": string(entry.Status),
	})

	return nil, ctx.JSON(http.StatusOK, entry)
}

func (p *Plugin) handleDelete(ctx forge.Context, req *DeleteRequest) (*struct{}, error) {
	entryID, err := id.ParseWaitlistID(req.EntryID)
	if err != nil {
		return nil, forge.BadRequest("invalid entry ID")
	}

	if err := p.store.DeleteEntry(ctx.Context(), entryID); err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, forge.NotFound("waitlist entry not found")
		}
		return nil, forge.InternalError(fmt.Errorf("failed to delete entry"))
	}

	actorID := ""
	if uid, ok := middleware.UserIDFrom(ctx.Context()); ok {
		actorID = uid.String()
	}

	p.audit(ctx.Context(), "waitlist.delete", "waitlist", entryID.String(), actorID, "", nil)

	return nil, ctx.JSON(http.StatusOK, map[string]string{"status": "deleted"})
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
		Category:   "waitlist",
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

// ──────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────

// resolveAppID returns the app ID from the request or context, falling back
// to the plugin-level default.
func (p *Plugin) resolveAppID(ctx forge.Context, reqAppID string) (id.AppID, error) {
	if reqAppID != "" {
		return id.ParseAppID(reqAppID)
	}
	if scopeAppID := forge.AppIDFrom(ctx.Context()); scopeAppID != "" {
		return id.ParseAppID(scopeAppID)
	}
	if p.defaultAppID != "" {
		return id.ParseAppID(p.defaultAppID)
	}
	return id.AppID{}, fmt.Errorf("no app_id available")
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
