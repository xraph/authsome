package authsome

import (
	"context"
	"errors"
	"strings"
	"time"

	"go.opentelemetry.io/otel/trace"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/internal/telemetry"
)

// Span names for sign-in, sessions and passwords. Every one of them is
// SpanKindInternal.
const (
	spanIssueSession   = "issue session"
	spanRefreshSession = "refresh session"
	spanHashPassword   = "hash password"
	spanVerifyPassword = "verify password"
)

// ──────────────────────────────────────────────────
// Sign-in results
// ──────────────────────────────────────────────────

// hookRefusedError marks an error a plugin's before hook returned to stop
// an operation, so the sign-in counter can say "denied" rather than
// "error". It changes nothing a caller sees: the message is the wrapped
// error's, and errors.Is and errors.As reach through it.
type hookRefusedError struct{ err error }

func (r *hookRefusedError) Error() string { return r.err.Error() }
func (r *hookRefusedError) Unwrap() error { return r.err }

// signInResult sorts the error a sign-in path returned into the bounded
// result label. A wrong password and an unknown identifier both arrive as
// account.ErrInvalidCredentials, so both are "invalid".
func signInResult(err error) string {
	var (
		mfa     *MFARequiredError
		dpopReq *DPoPRequiredError
		locked  *account.LockedError
		refused *hookRefusedError
	)
	switch {
	case err == nil:
		return telemetry.SignInOK
	case errors.As(err, &mfa):
		return telemetry.SignInMFARequired
	case errors.As(err, &locked), errors.Is(err, account.ErrAccountLocked):
		return telemetry.SignInLocked
	case errors.Is(err, account.ErrUserBanned):
		return telemetry.SignInBanned
	case errors.Is(err, account.ErrInvalidCredentials):
		return telemetry.SignInInvalid
	case errors.Is(err, account.ErrEmailNotVerified):
		return telemetry.SignInUnverified
	case errors.Is(err, account.ErrPasswordExpired):
		return telemetry.SignInPasswordExpired
	case errors.Is(err, account.ErrRateLimited):
		return telemetry.SignInRateLimited
	case errors.As(err, &refused), errors.As(err, &dpopReq):
		return telemetry.SignInDenied
	default:
		return telemetry.SignInError
	}
}

// sessionFormat names the shape of a minted access token without reading
// anything in it: a JWT is three dot-separated parts, an opaque token none.
func sessionFormat(token string) string {
	if strings.Count(token, ".") == 2 {
		return "jwt"
	}
	return "opaque"
}

// ──────────────────────────────────────────────────
// Refresh results
// ──────────────────────────────────────────────────

// refreshRefusalError tags a refusal whose sentinel alone (always
// account.ErrInvalidCredentials, so the caller cannot learn why) would hide
// the reason from the refresh counter. finishRefresh unwraps it, so callers
// receive exactly the sentinel they always did.
type refreshRefusalError struct {
	result string
	err    error
}

func (r *refreshRefusalError) Error() string { return r.err.Error() }
func (r *refreshRefusalError) Unwrap() error { return r.err }

// startRefresh starts the refresh session span for one entry point. The
// returned func ends it: it records the result on the span and the counter
// and hands back the error the caller should see.
func (e *Engine) startRefresh(ctx context.Context, via string) (spanCtx context.Context, finish func(error) error) {
	ctx, span := telemetry.Start(ctx, spanRefreshSession,
		trace.WithAttributes(telemetry.AttrRefreshVia.String(via)))
	return ctx, func(err error) error {
		result := telemetry.RefreshOK
		var rr *refreshRefusalError
		switch {
		case err == nil:
		case errors.As(err, &rr):
			result, err = rr.result, rr.err
		case errors.Is(err, account.ErrSessionExpired):
			result = telemetry.RefreshExpired
		case errors.Is(err, account.ErrUserBanned):
			result = telemetry.RefreshBanned
		case errors.Is(err, account.ErrInvalidCredentials):
			result = telemetry.RefreshInvalid
		default:
			result = telemetry.RefreshError
			telemetry.FailErr(span, err)
		}
		span.SetAttributes(telemetry.AttrResult.String(result))
		span.End()
		e.tel.Refresh(result)
		return err
	}
}

// ──────────────────────────────────────────────────
// Password hashing
// ──────────────────────────────────────────────────

// passwordAlgorithm names the algorithm a policy hashes with.
func passwordAlgorithm(policy account.PasswordPolicy) string {
	if policy.Algorithm == "argon2id" {
		return telemetry.AlgorithmArgon2id
	}
	return telemetry.AlgorithmBcrypt
}

// hashAlgorithm names the algorithm a stored hash was made with.
func hashAlgorithm(hash string) string {
	if account.IsArgon2Hash(hash) {
		return telemetry.AlgorithmArgon2id
	}
	return telemetry.AlgorithmBcrypt
}

// hashPassword hashes password under policy inside a hash password span.
func (e *Engine) hashPassword(ctx context.Context, password string, policy account.PasswordPolicy) (string, error) {
	algorithm := passwordAlgorithm(policy)
	_, span := telemetry.Start(ctx, spanHashPassword,
		trace.WithAttributes(telemetry.AttrPasswordAlgorithm.String(algorithm)))
	defer span.End()
	start := time.Now()
	hash, err := account.HashPasswordWithPolicy(password, policy)
	e.tel.PasswordHash(telemetry.PasswordHash, algorithm, time.Since(start))
	if err != nil {
		telemetry.FailErr(span, err)
	}
	return hash, err
}

// verifyPassword checks password against hash inside a verify password
// span. A mismatch is an answer, not a failure, so nothing on the span says
// how the check came out; the span also carries no algorithm, because an
// unknown identifier's stand-in check (consumeDummyVerify) has no stored
// hash to name one from and must look exactly like this.
func (e *Engine) verifyPassword(ctx context.Context, hash, password string) error {
	_, span := telemetry.Start(ctx, spanVerifyPassword)
	defer span.End()
	start := time.Now()
	err := account.CheckPassword(hash, password)
	e.tel.PasswordHash(telemetry.PasswordVerify, hashAlgorithm(hash), time.Since(start))
	return err
}

// consumeDummyVerify is ConsumeDummyHash for sign-in's unknown-identifier
// path. It spends the same budget under the same span a real check gets,
// so neither the response time nor the trace says whether the identifier
// exists.
func (e *Engine) consumeDummyVerify(ctx context.Context, password string) {
	_, span := telemetry.Start(ctx, spanVerifyPassword)
	defer span.End()
	start := time.Now()
	policy := e.ConsumeDummyHash(password)
	e.tel.PasswordHash(telemetry.PasswordVerify, passwordAlgorithm(policy), time.Since(start))
}
