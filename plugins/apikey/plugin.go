package apikey

import (
	"context"
	"errors"
	"fmt"
	log "github.com/xraph/go-utils/log"
	"net/http"
	"strings"
	"time"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/apikey"
	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/hook"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugin"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/store"
	"github.com/xraph/authsome/strategy"
	"github.com/xraph/authsome/user"
)

// Compile-time interface checks.
var (
	_ plugin.Plugin           = (*Plugin)(nil)
	_ plugin.RouteProvider    = (*Plugin)(nil)
	_ plugin.OnInit           = (*Plugin)(nil)
	_ plugin.StrategyProvider = (*Plugin)(nil)
)

// Config configures the API key plugin.
type Config struct {
	// PathPrefix is the HTTP path prefix for API key routes.
	// Defaults to "/v1/auth/keys".
	PathPrefix string

	// MaxKeysPerUser limits the number of active API keys per user.
	// 0 means unlimited.
	MaxKeysPerUser int

	// DefaultExpiry sets the default TTL for newly created keys.
	// Zero means keys never expire.
	DefaultExpiry time.Duration
}

// UserResolver resolves a user by ID string.
type UserResolver func(userID string) (*user.User, error)

// Plugin is the API key authentication plugin.
type Plugin struct {
	config Config
	store  apikey.Store

	resolveUser UserResolver
	chronicle   bridge.Chronicle
	relay       bridge.EventRelay
	hooks       *hook.Bus
	logger      log.Logger
}

// New creates a new API key plugin with the given configuration.
func New(s apikey.Store, cfg ...Config) *Plugin {
	var c Config
	if len(cfg) > 0 {
		c = cfg[0]
	}
	if c.PathPrefix == "" {
		c.PathPrefix = "/v1/auth/keys"
	}
	return &Plugin{config: c, store: s}
}

// Name returns the plugin name.
func (p *Plugin) Name() string { return "apikey" }

// OnInit captures bridge references from the engine.
func (p *Plugin) OnInit(_ context.Context, engine any) error {
	type chronicleGetter interface {
		Chronicle() bridge.Chronicle
	}
	if cg, ok := engine.(chronicleGetter); ok {
		p.chronicle = cg.Chronicle()
	}

	type relayGetter interface {
		Relay() bridge.EventRelay
	}
	if rg, ok := engine.(relayGetter); ok {
		p.relay = rg.Relay()
	}

	type hooksGetter interface {
		Hooks() *hook.Bus
	}
	if hg, ok := engine.(hooksGetter); ok {
		p.hooks = hg.Hooks()
	}

	type loggerGetter interface {
		Logger() log.Logger
	}
	if lg, ok := engine.(loggerGetter); ok {
		p.logger = lg.Logger()
	}

	type userResolver interface {
		ResolveUser(userID string) (*user.User, error)
	}
	if ur, ok := engine.(userResolver); ok {
		p.resolveUser = ur.ResolveUser
	}

	return nil
}

// Strategy returns the API key authentication strategy.
func (p *Plugin) Strategy() strategy.Strategy {
	return &apikeyStrategy{store: p.store, resolveUser: p.resolveUser}
}

// StrategyPriority returns the evaluation priority for the API key strategy.
// API key auth is evaluated after session-based auth (priority 100).
func (p *Plugin) StrategyPriority() int { return 100 }

// ──────────────────────────────────────────────────
// Route Registration
// ──────────────────────────────────────────────────

