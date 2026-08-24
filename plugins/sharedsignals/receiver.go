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

func (s *streamKeys) Key(ctx context.Context, kid string) (crypto.PublicKey, error) {
	return s.client.Key(ctx, s.uri, kid)
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

		outcome, action, failure, infra := p.processOneEvent(ctx, stream, eventType, payload)
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

		record.Outcome = outcome
		record.ActionTaken = action
		if failure != nil {
			record.Error = failure.Error()
		}
		if err := p.store.UpdateReceivedEvent(ctx, record); err != nil {
			return err
		}
	}
	if !anyNew {
		return ErrDuplicateJTI
	}
	return nil
}

// processOneEvent handles one event from a SET. infra is true only when
// failure came from a store call breaking for its own reasons -- not from
// any decision about the event's content -- and tells the caller this event
// was never actually resolved one way or the other, so its dedupe row must
// not be left standing. Every other return represents a genuine policy
// outcome (ignored, unresolved, rejected, applied) and is meant to be
// recorded and answered with success, exactly as before.
func (p *Plugin) processOneEvent(ctx context.Context, stream *InboundStream,
	eventType string, payload json.RawMessage) (outcome, action string, failure error, infra bool) {
	if !caep.IsKnownEventType(eventType) {
		return OutcomeIgnored, "", nil, false
	}
	if !allowsEventType(stream, eventType) {
		return OutcomeIgnored, "", nil, false
	}

	ev, err := caep.ParseEvent(eventType, payload)
	if err != nil {
		return OutcomeRejected, "", err, false
	}

	res, err := p.resolveSubject(ctx, stream, ev.Subject)
	if err != nil {
		// Every error resolveSubject can return here traces back to a store
		// call (resolveViaLink's GetSubjectLink) that failed for a reason
		// other than "not found" -- an infrastructure problem, not a policy
		// decision about the subject. A miss is unresolved, not an error;
		// see resolveViaLink.
		return "", "", err, true
	}
	if res.Outcome != OutcomeApplied {
		// Unresolved and rejected both stop here without an error, so the
		// transmitter learns nothing about who does or does not have an
		// account and stops retrying.
		return res.Outcome, "", nil, false
	}

	allowed, err := p.checkCircuitBreaker(ctx, stream)
	if err != nil {
		// Same reasoning as resolveSubject above: CountActionsSince failing
		// is the store breaking, not a verdict that this event should be
		// rejected.
		return "", "", err, true
	}
	if !allowed {
		return OutcomeRejected, "", errors.New("stream paused by the circuit breaker"), false
	}

	action, err = p.applyEvent(ctx, stream, ev, res)
	if err != nil {
		return OutcomeRejected, "", err, false
	}
	return OutcomeApplied, action, nil, false
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
