package sharedsignals

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/xraph/forge"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/sharedsignals/caep"
	"github.com/xraph/authsome/plugins/sharedsignals/jwksclient"
	"github.com/xraph/authsome/plugins/sharedsignals/setjwt"
	"github.com/xraph/authsome/settings"
)

// HashSecret returns the hex SHA-256 of a push secret. Only the hash is
// stored, so a database copy does not yield a working push endpoint.
func HashSecret(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// NewPushSecret mints a URL-safe random secret and its hash. The plaintext is
// shown to the operator once at stream creation and never persisted.
func NewPushSecret() (raw, hash string, err error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", err
	}
	raw = base64.RawURLEncoding.EncodeToString(buf)
	return raw, HashSecret(raw), nil
}

// registerReceiverRoutes mounts the push endpoint.
func (p *Plugin) registerReceiverRoutes(router forge.Router) error {
	g := router.Group("/v1/ssf", forge.WithGroupTags("Shared Signals"))

	opts := []forge.RouteOption{
		forge.WithSummary("Receive Security Event Tokens"),
		forge.WithOperationID("receiveSecurityEventTokens"),
	}
	// Nil when the host is not the concrete engine or rate limiting is off,
	// which is why this is defence in depth and not the control.
	opts = append(opts, authsome.PluginRateLimit(p.engine,
		func(c authsome.RateLimitConfig) int { return c.SSFPushLimit })...)

	// A raw handler: the body is a compact JWS rather than JSON, success is a
	// bodiless 202, and failure is the RFC 8935 error object.
	return g.POST("/streams/:push_path/events", p.handlePush, opts...)
}

func (p *Plugin) handlePush(ctx forge.Context) error {
	p.servePush(ctx.Response(), ctx.Request(), ctx.Param("push_path"))
	return nil
}

// servePushForTest exposes the pipeline to tests without a forge context.
func (p *Plugin) servePushForTest(w http.ResponseWriter, r *http.Request, pushPath string) {
	p.servePush(w, r, pushPath)
}

// setError writes the RFC 8935 error object. The description never echoes
// anything the caller sent.
func setError(w http.ResponseWriter, status int, code, description string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{ //nolint:errcheck // response write
		"err": code, "description": description,
	})
}

// dummyHash is compared against on a stream miss so the lookup itself takes
// roughly the same time whether or not the push path exists. The 404 and 401
// status codes are already distinguishable to the caller -- this compare is
// not trying to hide that -- it only stops an attacker from timing the
// lookup to learn whether a guessed push path is real before ever reaching
// gate 4's token compare.
const dummyHash = "0000000000000000000000000000000000000000000000000000000000000000"