// RegisterRoutes registers API key management routes on a forge.Router.
func (p *Plugin) RegisterRoutes(r any) error {
	router, ok := r.(forge.Router)
	if !ok {
		return fmt.Errorf("apikey: expected forge.Router, got %T", r)
	}

	prefix := p.config.PathPrefix
	g := router.Group(prefix, forge.WithGroupTags("API Keys"))

	if err := g.POST("", p.handleCreate,
		forge.WithSummary("Create API key"),
		forge.WithOperationID("createAPIKey"),
		forge.WithCreatedResponse(CreateKeyResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.GET("", p.handleList,
		forge.WithSummary("List API keys"),
		forge.WithOperationID("listAPIKeys"),
		forge.WithResponseSchema(http.StatusOK, "API keys list", ListKeysResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	return g.DELETE("/:keyId", p.handleRevoke,
		forge.WithSummary("Revoke API key"),
		forge.WithOperationID("revokeAPIKey"),
		forge.WithErrorResponses(),
	)
}

// ──────────────────────────────────────────────────
// Request/Response Types
// ──────────────────────────────────────────────────

// CreateKeyRequest is the request body for creating an API key.
type CreateKeyRequest struct {
	AppID  string   `json:"app_id"`
	UserID string   `json:"user_id"`
	Name   string   `json:"name"`
	Scopes []string `json:"scopes,omitempty"`
}

// CreateKeyResponse is returned when a key is created.
type CreateKeyResponse struct {
	ID        string   `json:"id"`
	Key       string   `json:"key"`
	KeyPrefix string   `json:"key_prefix"`
	Name      string   `json:"name"`
	Scopes    []string `json:"scopes,omitempty"`
	ExpiresAt *string  `json:"expires_at,omitempty"`
	CreatedAt string   `json:"created_at"`
}

// ListKeysRequest contains query parameters for listing API keys.
type ListKeysRequest struct {
	AppID  string `query:"app_id"`
	UserID string `query:"user_id,omitempty"`
}

// ListKeysResponse is the response for listing API keys.
type ListKeysResponse struct {
	Keys  []KeyListItem `json:"keys"`
	Total int           `json:"total"`
}

// KeyListItem represents an API key in list responses (no raw key).
type KeyListItem struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	KeyPrefix  string   `json:"key_prefix"`
	Scopes     []string `json:"scopes,omitempty"`
	ExpiresAt  *string  `json:"expires_at,omitempty"`
	LastUsedAt *string  `json:"last_used_at,omitempty"`
	Revoked    bool     `json:"revoked"`
	CreatedAt  string   `json:"created_at"`
}

// RevokeKeyRequest contains the path parameter for revoking an API key.
type RevokeKeyRequest struct {
	KeyID string `path:"keyId"`
}

// ──────────────────────────────────────────────────
// Forge Handlers
// ──────────────────────────────────────────────────

func (p *Plugin) handleCreate(ctx forge.Context, req *CreateKeyRequest) (*CreateKeyResponse, error) {
	if req.AppID == "" || req.UserID == "" || req.Name == "" {
		return nil, forge.BadRequest("app_id, user_id, and name are required")
	}

	appID, err := id.ParseAppID(req.AppID)
	if err != nil {
		return nil, forge.BadRequest("invalid app_id")
	}
	userID, err := id.ParseUserID(req.UserID)
	if err != nil {
		return nil, forge.BadRequest("invalid user_id")
	}

	// Check max keys limit
	if p.config.MaxKeysPerUser > 0 {
		existing, err := p.store.ListAPIKeysByUser(ctx.Context(), appID, userID)
		if err != nil {
			return nil, forge.InternalError(fmt.Errorf("failed to check existing keys: %w", err))
		}
		activeCount := 0
		for _, k := range existing {
			if !k.Revoked {
				activeCount++
			}
		}
		if activeCount >= p.config.MaxKeysPerUser {
			return nil, forge.NewHTTPError(http.StatusConflict, fmt.Sprintf("maximum of %d active keys reached", p.config.MaxKeysPerUser))
		}
	}

	// Generate key
	raw, hash, prefix, err := apikey.GenerateKey()
	if err != nil {
		return nil, forge.InternalError(fmt.Errorf("failed to generate key: %w", err))
	}

	now := time.Now()
	key := &apikey.APIKey{
		ID:        id.NewAPIKeyID(),
		AppID:     appID,
		UserID:    userID,
		Name:      req.Name,
		KeyHash:   hash,
		KeyPrefix: prefix,
		Scopes:    req.Scopes,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if p.config.DefaultExpiry > 0 {
		exp := now.Add(p.config.DefaultExpiry)
		key.ExpiresAt = &exp
	}

	if err := p.store.CreateAPIKey(ctx.Context(), key); err != nil {
		return nil, forge.InternalError(fmt.Errorf("failed to create key: %w", err))
	}

	p.audit(ctx.Context(), hook.ActionAPIKeyCreate, hook.ResourceAPIKey, key.ID.String(), userID.String(), "", bridge.OutcomeSuccess)
	p.relayEvent(ctx.Context(), "apikey.created", "", map[string]string{"user_id": userID.String()})
	p.emitHook(ctx.Context(), hook.ActionAPIKeyCreate, hook.ResourceAPIKey, key.ID.String(), userID.String(), "")

	resp := &CreateKeyResponse{
		ID:        key.ID.String(),
		Key:       raw,
		KeyPrefix: key.KeyPrefix,
		Name:      key.Name,
		Scopes:    key.Scopes,
		CreatedAt: key.CreatedAt.Format(time.RFC3339),
	}
	if key.ExpiresAt != nil {
		s := key.ExpiresAt.Format(time.RFC3339)
		resp.ExpiresAt = &s
	}

	return nil, ctx.JSON(http.StatusCreated, resp)
}

func (p *Plugin) handleList(ctx forge.Context, req *ListKeysRequest) (*ListKeysResponse, error) {
	if req.AppID == "" {
		return nil, forge.BadRequest("app_id query parameter is required")
	}

	appID, err := id.ParseAppID(req.AppID)
	if err != nil {
		return nil, forge.BadRequest("invalid app_id")
	}

	var keys []*apikey.APIKey
	if req.UserID != "" {
		userID, err := id.ParseUserID(req.UserID)
		if err != nil {
			return nil, forge.BadRequest("invalid user_id")
		}
		keys, err = p.store.ListAPIKeysByUser(ctx.Context(), appID, userID)
		if err != nil {
			return nil, forge.InternalError(fmt.Errorf("failed to list keys: %w", err))
		}
	} else {
		keys, err = p.store.ListAPIKeysByApp(ctx.Context(), appID)
		if err != nil {
			return nil, forge.InternalError(fmt.Errorf("failed to list keys: %w", err))
		}
	}

	items := make([]KeyListItem, 0, len(keys))
	for _, k := range keys {
		item := KeyListItem{
			ID:        k.ID.String(),
			Name:      k.Name,
			KeyPrefix: k.KeyPrefix,
			Scopes:    k.Scopes,
			Revoked:   k.Revoked,
			CreatedAt: k.CreatedAt.Format(time.RFC3339),
		}
		if k.ExpiresAt != nil {
			s := k.ExpiresAt.Format(time.RFC3339)
			item.ExpiresAt = &s
		}
		if k.LastUsedAt != nil {
			s := k.LastUsedAt.Format(time.RFC3339)
			item.LastUsedAt = &s
		}
		items = append(items, item)
	}

	resp := &ListKeysResponse{Keys: items, Total: len(items)}
	return resp, nil
}

func (p *Plugin) handleRevoke(ctx forge.Context, req *RevokeKeyRequest) (*struct{}, error) {
	if req.KeyID == "" {
		return nil, forge.BadRequest("key id is required")
	}

	keyID, err := id.ParseAPIKeyID(req.KeyID)
	if err != nil {
		return nil, forge.BadRequest("invalid key id")
	}

	key, err := p.store.GetAPIKey(ctx.Context(), keyID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) || errors.Is(err, apikey.ErrNotFound) {
			return nil, forge.NotFound("key not found")
		}
		return nil, forge.InternalError(fmt.Errorf("failed to get key: %w", err))
	}

	key.Revoked = true
	key.UpdatedAt = time.Now()
	if err := p.store.UpdateAPIKey(ctx.Context(), key); err != nil {
		return nil, forge.InternalError(fmt.Errorf("failed to revoke key: %w", err))
	}

	p.audit(ctx.Context(), hook.ActionAPIKeyRevoke, hook.ResourceAPIKey, key.ID.String(), key.UserID.String(), "", bridge.OutcomeSuccess)
	p.relayEvent(ctx.Context(), "apikey.revoked", "", map[string]string{"user_id": key.UserID.String()})
	p.emitHook(ctx.Context(), hook.ActionAPIKeyRevoke, hook.ResourceAPIKey, key.ID.String(), key.UserID.String(), "")

	return nil, ctx.NoContent(http.StatusNoContent)
}

// ──────────────────────────────────────────────────
// Strategy
// ──────────────────────────────────────────────────

// apikeyStrategy authenticates requests using API keys.
type apikeyStrategy struct {
	store       apikey.Store
	resolveUser UserResolver
}

var _ strategy.Strategy = (*apikeyStrategy)(nil)

// Name returns the strategy name.
func (s *apikeyStrategy) Name() string { return "apikey" }

// Authenticate checks for an API key in the Authorization or X-API-Key header.
// API keys are identified by the "ask_" marker prefix.
func (s *apikeyStrategy) Authenticate(ctx context.Context, r *http.Request) (*strategy.Result, error) {
	rawKey := extractAPIKey(r)
	if rawKey == "" {
		return nil, strategy.ErrStrategyNotApplicable{}
	}

	// Extract the app_id from query or X-App-ID header
	appIDStr := r.Header.Get("X-App-ID")
	if appIDStr == "" {
		appIDStr = r.URL.Query().Get("app_id")
	}
	if appIDStr == "" {
		return nil, fmt.Errorf("apikey: X-App-ID header or app_id query parameter required")
	}

	appID, err := id.ParseAppID(appIDStr)
	if err != nil {
		return nil, fmt.Errorf("apikey: invalid app_id: %w", err)
	}

	// Extract prefix from the raw key for lookup
	prefix := extractPrefix(rawKey)
	if prefix == "" {
		return nil, fmt.Errorf("apikey: invalid key format")
	}

	key, err := s.store.GetAPIKeyByPrefix(ctx, appID, prefix)
	if err != nil {
		return nil, fmt.Errorf("apikey: key not found")
	}

	// Verify the raw key against stored hash
	if !apikey.VerifyKey(rawKey, key.KeyHash) {
		return nil, fmt.Errorf("apikey: invalid key")
	}

	if !key.IsValid() {
		return nil, fmt.Errorf("apikey: key is revoked or expired")
	}

	// Update last used timestamp (best-effort, don't fail auth)
	now := time.Now()
	key.LastUsedAt = &now
	_ = s.store.UpdateAPIKey(ctx, key)

	// Resolve the user associated with this API key.
	if s.resolveUser == nil {
		return nil, fmt.Errorf("apikey: user resolver not configured")
	}
	u, err := s.resolveUser(key.UserID.String())
	if err != nil {
		return nil, fmt.Errorf("apikey: resolve user: %w", err)
	}

	// Create a synthetic (non-persisted) session for context propagation.
	syntheticSession := &session.Session{
		ID:        id.NewSessionID(),
		AppID:     key.AppID,
		UserID:    key.UserID,
		EnvID:     key.EnvID,
		CreatedAt: now,
		ExpiresAt: now.Add(24 * time.Hour),
	}

	return &strategy.Result{User: u, Session: syntheticSession}, nil
}

// extractAPIKey extracts the API key from the request.
func extractAPIKey(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if auth != "" {
		parts := strings.SplitN(auth, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			token := strings.TrimSpace(parts[1])
			if strings.HasPrefix(token, "ask_") {
				return token
			}
		}
	}

	if key := r.Header.Get("X-API-Key"); key != "" {
		return key
	}

	return ""
}

// extractPrefix returns the visible prefix of a raw API key.
func extractPrefix(raw string) string {
	if !strings.HasPrefix(raw, "ask_") {
		return ""
	}
	if len(raw) < 12 {
		return ""
	}
	return raw[:12]
}

// ──────────────────────────────────────────────────
// Bridge helpers
// ──────────────────────────────────────────────────

// audit records an audit event via Chronicle (nil-safe).
func (p *Plugin) audit(ctx context.Context, action, resource, resourceID, actorID, tenant, outcome string) {
	if p.chronicle == nil {
		return
	}
	_ = p.chronicle.Record(ctx, &bridge.AuditEvent{
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		ActorID:    actorID,
		Tenant:     tenant,
		Outcome:    outcome,
		Severity:   bridge.SeverityInfo,
	})
}

// relayEvent sends a webhook event to EventRelay (nil-safe).
func (p *Plugin) relayEvent(ctx context.Context, eventType, tenantID string, data map[string]string) {
	if p.relay == nil {
		return
	}
	_ = p.relay.Send(ctx, &bridge.WebhookEvent{
		Type:     eventType,
		TenantID: tenantID,
		Data:     data,
	})
}

// emitHook fires a global hook event (nil-safe).
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
