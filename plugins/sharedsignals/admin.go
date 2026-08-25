package sharedsignals

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/plugin"
	"github.com/xraph/authsome/plugins/sharedsignals/caep"
	"github.com/xraph/authsome/plugins/sharedsignals/jwksclient"
)

// permStreamResource is the RBAC resource name for the stream admin API,
// paired with the "manage" action already used across the repo's admin
// surfaces (see plugins/oauth2provider, plugins/sso, plugins/social:
// "manage"/"oauth2_client", "manage"/"sso_connection",
// "manage"/"social_provider" -- each names the specific entity the group
// manages rather than the whole plugin, which is the pattern this follows).
// A stream is a session-revocation capability, so it gets its own
// permission rather than folding into some broader "sharedsignals" grant
// that might get handed out for unrelated reasons.
const permStreamResource = "ssf_stream"

// CreateStreamRequest registers an identity provider we will accept events
// from.
type CreateStreamRequest struct {
	Name                  string            `json:"name"`
	Issuer                string            `json:"issuer"`
	JWKSURI               string            `json:"jwks_uri"`
	Audience              string            `json:"audience,omitempty"`
	AllowedEventTypes     []string          `json:"allowed_event_types,omitempty"`
	AllowedSubjectFormats []string          `json:"allowed_subject_formats,omitempty"`
	VerifiedDomains       []string          `json:"verified_domains,omitempty"`
	ActionOverrides       map[string]string `json:"action_overrides,omitempty"`
	EnforcementMode       string            `json:"enforcement_mode,omitempty"`
	MaxActionsPerHour     int               `json:"max_actions_per_hour,omitempty"`
}

// StreamView is the read model. It deliberately carries no secret and no
// hash of one.
type StreamView struct {
	ID                    string            `json:"id"`
	Name                  string            `json:"name"`
	Issuer                string            `json:"issuer"`
	Audience              string            `json:"audience"`
	JWKSURI               string            `json:"jwks_uri"`
	AllowedEventTypes     []string          `json:"allowed_event_types"`
	AllowedSubjectFormats []string          `json:"allowed_subject_formats"`
	VerifiedDomains       []string          `json:"verified_domains"`
	ActionOverrides       map[string]string `json:"action_overrides"`
	EnforcementMode       string            `json:"enforcement_mode"`
	Status                string            `json:"status"`
	MaxActionsPerHour     int               `json:"max_actions_per_hour"`
	LastVerifiedAt        *time.Time        `json:"last_verified_at,omitempty"`
	CreatedAt             time.Time         `json:"created_at"`
	UpdatedAt             time.Time         `json:"updated_at"`
}

// CreateStreamResponse carries the two secrets. This is the only time either
// is readable; the store keeps hashes.
type CreateStreamResponse struct {
	Stream StreamView `json:"stream"`
	// PushURLPath is the secret segment in the push URL.
	PushURLPath string `json:"push_url_path"`
	// PushToken is the bearer token the transmitter must send.
	PushToken string `json:"push_token"`
	// PushURL is the full URL to hand to the transmitter.
	PushURL string `json:"push_url"`
}

// UpdateStreamRequest changes the mutable parts of a stream. The two push
// secrets are not among them; rotate by creating a new stream.
type UpdateStreamRequest struct {
	Name                  *string           `json:"name,omitempty"`
	Status                *string           `json:"status,omitempty"`
	EnforcementMode       *string           `json:"enforcement_mode,omitempty"`
	MaxActionsPerHour     *int              `json:"max_actions_per_hour,omitempty"`
	JWKSURI               *string           `json:"jwks_uri,omitempty"`
	AllowedEventTypes     []string          `json:"allowed_event_types,omitempty"`
	AllowedSubjectFormats []string          `json:"allowed_subject_formats,omitempty"`
	VerifiedDomains       []string          `json:"verified_domains,omitempty"`
	ActionOverrides       map[string]string `json:"action_overrides,omitempty"`
}

// ListStreamsResponse is the list payload.
type ListStreamsResponse struct {
	Streams []StreamView `json:"streams"`
}

func toStreamView(s *InboundStream) StreamView {
	return StreamView{
		ID: s.ID.String(), Name: s.Name, Issuer: s.Issuer,
		Audience: s.Audience, JWKSURI: s.JWKSURI,
		AllowedEventTypes:     s.AllowedEventTypes,
		AllowedSubjectFormats: s.AllowedSubjectFormats,
		VerifiedDomains:       s.VerifiedDomains,
		ActionOverrides:       s.ActionOverrides,
		EnforcementMode:       s.EnforcementMode,
		Status:                s.Status,
		MaxActionsPerHour:     s.MaxActionsPerHour,
		LastVerifiedAt:        s.LastVerifiedAt,
		CreatedAt:             s.CreatedAt, UpdatedAt: s.UpdatedAt,
	}
}