func (p *Plugin) servePush(w http.ResponseWriter, r *http.Request, pushPath string) {
	ctx := r.Context()

	// Gate 1: bound the body before reading any of it.
	body, err := io.ReadAll(io.LimitReader(r.Body, p.config.MaxBodyBytes+1))
	if err != nil {
		setError(w, http.StatusBadRequest, "invalid_request", "could not read the request body")
		return
	}
	if int64(len(body)) > p.config.MaxBodyBytes {
		setError(w, http.StatusBadRequest, "invalid_request", "the request body is too large")
		return
	}

	// Gate 2: the secret URL segment selects the stream. Everything we trust
	// comes from this row, never from the token.
	stream, err := p.store.GetInboundStreamByPushPathHash(ctx, HashSecret(pushPath))
	if err != nil {
		// Burn a comparison so a missing stream and a bad token look alike.
		_ = subtle.ConstantTimeCompare([]byte(dummyHash), []byte(dummyHash))
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// Gate 3: a paused or disabled stream acts on nothing.
	if stream.Status != StatusEnabled {
		setError(w, http.StatusForbidden, "access_denied", "the stream is not enabled")
		return
	}

	// Gate 4: the bearer token the transmitter was given at registration.
	presented := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if subtle.ConstantTimeCompare(
		[]byte(HashSecret(presented)), []byte(stream.PushTokenHash)) != 1 {
		setError(w, http.StatusUnauthorized, "authentication_failed",
			"the transmitter could not be authenticated")
		return
	}

	// Gate 4b: the operator's kill switch. sharedsignals.enabled is declared
	// WithEnforceable(), so an operator who turns it off believes they have
	// stopped the receiver -- and until now they still had a live remote
	// session-kill endpoint. It is checked after authentication so that
	// turning the receiver off does not also hand an unauthenticated caller
	// a way to probe which push paths exist.
	//
	// 503, not 403: this is a temporary operator decision, and every RFC
	// 8935 error code tells a well-behaved transmitter that the token is
	// wrong and retrying is pointless. A 5xx with Retry-After leaves the
	// transmitter free to deliver the same SET again once the receiver is
	// switched back on, so nothing is lost while it is off.
	if !p.receiverEnabled(ctx, stream) {
		w.Header().Set("Retry-After", "300")
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	// Gates 5 to 11 live in setjwt: algorithm allow-list, typ, key
	// resolution, signature, issuer, audience, jti and iat.
	token, err := setjwt.Validate(ctx, body, setjwt.Options{
		Issuer:    stream.Issuer,
		Audience:  p.audienceFor(stream),
		Keys:      &streamKeys{client: p.jwks, uri: stream.JWKSURI},
		Now:       time.Now,
		MaxAge:    p.config.MaxSETAge,
		ClockSkew: p.config.ClockSkew,
		MaxEvents: 10,
	})
	if err != nil {
		// One case in here is not a verdict on the token at all: the key set
		// could not be loaded, so the signature was never actually checked.
		// Answering 400 invalid_key there tells a well-behaved transmitter
		// to STOP retrying, which permanently drops whatever that SET was
		// carrying because our own IdP fetch had a bad five minutes. Same
		// reasoning as the store-failure branch below: a 5xx invites the
		// retry that would succeed.
		if errors.Is(err, setjwt.ErrKeyUnavailable) {
			p.logger.Error("sharedsignals: could not load the stream's key set",
				logString("stream_id", stream.ID.String()),
				logString("jwks_uri", stream.JWKSURI),
				logString("error", err.Error()),
			)
			w.Header().Set("Retry-After", "60")
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		setError(w, http.StatusBadRequest, setjwt.ErrCode(err),
			"the security event token was rejected")
		return
	}

	// Gate 12: each event's dedupe row commits before it is acted on, which
	// makes the row the ledger. A conflict on every event in the SET means
	// the whole delivery is a replay, and a replay is a success.
	if err := p.processSET(ctx, stream, token); err != nil {
		if errors.Is(err, ErrDuplicateJTI) {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		p.logger.Error("sharedsignals: process security event token",
			logString("stream_id", stream.ID.String()),
			logString("error", err.Error()),
		)
		// Nothing here is a reason the TOKEN was rejected -- it is our own
		// store failing to insert, update or undo a received-event row, or
		// an infrastructure error surfaced from processing one event (see
		// processOneEvent). None of RFC 8935's error codes describe "our
		// database had a blip," and a 400 tells a well-behaved transmitter
		// to stop retrying, which is exactly wrong when a retry would
		// actually succeed. Answer plain 500 instead, with no RFC 8935 body.
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func (p *Plugin) audienceFor(s *InboundStream) string {
	if s.Audience != "" {
		return s.Audience
	}
	return p.config.Audience
}

// streamKeys adapts the shared JWKS client to the per-stream resolver setjwt
// expects, so a kid can only ever select a key from this stream's key set.
type streamKeys struct {
	client *jwksclient.Client
	uri    string
}

// Key translates the one jwksclient failure that is ours rather than the
// token's. Everything else stays a plain error and setjwt turns it into
// invalid_key, which is the right permanent answer for a token naming a key
// the issuer does not publish.
func (s *streamKeys) Key(ctx context.Context, kid string) (crypto.PublicKey, error) {
	key, err := s.client.Key(ctx, s.uri, kid)
	if err != nil && errors.Is(err, jwksclient.ErrFetchFailed) {
		return nil, fmt.Errorf("%w: %w", setjwt.ErrKeyUnavailable, err)
	}
	return key, err
}

// receiverEnabled reads the sharedsignals.enabled kill switch for this
// stream's app. A settings manager that is absent or unreachable leaves the
// receiver on: this switch exists so an operator can stop the receiver
// deliberately, and a settings outage is not a deliberate decision. The
// stream's own status field is the control that fails closed.
func (p *Plugin) receiverEnabled(ctx context.Context, s *InboundStream) bool {
	if p.settingsMgr == nil {
		return true
	}
	enabled, err := settings.Get(ctx, p.settingsMgr, SettingEnabled,
		settings.ResolveOpts{AppID: s.AppID.String()})
	if err != nil {
		// settings.Get already falls back to the registered default (true)
		// on an error, so this only logs why.
		p.logger.Warn("sharedsignals: could not resolve the enabled setting, staying on",
			logString("stream_id", s.ID.String()),
			logString("error", err.Error()),
		)
		return true
	}
	return enabled
}

// processSET records each event, resolves its subject and applies the
// matrix. Each event is deduped on its own -- see ErrDuplicateJTI's doc
// comment on why the key includes event_type -- so one event colliding with
// a prior delivery does not stop the SET's other events, which carry their
// own event types and so their own dedupe identity, from being processed.
// Only when every event in the SET has already been recorded is the whole
// delivery reported as the replay it actually is.
//
// A single unusable event never fails the whole SET for a POLICY reason: an
// unresolved subject, a rejected subject, an event type the stream did not
// subscribe to and so on are all recorded with their outcome and the rest
// carry on. An INFRASTRUCTURE failure is different: see processOneEvent's
// infra return value.
func (p *Plugin) processSET(ctx context.Context, stream *InboundStream,
	token *setjwt.Token) error {
	anyNew := false
	for eventType, payload := range token.Events {
		record := &ReceivedEvent{
			ID:        id.NewSSFEventID(),
			StreamID:  stream.ID,
			JTI:       token.JTI,
			EventType: eventType,
			Outcome:   OutcomePending,
		}
		if err := p.store.InsertReceivedEvent(ctx, record); err != nil {
			if errors.Is(err, ErrDuplicateJTI) {
				continue
			}
			return err
		}
		anyNew = true

		result := p.processOneEvent(ctx, stream, eventType, payload)
		outcome, action, failure, infra := result.Outcome, result.Action, result.Failure, result.Infra
		if infra {
			// The dedupe row this event just wrote must not survive an
			// infrastructure failure: leaving it committed would make the
			// transmitter's retry read back as a replay of a delivery that
			// was never actually handled, permanently dropping whatever
			// this event was. Best-effort undo and let the caller answer
			// with an error so the transmitter retries and this time the
			// insert starts fresh.
			if derr := p.store.DeleteReceivedEvent(ctx, record.ID); derr != nil {
				p.logger.Error(
					"sharedsignals: could not undo dedupe row after an infrastructure failure",
					logString("event_id", record.ID.String()),
					logString("stream_id", stream.ID.String()),
					logString("error", derr.Error()),
				)
			}
			return failure
		}

		// The two fields that make this row an audit trail rather than a
		// bare dedupe key. They were modelled, migrated and round-tripped by
		// every backend, and never once written: processSET used to throw
		// the Resolution away. That matters more here than it looks, because
		// the receiver deliberately keeps offending values OUT of its error
		// responses on the reasoning that they belong in the audit record.
		record.SubjectJSON = result.SubjectJSON
		record.ResolvedUserID = result.UserID
		record.Outcome = outcome
		record.ActionTaken = action
		if failure != nil {
			record.Error = failure.Error()
		}
		if err := p.store.UpdateReceivedEvent(ctx, record); err != nil {
			return err
		}

		// applyEvent audits every outcome that produced an action, with the
		// metadata only it has (partial-revocation counts, observe mode).
		// Everything else -- ignored, unresolved, rejected, signal-only --
		// used to reach Chronicle never, against a spec that says every
		// accepted and rejected event goes there.
		if action == "" {
			p.auditOutcome(ctx, stream, eventType, result)
		}
	}
	if !anyNew {
		return ErrDuplicateJTI
	}
	return nil
}

// eventResult is what one event from a SET came to. It exists so the
// audit row can carry the subject and the resolved user, which a bare
// (outcome, action) pair had no way to express.
type eventResult struct {
	Outcome string
	Action  string
	// SubjectJSON is the subject exactly as the transmitter sent it.
	SubjectJSON string
	// UserID is set only when the subject resolved to one of our users.
	UserID id.UserID
	// Failure is the reason an outcome is not "applied", or the error that
	// aborted an action. Recorded on the row; never returned to the caller.
	Failure error
	// Infra is true only when Failure came from a store call breaking for
	// its own reasons -- not from any decision about the event's content --
	// and tells the caller this event was never actually resolved one way
	// or the other, so its dedupe row must not be left standing. Every
	// other result represents a genuine policy outcome (ignored,
	// unresolved, rejected, applied) and is meant to be recorded and
	// answered with success.
	Infra bool
}

func ignoredResult() eventResult { return eventResult{Outcome: OutcomeIgnored} }

func infraResult(err error) eventResult { return eventResult{Failure: err, Infra: true} }

// processOneEvent handles one event from a SET.
func (p *Plugin) processOneEvent(ctx context.Context, stream *InboundStream,
	eventType string, payload json.RawMessage) eventResult {
	if !caep.IsKnownEventType(eventType) {
		return ignoredResult()
	}
	if !allowsEventType(stream, eventType) {
		return ignoredResult()
	}

	ev, err := caep.ParseEvent(eventType, payload)
	if err != nil {
		return eventResult{Outcome: OutcomeRejected, Failure: err}
	}
	subjectJSON := string(ev.RawSubject)

	// A stream-level event describes the stream, not a principal on it, so
	// it is dispatched ahead of subject resolution. Routing it through the
	// subject path instead is what made the verification handshake
	// impossible to complete: an event with no subject can never resolve,
	// and an unresolved event stops before applyEvent is ever reached.
	if caep.IsStreamLevel(eventType) {
		if err := p.completeVerification(ctx, stream, ev); err != nil {
			// The only failure in there is UpdateInboundStream breaking,
			// which is our store, not the transmitter's problem.
			return infraResult(err)
		}
		return eventResult{Outcome: OutcomeApplied, SubjectJSON: subjectJSON}
	}

	res, err := p.resolveSubject(ctx, stream, ev.Subject)
	if err != nil {
		// Every error resolveSubject can return here traces back to a store
		// call (resolveViaLink's GetSubjectLink) that failed for a reason
		// other than "not found" -- an infrastructure problem, not a policy
		// decision about the subject. A miss is unresolved, not an error;
		// see resolveViaLink.
		return infraResult(err)
	}
	if res.Outcome != OutcomeApplied {
		// Unresolved and rejected both stop here without an error, so the
		// transmitter learns nothing about who does or does not have an
		// account and stops retrying.
		return eventResult{Outcome: res.Outcome, SubjectJSON: subjectJSON}
	}

	allowed, err := p.checkCircuitBreaker(ctx, stream)
	if err != nil {
		// Same reasoning as resolveSubject above: the breaker's count
		// failing is the store breaking, not a verdict that this event
		// should be rejected.
		return infraResult(err)
	}
	if !allowed {
		return eventResult{
			Outcome: OutcomeRejected, SubjectJSON: subjectJSON, UserID: res.UserID,
			Failure: errors.New("stream paused by the circuit breaker"),
		}
	}

	action, err := p.applyEvent(ctx, stream, ev, res)
	if err != nil {
		// The action comes back even on the error path, because a partial
		// revocation really did end some sessions and the row has to say so.
		// It is also what tells the caller applyEvent already wrote its own,
		// richer audit record for this event.
		return eventResult{
			Outcome: OutcomeRejected, Action: action, SubjectJSON: subjectJSON,
			UserID: res.UserID, Failure: err,
		}
	}
	return eventResult{
		Outcome: OutcomeApplied, Action: action,
		SubjectJSON: subjectJSON, UserID: res.UserID,
	}
}

func allowsEventType(s *InboundStream, eventType string) bool {
	// An empty list means the stream accepts every type it understands.
	if len(s.AllowedEventTypes) == 0 {
		return true
	}
	for _, t := range s.AllowedEventTypes {
		if t == eventType {
			return true
		}
	}
	return false
}
