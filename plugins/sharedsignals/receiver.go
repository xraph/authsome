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

// dummyHash is compared against on a stream miss so an unknown push path and
// a bad token cost roughly the same work.
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

	// Gates 5 to 11 live in setjwt: typ, algorithm allow-list, key
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

	// Gate 12: the dedupe row commits before anything acts, which makes the
	// row the ledger. A conflict is a replay, and a replay is a success.
	if err := p.processSET(ctx, stream, token); err != nil {
		if errors.Is(err, ErrDuplicateJTI) {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		p.logger.Error("sharedsignals: process security event token",
			logString("stream_id", stream.ID.String()),
			logString("error", err.Error()),
		)
		setError(w, http.StatusBadRequest, "invalid_request",
			"the security event token could not be processed")
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

// processSET records each event, resolves its subject and applies the matrix.
// A single unusable event never fails the whole SET: it is recorded with its
// outcome and the rest carry on.
func (p *Plugin) processSET(ctx context.Context, stream *InboundStream,
	token *setjwt.Token) error {
	for eventType, payload := range token.Events {
		record := &ReceivedEvent{
			ID:        id.NewSSFEventID(),
			StreamID:  stream.ID,
			JTI:       token.JTI,
			EventType: eventType,
			Outcome:   OutcomePending,
		}
		if err := p.store.InsertReceivedEvent(ctx, record); err != nil {
			return err
		}

		outcome, action, failure := p.processOneEvent(ctx, stream, eventType, payload)
		record.Outcome = outcome
		record.ActionTaken = action
		if failure != nil {
			record.Error = failure.Error()
		}
		if err := p.store.UpdateReceivedEvent(ctx, record); err != nil {
			return err
		}
	}
	return nil
}

func (p *Plugin) processOneEvent(ctx context.Context, stream *InboundStream,
	eventType string, payload json.RawMessage) (outcome, action string, failure error) {
	if !caep.IsKnownEventType(eventType) {
		return OutcomeIgnored, "", nil
	}
	if !allowsEventType(stream, eventType) {
		return OutcomeIgnored, "", nil
	}

	ev, err := caep.ParseEvent(eventType, payload)
	if err != nil {
		return OutcomeRejected, "", err
	}

	res, err := p.resolveSubject(ctx, stream, ev.Subject)
	if err != nil {
		return OutcomeRejected, "", err
	}
	if res.Outcome != OutcomeApplied {
		// Unresolved and rejected both stop here without an error, so the
		// transmitter learns nothing about who does or does not have an
		// account and stops retrying.
		return res.Outcome, "", nil
	}

	allowed, err := p.checkCircuitBreaker(ctx, stream)
	if err != nil {
		return OutcomeRejected, "", err
	}
	if !allowed {
		return OutcomeRejected, "", errors.New("stream paused by the circuit breaker")
	}

	action, err = p.applyEvent(ctx, stream, ev, res)
	if err != nil {
		return OutcomeRejected, "", err
	}
	return OutcomeApplied, action, nil
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