func containsString(haystack []string, needle string) bool {
	for _, h := range haystack {
		if h == needle {
			return true
		}
	}
	return false
}

// validateStreamState checks the invariants a stream must hold no matter how
// it got there. CreateStream runs it against the row it is about to insert;
// UpdateStream must run it against the fully-merged RESULTING state of a
// patch, not just the fields that particular patch happened to touch --
// otherwise a PATCH that only clears verified_domains (leaving
// allowed_subject_formats untouched, still carrying "email") sails through
// with neither field individually looking wrong, and the stream ends up in a
// state CreateStream would have refused outright: the email/phone_number
// formats active with no verified domain backing them, which lets the
// transmitter name anyone.
func validateStreamState(s *InboundStream) error {
	// The same check the fetcher runs, applied before the URL is ever
	// stored -- and re-applied here so an update can't introduce a URL the
	// create path would have refused either.
	if err := jwksclient.ValidateURI(s.JWKSURI); err != nil {
		return fmt.Errorf("sharedsignals: %w", err)
	}
	// Email and phone name a person without the IdP proving it issued them,
	// so they only make sense inside domains the operator has claimed.
	if (containsString(s.AllowedSubjectFormats, caep.FormatEmail) ||
		containsString(s.AllowedSubjectFormats, caep.FormatPhoneNumber)) &&
		len(s.VerifiedDomains) == 0 {
		return errors.New(
			"sharedsignals: the email and phone_number formats require verified_domains")
	}
	return nil
}

// CreateStream registers a stream and mints its two secrets.
func (p *Plugin) CreateStream(ctx context.Context, appID id.AppID,
	envID id.EnvironmentID, req CreateStreamRequest) (*CreateStreamResponse, error) {
	if req.Issuer == "" {
		return nil, errors.New("sharedsignals: issuer is required")
	}
	if req.JWKSURI == "" {
		return nil, errors.New("sharedsignals: jwks_uri is required")
	}

	formats := req.AllowedSubjectFormats
	if len(formats) == 0 {
		// iss_sub only by default. Every other format widens who the
		// transmitter is allowed to name.
		formats = []string{caep.FormatIssSub}
	}
	if err := validateStreamState(&InboundStream{
		JWKSURI: req.JWKSURI, AllowedSubjectFormats: formats,
		VerifiedDomains: req.VerifiedDomains,
	}); err != nil {
		return nil, err
	}

	mode := req.EnforcementMode
	if mode == "" {
		mode = EnforcementEnforce
	}
	if mode != EnforcementEnforce && mode != EnforcementObserve {
		return nil, fmt.Errorf("sharedsignals: unknown enforcement mode %q", mode)
	}

	audience := req.Audience
	if audience == "" {
		audience = p.config.Audience
	}

	limit := req.MaxActionsPerHour
	if limit <= 0 {
		limit = p.config.MaxActionsPerHour
	}

	pushPath, pushPathHash, err := NewPushSecret()
	if err != nil {
		return nil, err
	}
	pushToken, pushTokenHash, err := NewPushSecret()
	if err != nil {
		return nil, err
	}

	stream := &InboundStream{
		ID: id.NewSSFStreamID(), AppID: appID, EnvID: envID,
		Name: req.Name, Issuer: req.Issuer, Audience: audience,
		JWKSURI:      req.JWKSURI,
		PushPathHash: pushPathHash, PushTokenHash: pushTokenHash,
		AllowedEventTypes:     req.AllowedEventTypes,
		AllowedSubjectFormats: formats,
		VerifiedDomains:       req.VerifiedDomains,
		ActionOverrides:       req.ActionOverrides,
		EnforcementMode:       mode,
		Status:                StatusEnabled,
		MaxActionsPerHour:     limit,
	}
	if err := p.store.CreateInboundStream(ctx, stream); err != nil {
		return nil, err
	}

	return &CreateStreamResponse{
		Stream:      toStreamView(stream),
		PushURLPath: pushPath,
		PushToken:   pushToken,
		PushURL:     "/v1/ssf/streams/" + pushPath + "/events",
	}, nil
}

// UpdateStream changes the mutable fields of a stream.
func (p *Plugin) UpdateStream(ctx context.Context, streamID id.SSFStreamID,
	req UpdateStreamRequest) (*StreamView, error) {
	stream, err := p.store.GetInboundStream(ctx, streamID)
	if err != nil {
		return nil, err
	}
	if req.Name != nil {
		stream.Name = *req.Name
	}
	if req.Status != nil {
		switch *req.Status {
		case StatusEnabled, StatusPaused, StatusDisabled:
			stream.Status = *req.Status
		default:
			return nil, fmt.Errorf("sharedsignals: unknown status %q", *req.Status)
		}
	}
	if req.EnforcementMode != nil {
		switch *req.EnforcementMode {
		case EnforcementEnforce, EnforcementObserve:
			stream.EnforcementMode = *req.EnforcementMode
		default:
			return nil, fmt.Errorf("sharedsignals: unknown enforcement mode %q", *req.EnforcementMode)
		}
	}
	if req.MaxActionsPerHour != nil {
		stream.MaxActionsPerHour = *req.MaxActionsPerHour
	}
	if req.JWKSURI != nil {
		stream.JWKSURI = *req.JWKSURI
	}
	if req.AllowedEventTypes != nil {
		stream.AllowedEventTypes = req.AllowedEventTypes
	}
	if req.AllowedSubjectFormats != nil {
		stream.AllowedSubjectFormats = req.AllowedSubjectFormats
	}
	if req.VerifiedDomains != nil {
		stream.VerifiedDomains = req.VerifiedDomains
	}
	if req.ActionOverrides != nil {
		stream.ActionOverrides = req.ActionOverrides
	}

	// Validate the RESULTING state, not the individual fields this patch
	// happened to touch -- see validateStreamState's doc comment for why
	// that distinction matters.
	if err := validateStreamState(stream); err != nil {
		return nil, err
	}

	if err := p.store.UpdateInboundStream(ctx, stream); err != nil {
		return nil, err
	}
	view := toStreamView(stream)
	return &view, nil
}

// registerAdminRoutes mounts the stream CRUD behind session auth, plus an
// RBAC permission check on the three routes that mutate a stream.
//
// forge.WithGroupAuth("session") on its own writes OpenAPI metadata only --
// it documents the requirement but enforces nothing at request time. The
// actual gate is forge.WithGroupMiddleware(plugin.SessionGuard(p.engine)...),
// which is what plugin/authz.go's own doc comment pairs it with (see also
// plugins/waitlist/plugin.go and plugins/subscription/routes.go for the same
// pairing elsewhere in this codebase). Without it this group was reachable
// by anyone, session or no session.
//
// SessionGuard alone still isn't the right bar for this group: it only
// proves SOME session exists, and a stream is a session-revocation
// capability, so any signed-in end user of the app would otherwise be able
// to mint one for themselves. The create/update/delete routes carry an
// additional permStreamResource permission check for that reason; list and
// get stay at session-only, same as the read/write split in
// plugins/subscription/routes.go.
func (p *Plugin) registerAdminRoutes(router forge.Router) error {
	g := router.Group("/v1/ssf/admin",
		forge.WithGroupTags("Shared Signals"),
		forge.WithGroupAuth("session"),
		forge.WithGroupMiddleware(plugin.SessionGuard(p.engine)...),
	)

	writeGuard := plugin.PermissionRouteOptions(p.engine, "manage", permStreamResource)

	if err := g.POST("/streams", p.handleCreateStream,
		append([]forge.RouteOption{
			forge.WithSummary("Register an inbound Shared Signals stream"),
			forge.WithOperationID("createSSFStream"),
		}, writeGuard...)...,
	); err != nil {
		return err
	}
	if err := g.GET("/streams", p.handleListStreams,
		forge.WithSummary("List inbound Shared Signals streams"),
		forge.WithOperationID("listSSFStreams"),
	); err != nil {
		return err
	}
	if err := g.PATCH("/streams/:id", p.handleUpdateStream,
		append([]forge.RouteOption{
			forge.WithSummary("Update an inbound Shared Signals stream"),
			forge.WithOperationID("updateSSFStream"),
		}, writeGuard...)...,
	); err != nil {
		return err
	}
	return g.DELETE("/streams/:id", p.handleDeleteStream,
		append([]forge.RouteOption{
			forge.WithSummary("Delete an inbound Shared Signals stream"),
			forge.WithOperationID("deleteSSFStream"),
		}, writeGuard...)...,
	)
}

func writeJSON(w http.ResponseWriter, status int, v any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(v)
}

// writeJSONForTest lets the admin tests assert on the encoded shape.
func writeJSONForTest(w http.ResponseWriter, v any) error {
	return writeJSON(w, http.StatusOK, v)
}

func (p *Plugin) handleCreateStream(ctx forge.Context) error {
	var req CreateStreamRequest
	body, err := io.ReadAll(io.LimitReader(ctx.Request().Body, 64*1024))
	if err != nil {
		return forge.BadRequest("could not read the request body")
	}
	if unmarshalErr := json.Unmarshal(body, &req); unmarshalErr != nil {
		return forge.BadRequest("the request body is not valid JSON")
	}

	appID, envID, err := p.requestScope(ctx)
	if err != nil {
		return err
	}

	res, cerr := p.CreateStream(ctx.Context(), appID, envID, req)
	if cerr != nil {
		return forge.BadRequest(cerr.Error())
	}
	return writeJSON(ctx.Response(), http.StatusCreated, res)
}

func (p *Plugin) handleListStreams(ctx forge.Context) error {
	appID, _, err := p.requestScope(ctx)
	if err != nil {
		return err
	}
	streams, lerr := p.store.ListInboundStreams(ctx.Context(), appID)
	if lerr != nil {
		return forge.InternalError(lerr)
	}
	views := make([]StreamView, 0, len(streams))
	for _, s := range streams {
		views = append(views, toStreamView(s))
	}
	return writeJSON(ctx.Response(), http.StatusOK, ListStreamsResponse{Streams: views})
}

// requireStreamInCallerApp confirms the stream belongs to appID. A
// stream is what authorises session revocation, so an ID from one tenant
// must never be actionable by another. It answers a mismatch identically to
// a stream that does not exist -- NotFound, not Forbidden -- because
// Forbidden would itself confirm to the caller that the ID is valid for some
// OTHER tenant, which is exactly the fact an IDOR probe is trying to learn.
func (p *Plugin) requireStreamInCallerApp(ctx context.Context, appID id.AppID,
	streamID id.SSFStreamID) error {
	stream, err := p.store.GetInboundStream(ctx, streamID)
	if err != nil {
		return err
	}
	if stream.AppID.String() != appID.String() {
		return ErrNotFound
	}

	return nil
}

func (p *Plugin) handleUpdateStream(ctx forge.Context) error {
	streamID, perr := id.ParseWithPrefix(ctx.Param("id"), id.PrefixSSFStream)
	if perr != nil {
		return forge.BadRequest("invalid stream id")
	}

	appID, _, err := p.requestScope(ctx)
	if err != nil {
		return err
	}
	if serr := p.requireStreamInCallerApp(ctx.Context(), appID, streamID); serr != nil {
		return forge.NotFound("stream not found")
	}

	var req UpdateStreamRequest
	body, err := io.ReadAll(io.LimitReader(ctx.Request().Body, 64*1024))
	if err != nil {
		return forge.BadRequest("could not read the request body")
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return forge.BadRequest("the request body is not valid JSON")
	}

	view, uerr := p.UpdateStream(ctx.Context(), streamID, req)
	if uerr != nil {
		if errors.Is(uerr, ErrNotFound) {
			return forge.NotFound("stream not found")
		}
		return forge.BadRequest(uerr.Error())
	}
	return writeJSON(ctx.Response(), http.StatusOK, view)
}

func (p *Plugin) handleDeleteStream(ctx forge.Context) error {
	streamID, perr := id.ParseWithPrefix(ctx.Param("id"), id.PrefixSSFStream)
	if perr != nil {
		return forge.BadRequest("invalid stream id")
	}

	appID, _, err := p.requestScope(ctx)
	if err != nil {
		return err
	}
	if serr := p.requireStreamInCallerApp(ctx.Context(), appID, streamID); serr != nil {
		return forge.NotFound("stream not found")
	}

	if err := p.store.DeleteInboundStream(ctx.Context(), streamID); err != nil {
		if errors.Is(err, ErrNotFound) {
			return forge.NotFound("stream not found")
		}
		return forge.InternalError(err)
	}
	ctx.Response().WriteHeader(http.StatusNoContent)
	return nil
}

// requestScope resolves the app and environment the caller is acting in.
// The publishable-key middleware puts both on the context; the configured
// default app is the fallback, matching how plugins/sso resolves it in
// requestAppID.
func (p *Plugin) requestScope(ctx forge.Context) (appID id.AppID, envID id.EnvironmentID, err error) {
	envID, _ = middleware.EnvIDFrom(ctx.Context())

	if ctxAppID, ok := middleware.AppIDFrom(ctx.Context()); ok {
		return ctxAppID, envID, nil
	}
	appID, err = id.ParseAppID(p.engine.DefaultAppID())
	if err != nil {
		return id.Nil, id.Nil, forge.BadRequest("invalid app configuration")
	}
	return appID, envID, nil
}
