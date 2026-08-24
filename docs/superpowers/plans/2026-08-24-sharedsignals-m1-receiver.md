# sharedsignals M1 Receiver Core Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make authsome an OpenID Shared Signals Framework Receiver, so a CAEP `session-revoked` event pushed by Okta, Entra or Google Workspace actually ends that user's authsome sessions.

**Architecture:** Three dependency-free packages (`caep`, `setjwt`, `jwksclient`) carry the spec plumbing and are tested against fixtures alone. The `sharedsignals` plugin wires them to authsome through the standard plugin hooks, persisting streams, subject links, received events and risk signals across memory, postgres, sqlite and mongo. A push endpoint validates every SET through an ordered gate chain before any session dies, and a `RiskContributor` replays stored signals into `riskengine` at the next sign-in.

**Tech Stack:** Go 1.26.0, `github.com/golang-jwt/jwt/v5` (already a direct dependency), `github.com/xraph/grove` + `grove/migrate` for persistence, `github.com/xraph/forge` for routing, `stretchr/testify` for assertions.

**Spec:** `docs/superpowers/specs/2026-08-24-sharedsignals-ssf-caep-design.md`

## Global Constraints

- Go 1.26.0. Module path `github.com/xraph/authsome`.
- No new third-party dependencies. Use `golang-jwt/jwt/v5`. Do NOT add `go-jose`, `lestrrat-go/jwx`, or `MicahParks/keyfunc`.
- All tables are prefixed `authsome_ssf_`. All migration groups are named `authsome-sharedsignals` and declare `migrate.DependsOn("authsome")`.
- Signature algorithms accepted: `RS256`, `RS384`, `RS512`, `ES256`, `ES384`, `ES512`, `EdDSA`. `none` and every HMAC algorithm (`HS*`) are rejected.
- SET `typ` header must be `secevent+jwt`.
- `iat` must fall inside `[now - 24h, now + 5m]`.
- Push success is `202 Accepted` with an empty body. Failure is `400` with `{"err": ..., "description": ...}` using only these codes: `invalid_request`, `invalid_key`, `invalid_issuer`, `invalid_audience`, `authentication_failed`, `access_denied`. The `description` never echoes caller-supplied input.
- An unresolvable subject returns `202`, never an error.
- Secrets (push path segment, push bearer token) are stored only as SHA-256 hashes.
- Every lookup is scoped to the stream's `app_id` and `env_id`. No cross-app resolution.
- Test command for a single package: `go test ./plugins/sharedsignals/... -run TestName -v`. Full suite: `make test`.

## Spec addendum

The spec's "Changes outside the plugin" section lists three core changes. Implementation review found two corrections, applied in this plan:

1. `*authsome.Engine` already has both `RevokeSession` (`service.go:694`) and `Dispatcher` (`engine.go:750`). The two new capability interfaces in `plugin/plugin.go` are therefore declarations only; no method is added to the engine.
2. A fourth core change is needed that the spec did not name: four ID prefixes in `id/id.go`, following the convention every other plugin uses.

## File Structure

**New, dependency-free spec packages:**

| File | Responsibility |
|---|---|
| `plugins/sharedsignals/caep/events.go` | Event type URI constants, `Event` struct, `ParseEvent` |
| `plugins/sharedsignals/caep/subject.go` | RFC 9493 `SubjectID`, complex subjects, `ParseSubjectID` |
| `plugins/sharedsignals/setjwt/setjwt.go` | `Header`, `Validate`, `Token`, the RFC 8935 error sentinels |
| `plugins/sharedsignals/jwksclient/client.go` | Fetch, cache, kid lookup, all the hardening limits |

**Plugin:**

| File | Responsibility |
|---|---|
| `plugins/sharedsignals/plugin.go` | `Plugin`, `Config`, settings, `OnInit`, store selection |
| `plugins/sharedsignals/store.go` | Domain types and the `Store` interface |
| `plugins/sharedsignals/store_memory.go` | In-memory `Store` |
| `plugins/sharedsignals/store_models.go` | Grove models plus `from*`/`to*` mappers |
| `plugins/sharedsignals/store_postgres.go` | Postgres `Store` |
| `plugins/sharedsignals/store_sqlite.go` | SQLite `Store` |
| `plugins/sharedsignals/store_mongo.go` | Mongo `Store` |
| `plugins/sharedsignals/migrations.go` | Three `migrate.Group` values |
| `plugins/sharedsignals/subject.go` | `SubjectID` to authsome user resolution |
| `plugins/sharedsignals/links.go` | `LinkSubject` and the social derivation |
| `plugins/sharedsignals/actions.go` | The action matrix and revocation |
| `plugins/sharedsignals/receiver.go` | The push endpoint and its gate chain |
| `plugins/sharedsignals/admin.go` | Stream CRUD routes |
| `plugins/sharedsignals/risk.go` | `riskengine.RiskContributor` |

**Modified:**

| File | Change |
|---|---|
| `id/id.go` | Four new `Prefix` constants and their constructors |
| `plugin/plugin.go` | `SessionRevoker` and `DispatcherProvider` interfaces |
| `plugins/riskengine/plugin.go` | Populate `Identifier` and `UserID` on `RiskRequest` |
| `plugins/sso/plugin.go` | Call `LinkSubject` after OIDC sign-in |

---

### Task 1: caep package, subject identifiers

**Files:**
- Create: `plugins/sharedsignals/caep/subject.go`
- Test: `plugins/sharedsignals/caep/subject_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces: `caep.SubjectID` struct with fields `Format, Email, PhoneNumber, Issuer, Subject, ID, URI, URL, Identifiers []SubjectID, Members map[string]SubjectID`; `caep.ParseSubjectID(raw json.RawMessage) (SubjectID, error)`; `func (s SubjectID) IsComplex() bool`; `func (s SubjectID) Member(name string) (SubjectID, bool)`.

- [ ] **Step 1: Write the failing test**

Create `plugins/sharedsignals/caep/subject_test.go`:

```go
package caep

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSubjectID_IssSub(t *testing.T) {
	got, err := ParseSubjectID(json.RawMessage(
		`{"format":"iss_sub","iss":"https://org.okta.com","sub":"okta-user-id1"}`))
	require.NoError(t, err)
	assert.Equal(t, "iss_sub", got.Format)
	assert.Equal(t, "https://org.okta.com", got.Issuer)
	assert.Equal(t, "okta-user-id1", got.Subject)
	assert.False(t, got.IsComplex())
}

func TestParseSubjectID_Email(t *testing.T) {
	got, err := ParseSubjectID(json.RawMessage(`{"format":"email","email":"a@b.com"}`))
	require.NoError(t, err)
	assert.Equal(t, "email", got.Format)
	assert.Equal(t, "a@b.com", got.Email)
}

func TestParseSubjectID_Complex(t *testing.T) {
	got, err := ParseSubjectID(json.RawMessage(
		`{"user":{"format":"iss_sub","iss":"https://i","sub":"u1"},` +
			`"session":{"format":"opaque","id":"sess-9"}}`))
	require.NoError(t, err)
	assert.True(t, got.IsComplex())

	user, ok := got.Member("user")
	require.True(t, ok)
	assert.Equal(t, "u1", user.Subject)

	sess, ok := got.Member("session")
	require.True(t, ok)
	assert.Equal(t, "sess-9", sess.ID)

	_, ok = got.Member("device")
	assert.False(t, ok)
}

func TestParseSubjectID_Aliases(t *testing.T) {
	got, err := ParseSubjectID(json.RawMessage(
		`{"format":"aliases","identifiers":[` +
			`{"format":"email","email":"a@b.com"},` +
			`{"format":"iss_sub","iss":"https://i","sub":"u1"}]}`))
	require.NoError(t, err)
	assert.Equal(t, "aliases", got.Format)
	require.Len(t, got.Identifiers, 2)
	assert.Equal(t, "a@b.com", got.Identifiers[0].Email)
	assert.Equal(t, "u1", got.Identifiers[1].Subject)
}

// Nested aliases are forbidden by RFC 9493 and would let a sender build an
// arbitrarily deep structure for us to walk.
func TestParseSubjectID_NestedAliasesRejected(t *testing.T) {
	_, err := ParseSubjectID(json.RawMessage(
		`{"format":"aliases","identifiers":[{"format":"aliases","identifiers":[]}]}`))
	require.Error(t, err)
}

func TestParseSubjectID_Malformed(t *testing.T) {
	_, err := ParseSubjectID(json.RawMessage(`"just-a-string"`))
	require.Error(t, err)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./plugins/sharedsignals/caep/ -v`
Expected: FAIL, the package does not compile because `ParseSubjectID` and `SubjectID` are undefined.

- [ ] **Step 3: Write minimal implementation**

Create `plugins/sharedsignals/caep/subject.go`:

```go
// Package caep implements the OpenID CAEP event types and the RFC 9493
// subject identifiers they carry. It has no authsome dependencies so it can
// be tested against spec fixtures on its own.
package caep

import (
	"encoding/json"
	"fmt"
)

// Subject identifier formats defined by RFC 9493.
const (
	FormatAccount     = "account"
	FormatEmail       = "email"
	FormatIssSub      = "iss_sub"
	FormatOpaque      = "opaque"
	FormatPhoneNumber = "phone_number"
	FormatDID         = "did"
	FormatURI         = "uri"
	FormatAliases     = "aliases"
)

// ComplexSubjectMembers are the member names the SSF spec defines for a
// Complex Subject. Any other member name is carried but not interpreted.
var ComplexSubjectMembers = []string{
	"user", "device", "session", "application", "tenant", "org_unit", "group",
}

// SubjectID is an RFC 9493 subject identifier. A simple subject sets Format
// and the members that format requires. A complex subject sets Members
// instead, with Format empty.
type SubjectID struct {
	Format      string
	Email       string
	PhoneNumber string
	Issuer      string
	Subject     string
	ID          string
	URI         string
	URL         string
	Identifiers []SubjectID
	Members     map[string]SubjectID
}

// IsComplex reports whether this is a Complex Subject, meaning it carries
// named members rather than a format of its own.
func (s SubjectID) IsComplex() bool { return s.Format == "" && len(s.Members) > 0 }

// Member returns the named member of a Complex Subject.
func (s SubjectID) Member(name string) (SubjectID, bool) {
	m, ok := s.Members[name]
	return m, ok
}

type subjectWire struct {
	Format      string            `json:"format"`
	Email       string            `json:"email"`
	PhoneNumber string            `json:"phone_number"`
	Issuer      string            `json:"iss"`
	Subject     string            `json:"sub"`
	ID          string            `json:"id"`
	URI         string            `json:"uri"`
	URL         string            `json:"url"`
	Identifiers []json.RawMessage `json:"identifiers"`
}

// ParseSubjectID decodes a subject identifier, simple or complex.
func ParseSubjectID(raw json.RawMessage) (SubjectID, error) {
	return parseSubjectID(raw, 0)
}

func parseSubjectID(raw json.RawMessage, depth int) (SubjectID, error) {
	if depth > 1 {
		return SubjectID{}, fmt.Errorf("caep: subject nested too deeply")
	}

	var w subjectWire
	if err := json.Unmarshal(raw, &w); err != nil {
		return SubjectID{}, fmt.Errorf("caep: decode subject: %w", err)
	}

	s := SubjectID{
		Format:      w.Format,
		Email:       w.Email,
		PhoneNumber: w.PhoneNumber,
		Issuer:      w.Issuer,
		Subject:     w.Subject,
		ID:          w.ID,
		URI:         w.URI,
		URL:         w.URL,
	}

	if w.Format == FormatAliases {
		for _, item := range w.Identifiers {
			alias, err := parseSubjectID(item, depth+1)
			if err != nil {
				return SubjectID{}, err
			}
			if alias.Format == FormatAliases {
				return SubjectID{}, fmt.Errorf("caep: aliases may not nest aliases")
			}
			s.Identifiers = append(s.Identifiers, alias)
		}
		return s, nil
	}

	if w.Format != "" {
		return s, nil
	}

	// No format member: treat it as a Complex Subject and decode the members
	// we recognise. Unknown member names are ignored here; the receiver
	// enforces critical_subject_members separately.
	var members map[string]json.RawMessage
	if err := json.Unmarshal(raw, &members); err != nil {
		return SubjectID{}, fmt.Errorf("caep: decode complex subject: %w", err)
	}
	for _, name := range ComplexSubjectMembers {
		item, ok := members[name]
		if !ok {
			continue
		}
		member, err := parseSubjectID(item, depth+1)
		if err != nil {
			return SubjectID{}, err
		}
		if s.Members == nil {
			s.Members = make(map[string]SubjectID, len(members))
		}
		s.Members[name] = member
	}
	if len(s.Members) == 0 {
		return SubjectID{}, fmt.Errorf("caep: subject has no format and no known members")
	}
	return s, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./plugins/sharedsignals/caep/ -v`
Expected: PASS, six tests.

- [ ] **Step 5: Commit**

```bash
git add plugins/sharedsignals/caep/
git commit -m "feat(sharedsignals): RFC 9493 subject identifier parsing"
```

---

### Task 2: caep package, event payloads

**Files:**
- Create: `plugins/sharedsignals/caep/events.go`
- Test: `plugins/sharedsignals/caep/events_test.go`

**Interfaces:**
- Consumes: `caep.SubjectID`, `caep.ParseSubjectID` from Task 1.
- Produces: the seven event type constants; `caep.Event` struct; `caep.ParseEvent(eventType string, payload json.RawMessage) (Event, error)`.

The `subject` versus `sub_id` split matters here. CAEP 1.0 final specifies `sub_id`, but Okta's published SET payloads and Google's RISC events both use the older `subject`. A receiver that reads only `sub_id` silently ignores every event Okta sends, so accept both and prefer `sub_id`.

- [ ] **Step 1: Write the failing test**

Create `plugins/sharedsignals/caep/events_test.go`:

```go
package caep

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The payload shape Okta actually emits, which uses "subject" rather than
// the "sub_id" the final CAEP spec names.
func TestParseEvent_OktaSessionRevoked(t *testing.T) {
	payload := json.RawMessage(`{
		"subject": {"format":"iss_sub","iss":"https://org.okta.com","sub":"okta-user-id1"},
		"reason_admin": {"en":"User logout from Okta"},
		"event_timestamp": 1615304991643
	}`)

	ev, err := ParseEvent(EventSessionRevoked, payload)
	require.NoError(t, err)
	assert.Equal(t, EventSessionRevoked, ev.Type)
	assert.Equal(t, "okta-user-id1", ev.Subject.Subject)
	assert.Equal(t, "User logout from Okta", ev.ReasonAdmin["en"])
	assert.Equal(t, int64(1615304991643), ev.EventTimestamp)
}

func TestParseEvent_SubIDPreferredOverSubject(t *testing.T) {
	payload := json.RawMessage(`{
		"sub_id": {"format":"opaque","id":"from-sub-id"},
		"subject": {"format":"opaque","id":"from-subject"}
	}`)

	ev, err := ParseEvent(EventSessionRevoked, payload)
	require.NoError(t, err)
	assert.Equal(t, "from-sub-id", ev.Subject.ID)
}

func TestParseEvent_CredentialChange(t *testing.T) {
	payload := json.RawMessage(`{
		"subject": {"format":"iss_sub","iss":"https://i","sub":"u1"},
		"credential_type": "fido2-roaming",
		"change_type": "create",
		"friendly_name": "FIDO_WEBAUTHN",
		"initiating_entity": "user"
	}`)

	ev, err := ParseEvent(EventCredentialChange, payload)
	require.NoError(t, err)
	assert.Equal(t, "fido2-roaming", ev.CredentialType)
	assert.Equal(t, "create", ev.ChangeType)
	assert.Equal(t, "user", ev.InitiatingEntity)
}

func TestParseEvent_AssuranceLevelChange(t *testing.T) {
	payload := json.RawMessage(`{
		"subject": {"format":"opaque","id":"u1"},
		"namespace": "NIST-AAL",
		"current_level": "aal1",
		"previous_level": "aal2",
		"change_direction": "decrease"
	}`)

	ev, err := ParseEvent(EventAssuranceLevelChange, payload)
	require.NoError(t, err)
	assert.Equal(t, "decrease", ev.ChangeDirection)
	assert.Equal(t, "aal1", ev.CurrentLevel)
	assert.Equal(t, "aal2", ev.PreviousLevel)
}

func TestParseEvent_DeviceComplianceChange(t *testing.T) {
	payload := json.RawMessage(`{
		"subject": {"format":"opaque","id":"u1"},
		"previous_status": "compliant",
		"current_status": "not-compliant"
	}`)

	ev, err := ParseEvent(EventDeviceComplianceChange, payload)
	require.NoError(t, err)
	assert.Equal(t, "not-compliant", ev.CurrentStatus)
	assert.Equal(t, "compliant", ev.PreviousStatus)
}

func TestParseEvent_TokenClaimsChange(t *testing.T) {
	payload := json.RawMessage(`{
		"subject": {"format":"opaque","id":"u1"},
		"claims": {"role":"admin"}
	}`)

	ev, err := ParseEvent(EventTokenClaimsChange, payload)
	require.NoError(t, err)
	assert.Equal(t, "admin", ev.Claims["role"])
}

// The SSF verification event carries the stream_id as an opaque subject and a
// state the receiver chose.
func TestParseEvent_Verification(t *testing.T) {
	payload := json.RawMessage(`{
		"sub_id": {"format":"opaque","id":"stream-1"},
		"state": "abc123"
	}`)

	ev, err := ParseEvent(EventVerification, payload)
	require.NoError(t, err)
	assert.Equal(t, "abc123", ev.State)
	assert.Equal(t, "stream-1", ev.Subject.ID)
}

func TestParseEvent_MissingSubject(t *testing.T) {
	_, err := ParseEvent(EventSessionRevoked, json.RawMessage(`{"reason_admin":{"en":"x"}}`))
	require.Error(t, err)
}

func TestIsKnownEventType(t *testing.T) {
	assert.True(t, IsKnownEventType(EventSessionRevoked))
	assert.False(t, IsKnownEventType("https://example.com/not-a-caep-event"))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./plugins/sharedsignals/caep/ -run TestParseEvent -v`
Expected: FAIL, `ParseEvent` and the event constants are undefined.

- [ ] **Step 3: Write minimal implementation**

Create `plugins/sharedsignals/caep/events.go`:

```go
package caep

import (
	"encoding/json"
	"fmt"
)

// CAEP and SSF event type URIs.
const (
	EventSessionRevoked         = "https://schemas.openid.net/secevent/caep/event-type/session-revoked"
	EventTokenClaimsChange      = "https://schemas.openid.net/secevent/caep/event-type/token-claims-change"
	EventCredentialChange       = "https://schemas.openid.net/secevent/caep/event-type/credential-change"
	EventAssuranceLevelChange   = "https://schemas.openid.net/secevent/caep/event-type/assurance-level-change"
	EventDeviceComplianceChange = "https://schemas.openid.net/secevent/caep/event-type/device-compliance-change"
	EventRiskLevelChange        = "https://schemas.openid.net/secevent/caep/event-type/risk-level-change"
	EventVerification           = "https://schemas.openid.net/secevent/ssf/event-type/verification"
)

var knownEventTypes = map[string]struct{}{
	EventSessionRevoked:         {},
	EventTokenClaimsChange:      {},
	EventCredentialChange:       {},
	EventAssuranceLevelChange:   {},
	EventDeviceComplianceChange: {},
	EventRiskLevelChange:        {},
	EventVerification:           {},
}

// IsKnownEventType reports whether this build understands the event type.
func IsKnownEventType(t string) bool {
	_, ok := knownEventTypes[t]
	return ok
}

// Event is one decoded event payload from a SET's events map.
type Event struct {
	Type             string
	Subject          SubjectID
	EventTimestamp   int64
	InitiatingEntity string
	ReasonAdmin      map[string]string
	ReasonUser       map[string]string

	// credential-change
	CredentialType string
	ChangeType     string
	FriendlyName   string

	// assurance-level-change
	Namespace       string
	CurrentLevel    string
	PreviousLevel   string
	ChangeDirection string

	// device-compliance-change
	CurrentStatus  string
	PreviousStatus string

	// token-claims-change
	Claims map[string]any

	// risk-level-change
	RiskReason string
	Principal  string

	// SSF verification
	State string
}

type eventWire struct {
	SubID            json.RawMessage   `json:"sub_id"`
	Subject          json.RawMessage   `json:"subject"`
	EventTimestamp   int64             `json:"event_timestamp"`
	InitiatingEntity string            `json:"initiating_entity"`
	ReasonAdmin      map[string]string `json:"reason_admin"`
	ReasonUser       map[string]string `json:"reason_user"`
	CredentialType   string            `json:"credential_type"`
	ChangeType       string            `json:"change_type"`
	FriendlyName     string            `json:"friendly_name"`
	Namespace        string            `json:"namespace"`
	CurrentLevel     string            `json:"current_level"`
	PreviousLevel    string            `json:"previous_level"`
	ChangeDirection  string            `json:"change_direction"`
	CurrentStatus    string            `json:"current_status"`
	PreviousStatus   string            `json:"previous_status"`
	Claims           map[string]any    `json:"claims"`
	RiskReason       string            `json:"risk_reason"`
	Principal        string            `json:"principal"`
	State            string            `json:"state"`
}

// ParseEvent decodes one event payload. It accepts the subject under either
// `sub_id` (CAEP 1.0 final) or `subject` (what Okta and Google ship today),
// preferring `sub_id` when a payload carries both.
func ParseEvent(eventType string, payload json.RawMessage) (Event, error) {
	var w eventWire
	if err := json.Unmarshal(payload, &w); err != nil {
		return Event{}, fmt.Errorf("caep: decode event payload: %w", err)
	}

	rawSubject := w.SubID
	if len(rawSubject) == 0 {
		rawSubject = w.Subject
	}
	if len(rawSubject) == 0 {
		return Event{}, fmt.Errorf("caep: event has neither sub_id nor subject")
	}

	subject, err := ParseSubjectID(rawSubject)
	if err != nil {
		return Event{}, err
	}

	return Event{
		Type:             eventType,
		Subject:          subject,
		EventTimestamp:   w.EventTimestamp,
		InitiatingEntity: w.InitiatingEntity,
		ReasonAdmin:      w.ReasonAdmin,
		ReasonUser:       w.ReasonUser,
		CredentialType:   w.CredentialType,
		ChangeType:       w.ChangeType,
		FriendlyName:     w.FriendlyName,
		Namespace:        w.Namespace,
		CurrentLevel:     w.CurrentLevel,
		PreviousLevel:    w.PreviousLevel,
		ChangeDirection:  w.ChangeDirection,
		CurrentStatus:    w.CurrentStatus,
		PreviousStatus:   w.PreviousStatus,
		Claims:           w.Claims,
		RiskReason:       w.RiskReason,
		Principal:        w.Principal,
		State:            w.State,
	}, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./plugins/sharedsignals/caep/ -v`
Expected: PASS, all tests in the package.

- [ ] **Step 5: Commit**

```bash
git add plugins/sharedsignals/caep/
git commit -m "feat(sharedsignals): CAEP event payload parsing"
```

---

### Task 3: setjwt package, SET validation

**Files:**
- Create: `plugins/sharedsignals/setjwt/setjwt.go`
- Test: `plugins/sharedsignals/setjwt/setjwt_test.go`

**Interfaces:**
- Consumes: nothing from earlier tasks.
- Produces: `setjwt.Token{Issuer string, Audience []string, JTI string, IssuedAt time.Time, Events map[string]json.RawMessage}`; `setjwt.KeyResolver` interface with `Key(ctx context.Context, kid string) (crypto.PublicKey, error)`; `setjwt.Options{Issuer, Audience string, Keys KeyResolver, Now func() time.Time, MaxAge, ClockSkew time.Duration, MaxEvents int}`; `setjwt.Validate(ctx context.Context, raw []byte, opts Options) (*Token, error)`; error sentinels `ErrInvalidRequest, ErrInvalidKey, ErrInvalidIssuer, ErrInvalidAudience`; `setjwt.ErrCode(err error) string` mapping a sentinel to its RFC 8935 code.

This is the package that decides whether a stranger gets to end sessions, so the tests are the point. Every negative case below corresponds to a real attack.

- [ ] **Step 1: Write the failing test**

Create `plugins/sharedsignals/setjwt/setjwt_test.go`:

```go
package setjwt

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testIssuer = "https://org.okta.com"
	testAud    = "https://authsome.example/ssf"
	testKID    = "kid-1"
)

var testNow = time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)

type staticKeys struct {
	key crypto.PublicKey
	err error
}

func (s staticKeys) Key(_ context.Context, _ string) (crypto.PublicKey, error) {
	return s.key, s.err
}

func newTestKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	k, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	return k
}

type claimOverride func(jwt.MapClaims)

func signSET(t *testing.T, key *rsa.PrivateKey, typ string, method jwt.SigningMethod,
	secret any, overrides ...claimOverride) []byte {
	t.Helper()
	claims := jwt.MapClaims{
		"iss": testIssuer,
		"aud": testAud,
		"jti": "jti-1",
		"iat": testNow.Unix(),
		"events": map[string]any{
			"https://schemas.openid.net/secevent/caep/event-type/session-revoked": map[string]any{
				"subject": map[string]any{"format": "opaque", "id": "u1"},
			},
		},
	}
	for _, o := range overrides {
		o(claims)
	}
	tok := jwt.NewWithClaims(method, claims)
	tok.Header["typ"] = typ
	tok.Header["kid"] = testKID
	if secret == nil {
		secret = key
	}
	s, err := tok.SignedString(secret)
	require.NoError(t, err)
	return []byte(s)
}

func baseOpts(key *rsa.PrivateKey) Options {
	return Options{
		Issuer:    testIssuer,
		Audience:  testAud,
		Keys:      staticKeys{key: key.Public()},
		Now:       func() time.Time { return testNow },
		MaxAge:    24 * time.Hour,
		ClockSkew: 5 * time.Minute,
		MaxEvents: 10,
	}
}

func TestValidate_Accepts(t *testing.T) {
	key := newTestKey(t)
	raw := signSET(t, key, "secevent+jwt", jwt.SigningMethodRS256, nil)

	tok, err := Validate(context.Background(), raw, baseOpts(key))
	require.NoError(t, err)
	assert.Equal(t, testIssuer, tok.Issuer)
	assert.Equal(t, "jti-1", tok.JTI)
	assert.Len(t, tok.Events, 1)
}

func TestHeader_ReadsKidAndAlg(t *testing.T) {
	key := newTestKey(t)
	raw := signSET(t, key, "secevent+jwt", jwt.SigningMethodRS256, nil)

	kid, alg, typ, err := Header(raw)
	require.NoError(t, err)
	assert.Equal(t, testKID, kid)
	assert.Equal(t, "RS256", alg)
	assert.Equal(t, "secevent+jwt", typ)
}

// alg:none is the oldest JWT attack there is.
func TestValidate_RejectsAlgNone(t *testing.T) {
	key := newTestKey(t)
	tok := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
		"iss": testIssuer, "aud": testAud, "jti": "j", "iat": testNow.Unix(),
		"events": map[string]any{"x": map[string]any{}},
	})
	tok.Header["typ"] = "secevent+jwt"
	s, err := tok.SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)

	_, err = Validate(context.Background(), []byte(s), baseOpts(key))
	require.ErrorIs(t, err, ErrInvalidKey)
}

// Algorithm confusion: sign with HMAC using the issuer's PUBLIC key as the
// shared secret. If we accepted HS*, this forgery would verify.
func TestValidate_RejectsHMACConfusion(t *testing.T) {
	key := newTestKey(t)
	pubDER, err := json.Marshal(testKID) // any attacker-known bytes work
	require.NoError(t, err)
	raw := signSET(t, key, "secevent+jwt", jwt.SigningMethodHS256, pubDER)

	_, err = Validate(context.Background(), raw, baseOpts(key))
	require.ErrorIs(t, err, ErrInvalidKey)
}

func TestValidate_RejectsWrongTyp(t *testing.T) {
	key := newTestKey(t)
	raw := signSET(t, key, "JWT", jwt.SigningMethodRS256, nil)

	_, err := Validate(context.Background(), raw, baseOpts(key))
	require.ErrorIs(t, err, ErrInvalidRequest)
}

func TestValidate_RejectsWrongIssuer(t *testing.T) {
	key := newTestKey(t)
	raw := signSET(t, key, "secevent+jwt", jwt.SigningMethodRS256, nil,
		func(c jwt.MapClaims) { c["iss"] = "https://evil.example" })

	_, err := Validate(context.Background(), raw, baseOpts(key))
	require.ErrorIs(t, err, ErrInvalidIssuer)
}

func TestValidate_RejectsWrongAudience(t *testing.T) {
	key := newTestKey(t)
	raw := signSET(t, key, "secevent+jwt", jwt.SigningMethodRS256, nil,
		func(c jwt.MapClaims) { c["aud"] = "https://someone-else.example" })

	_, err := Validate(context.Background(), raw, baseOpts(key))
	require.ErrorIs(t, err, ErrInvalidAudience)
}

func TestValidate_AcceptsAudienceArray(t *testing.T) {
	key := newTestKey(t)
	raw := signSET(t, key, "secevent+jwt", jwt.SigningMethodRS256, nil,
		func(c jwt.MapClaims) { c["aud"] = []string{"https://other", testAud} })

	_, err := Validate(context.Background(), raw, baseOpts(key))
	require.NoError(t, err)
}

func TestValidate_RejectsStaleIAT(t *testing.T) {
	key := newTestKey(t)
	raw := signSET(t, key, "secevent+jwt", jwt.SigningMethodRS256, nil,
		func(c jwt.MapClaims) { c["iat"] = testNow.Add(-25 * time.Hour).Unix() })

	_, err := Validate(context.Background(), raw, baseOpts(key))
	require.ErrorIs(t, err, ErrInvalidRequest)
}

func TestValidate_RejectsFutureIAT(t *testing.T) {
	key := newTestKey(t)
	raw := signSET(t, key, "secevent+jwt", jwt.SigningMethodRS256, nil,
		func(c jwt.MapClaims) { c["iat"] = testNow.Add(10 * time.Minute).Unix() })

	_, err := Validate(context.Background(), raw, baseOpts(key))
	require.ErrorIs(t, err, ErrInvalidRequest)
}

func TestValidate_RejectsMissingJTI(t *testing.T) {
	key := newTestKey(t)
	raw := signSET(t, key, "secevent+jwt", jwt.SigningMethodRS256, nil,
		func(c jwt.MapClaims) { delete(c, "jti") })

	_, err := Validate(context.Background(), raw, baseOpts(key))
	require.ErrorIs(t, err, ErrInvalidRequest)
}

func TestValidate_RejectsOverlongJTI(t *testing.T) {
	key := newTestKey(t)
	long := make([]byte, MaxJTILength+1)
	for i := range long {
		long[i] = 'a'
	}
	raw := signSET(t, key, "secevent+jwt", jwt.SigningMethodRS256, nil,
		func(c jwt.MapClaims) { c["jti"] = string(long) })

	_, err := Validate(context.Background(), raw, baseOpts(key))
	require.ErrorIs(t, err, ErrInvalidRequest)
}

func TestValidate_RejectsEmptyEvents(t *testing.T) {
	key := newTestKey(t)
	raw := signSET(t, key, "secevent+jwt", jwt.SigningMethodRS256, nil,
		func(c jwt.MapClaims) { c["events"] = map[string]any{} })

	_, err := Validate(context.Background(), raw, baseOpts(key))
	require.ErrorIs(t, err, ErrInvalidRequest)
}

func TestValidate_RejectsTooManyEvents(t *testing.T) {
	key := newTestKey(t)
	many := map[string]any{}
	for i := 0; i < 25; i++ {
		many["https://example.com/e"+string(rune('a'+i))] = map[string]any{}
	}
	raw := signSET(t, key, "secevent+jwt", jwt.SigningMethodRS256, nil,
		func(c jwt.MapClaims) { c["events"] = many })

	_, err := Validate(context.Background(), raw, baseOpts(key))
	require.ErrorIs(t, err, ErrInvalidRequest)
}

func TestValidate_RejectsUnknownKid(t *testing.T) {
	key := newTestKey(t)
	raw := signSET(t, key, "secevent+jwt", jwt.SigningMethodRS256, nil)

	opts := baseOpts(key)
	opts.Keys = staticKeys{err: assertKeyMiss{}}

	_, err := Validate(context.Background(), raw, opts)
	require.ErrorIs(t, err, ErrInvalidKey)
}

// Signed by a key that is not the one the resolver hands back.
func TestValidate_RejectsWrongSigningKey(t *testing.T) {
	signer := newTestKey(t)
	other := newTestKey(t)
	raw := signSET(t, signer, "secevent+jwt", jwt.SigningMethodRS256, nil)

	opts := baseOpts(signer)
	opts.Keys = staticKeys{key: other.Public()}

	_, err := Validate(context.Background(), raw, opts)
	require.ErrorIs(t, err, ErrInvalidKey)
}

type assertKeyMiss struct{}

func (assertKeyMiss) Error() string { return "no such kid" }

func TestErrCode(t *testing.T) {
	assert.Equal(t, "invalid_request", ErrCode(ErrInvalidRequest))
	assert.Equal(t, "invalid_key", ErrCode(ErrInvalidKey))
	assert.Equal(t, "invalid_issuer", ErrCode(ErrInvalidIssuer))
	assert.Equal(t, "invalid_audience", ErrCode(ErrInvalidAudience))
	assert.Equal(t, "invalid_request", ErrCode(assertKeyMiss{}))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./plugins/sharedsignals/setjwt/ -v`
Expected: FAIL, the package does not compile.

- [ ] **Step 3: Write minimal implementation**

Create `plugins/sharedsignals/setjwt/setjwt.go`:

```go
// Package setjwt parses and validates RFC 8417 Security Event Tokens. It has
// no authsome dependencies. Errors are the RFC 8935 sentinels so a caller can
// turn a failure straight into a push-endpoint error body.
package setjwt

import (
	"context"
	"crypto"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// MaxJTILength bounds the jti we are willing to store as a dedupe key.
const MaxJTILength = 255

// RFC 8935 error sentinels. ErrCode turns one into its wire code.
var (
	ErrInvalidRequest  = errors.New("setjwt: invalid_request")
	ErrInvalidKey      = errors.New("setjwt: invalid_key")
	ErrInvalidIssuer   = errors.New("setjwt: invalid_issuer")
	ErrInvalidAudience = errors.New("setjwt: invalid_audience")
)

// ErrCode maps an error to its RFC 8935 code, defaulting to invalid_request.
func ErrCode(err error) string {
	switch {
	case errors.Is(err, ErrInvalidKey):
		return "invalid_key"
	case errors.Is(err, ErrInvalidIssuer):
		return "invalid_issuer"
	case errors.Is(err, ErrInvalidAudience):
		return "invalid_audience"
	default:
		return "invalid_request"
	}
}

// allowedAlgs is the signature algorithm allow-list. HMAC is absent on
// purpose: accepting a symmetric algorithm alongside a published public key
// is the classic JWT confusion attack.
var allowedAlgs = map[string]struct{}{
	"RS256": {}, "RS384": {}, "RS512": {},
	"ES256": {}, "ES384": {}, "ES512": {},
	"EdDSA": {},
}

// KeyResolver hands back the public key for a kid.
type KeyResolver interface {
	Key(ctx context.Context, kid string) (crypto.PublicKey, error)
}

// Token is a validated SET.
type Token struct {
	Issuer   string
	Audience []string
	JTI      string
	IssuedAt time.Time
	Events   map[string]json.RawMessage
}

// Options configures Validate. Every field is required.
type Options struct {
	Issuer    string
	Audience  string
	Keys      KeyResolver
	Now       func() time.Time
	MaxAge    time.Duration
	ClockSkew time.Duration
	MaxEvents int
}

// Header reads kid, alg and typ without verifying anything. Use it only to
// pick a key and to reject an algorithm before doing real work.
func Header(raw []byte) (kid, alg, typ string, err error) {
	parser := jwt.NewParser()
	tok, _, err := parser.ParseUnverified(string(raw), jwt.MapClaims{})
	if err != nil {
		return "", "", "", fmt.Errorf("%w: parse header: %v", ErrInvalidRequest, err)
	}
	kid, _ = tok.Header["kid"].(string)
	alg, _ = tok.Header["alg"].(string)
	typ, _ = tok.Header["typ"].(string)
	return kid, alg, typ, nil
}

type setClaims struct {
	Issuer   string                     `json:"iss"`
	Audience any                        `json:"aud"`
	JTI      string                     `json:"jti"`
	IssuedAt int64                      `json:"iat"`
	Events   map[string]json.RawMessage `json:"events"`
}

func (setClaims) GetExpirationTime() (*jwt.NumericDate, error) { return nil, nil }
func (setClaims) GetIssuedAt() (*jwt.NumericDate, error)       { return nil, nil }
func (setClaims) GetNotBefore() (*jwt.NumericDate, error)      { return nil, nil }
func (c setClaims) GetIssuer() (string, error)                 { return c.Issuer, nil }
func (setClaims) GetSubject() (string, error)                  { return "", nil }
func (setClaims) GetAudience() (jwt.ClaimStrings, error)       { return nil, nil }

// Validate verifies a SET end to end and returns it. The claim checks run
// after signature verification, so nothing in the token is trusted until the
// key that the caller pinned has vouched for it.
func Validate(ctx context.Context, raw []byte, opts Options) (*Token, error) {
	kid, alg, typ, err := Header(raw)
	if err != nil {
		return nil, err
	}
	if typ != "secevent+jwt" {
		return nil, fmt.Errorf("%w: typ header must be secevent+jwt", ErrInvalidRequest)
	}
	if _, ok := allowedAlgs[alg]; !ok {
		return nil, fmt.Errorf("%w: algorithm %q is not accepted", ErrInvalidKey, alg)
	}

	key, err := opts.Keys.Key(ctx, kid)
	if err != nil {
		return nil, fmt.Errorf("%w: no key for kid", ErrInvalidKey)
	}

	var claims setClaims
	parser := jwt.NewParser(
		jwt.WithValidMethods(algList()),
		jwt.WithoutClaimsValidation(),
	)
	if _, err := parser.ParseWithClaims(string(raw), &claims,
		func(*jwt.Token) (any, error) { return key, nil }); err != nil {
		return nil, fmt.Errorf("%w: signature verification failed", ErrInvalidKey)
	}

	if claims.Issuer != opts.Issuer {
		return nil, fmt.Errorf("%w: issuer mismatch", ErrInvalidIssuer)
	}

	audience, err := normalizeAudience(claims.Audience)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidAudience, err)
	}
	if !contains(audience, opts.Audience) {
		return nil, fmt.Errorf("%w: audience mismatch", ErrInvalidAudience)
	}

	if claims.JTI == "" {
		return nil, fmt.Errorf("%w: jti is required", ErrInvalidRequest)
	}
	if len(claims.JTI) > MaxJTILength {
		return nil, fmt.Errorf("%w: jti is too long", ErrInvalidRequest)
	}

	if claims.IssuedAt == 0 {
		return nil, fmt.Errorf("%w: iat is required", ErrInvalidRequest)
	}
	now := opts.Now()
	issued := time.Unix(claims.IssuedAt, 0)
	if issued.Before(now.Add(-opts.MaxAge)) {
		return nil, fmt.Errorf("%w: iat is too old", ErrInvalidRequest)
	}
	if issued.After(now.Add(opts.ClockSkew)) {
		return nil, fmt.Errorf("%w: iat is in the future", ErrInvalidRequest)
	}

	if len(claims.Events) == 0 {
		return nil, fmt.Errorf("%w: events must not be empty", ErrInvalidRequest)
	}
	if len(claims.Events) > opts.MaxEvents {
		return nil, fmt.Errorf("%w: too many events in one SET", ErrInvalidRequest)
	}

	return &Token{
		Issuer:   claims.Issuer,
		Audience: audience,
		JTI:      claims.JTI,
		IssuedAt: issued,
		Events:   claims.Events,
	}, nil
}

func algList() []string {
	out := make([]string, 0, len(allowedAlgs))
	for a := range allowedAlgs {
		out = append(out, a)
	}
	return out
}

func normalizeAudience(v any) ([]string, error) {
	switch t := v.(type) {
	case string:
		return []string{t}, nil
	case []any:
		out := make([]string, 0, len(t))
		for _, item := range t {
			s, ok := item.(string)
			if !ok {
				return nil, errors.New("audience array holds a non-string")
			}
			out = append(out, s)
		}
		return out, nil
	case nil:
		return nil, errors.New("aud is required")
	default:
		return nil, errors.New("aud must be a string or an array of strings")
	}
}

func contains(haystack []string, needle string) bool {
	for _, h := range haystack {
		if h == needle {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./plugins/sharedsignals/setjwt/ -v`
Expected: PASS. If `TestValidate_RejectsHMACConfusion` fails, the algorithm allow-list is being applied after key resolution rather than before; move the check back above the `opts.Keys.Key` call.

- [ ] **Step 5: Run with the race detector**

Run: `go test ./plugins/sharedsignals/setjwt/ -race`
Expected: PASS, no data races.

- [ ] **Step 6: Commit**

```bash
git add plugins/sharedsignals/setjwt/
git commit -m "feat(sharedsignals): RFC 8417 security event token validation"
```

---

### Task 4: jwksclient package

**Files:**
- Create: `plugins/sharedsignals/jwksclient/client.go`
- Test: `plugins/sharedsignals/jwksclient/client_test.go`

**Interfaces:**
- Consumes: nothing from earlier tasks.
- Produces: `jwksclient.Options{HTTPClient *http.Client, MinRefetchInterval time.Duration, MaxResponseBytes int64, MaxKeys int, Now func() time.Time}`; `jwksclient.New(opts Options) *Client`; `func (c *Client) Key(ctx context.Context, jwksURI, kid string) (crypto.PublicKey, error)`; `jwksclient.ValidateURI(rawURL string) error`.

This is the only component that makes an outbound request because an inbound one arrived, so every limit here exists to stop a stranger driving our network behaviour.

- [ ] **Step 1: Write the failing test**

Create `plugins/sharedsignals/jwksclient/client_test.go`:

```go
package jwksclient

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func rsaJWKS(t *testing.T, kid string, pub *rsa.PublicKey) string {
	t.Helper()
	n := base64.RawURLEncoding.EncodeToString(pub.N.Bytes())
	e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes())
	body, err := json.Marshal(map[string]any{
		"keys": []map[string]string{
			{"kty": "RSA", "use": "sig", "alg": "RS256", "kid": kid, "n": n, "e": e},
		},
	})
	require.NoError(t, err)
	return string(body)
}

func newKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	k, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	return k
}

func testOptions(srv *httptest.Server) Options {
	return Options{
		HTTPClient:         srv.Client(),
		MinRefetchInterval: time.Minute,
		MaxResponseBytes:   256 * 1024,
		MaxKeys:            20,
		Now:                time.Now,
	}
}

func TestKey_FetchesAndReturnsKey(t *testing.T) {
	key := newKey(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, rsaJWKS(t, "kid-1", &key.PublicKey))
	}))
	defer srv.Close()

	c := New(testOptions(srv))
	got, err := c.Key(context.Background(), srv.URL, "kid-1")
	require.NoError(t, err)

	pub, ok := got.(*rsa.PublicKey)
	require.True(t, ok)
	assert.Equal(t, key.PublicKey.N, pub.N)
}

func TestKey_CachesBetweenCalls(t *testing.T) {
	key := newKey(t)
	var hits int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt64(&hits, 1)
		fmt.Fprint(w, rsaJWKS(t, "kid-1", &key.PublicKey))
	}))
	defer srv.Close()

	c := New(testOptions(srv))
	for i := 0; i < 5; i++ {
		_, err := c.Key(context.Background(), srv.URL, "kid-1")
		require.NoError(t, err)
	}
	assert.Equal(t, int64(1), atomic.LoadInt64(&hits))
}

// An unknown kid must not let a caller drive our fetch rate. The first miss
// may refetch once; every miss inside MinRefetchInterval must not.
func TestKey_UnknownKidRespectsRefetchInterval(t *testing.T) {
	key := newKey(t)
	var hits int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt64(&hits, 1)
		fmt.Fprint(w, rsaJWKS(t, "kid-1", &key.PublicKey))
	}))
	defer srv.Close()

	c := New(testOptions(srv))
	for i := 0; i < 10; i++ {
		_, err := c.Key(context.Background(), srv.URL, "unknown-kid")
		require.Error(t, err)
	}
	assert.LessOrEqual(t, atomic.LoadInt64(&hits), int64(1),
		"unknown kid drove more than one fetch")
}

func TestKey_RejectsOversizedDocument(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"keys":[`+strings.Repeat(`{"kty":"oct"},`, 100_000)+`{"kty":"oct"}]}`)
	}))
	defer srv.Close()

	opts := testOptions(srv)
	opts.MaxResponseBytes = 1024
	c := New(opts)

	_, err := c.Key(context.Background(), srv.URL, "kid-1")
	require.Error(t, err)
}

func TestKey_RejectsTooManyKeys(t *testing.T) {
	key := newKey(t)
	n := base64.RawURLEncoding.EncodeToString(key.PublicKey.N.Bytes())
	e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.PublicKey.E)).Bytes())
	var entries []string
	for i := 0; i < 50; i++ {
		entries = append(entries, fmt.Sprintf(
			`{"kty":"RSA","kid":"k%d","n":%q,"e":%q}`, i, n, e))
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"keys":[`+strings.Join(entries, ",")+`]}`)
	}))
	defer srv.Close()

	opts := testOptions(srv)
	opts.MaxKeys = 20
	c := New(opts)

	_, err := c.Key(context.Background(), srv.URL, "k0")
	require.Error(t, err)
}

func TestValidateURI(t *testing.T) {
	require.NoError(t, ValidateURI("https://idp.example.com/keys"))

	// Plain HTTP would let a network attacker swap the keys.
	require.Error(t, ValidateURI("http://idp.example.com/keys"))
	// The cloud metadata endpoint is the canonical SSRF target.
	require.Error(t, ValidateURI("https://169.254.169.254/latest/meta-data/"))
	require.Error(t, ValidateURI("https://127.0.0.1/keys"))
	require.Error(t, ValidateURI("https://10.0.0.5/keys"))
	require.Error(t, ValidateURI("https://192.168.1.1/keys"))
	require.Error(t, ValidateURI("not-a-url"))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./plugins/sharedsignals/jwksclient/ -v`
Expected: FAIL, the package does not compile.

- [ ] **Step 3: Write minimal implementation**

Create `plugins/sharedsignals/jwksclient/client.go`:

```go
// Package jwksclient fetches and caches JSON Web Key Sets for inbound token
// verification. Every limit in here exists because the fetch is triggered by
// traffic we do not control.
package jwksclient

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// ErrKeyNotFound is returned when no key in the set carries the wanted kid.
var ErrKeyNotFound = errors.New("jwksclient: no key for kid")

// Options configures a Client.
type Options struct {
	HTTPClient         *http.Client
	MinRefetchInterval time.Duration
	MaxResponseBytes   int64
	MaxKeys            int
	Now                func() time.Time
}

func (o *Options) defaults() {
	if o.HTTPClient == nil {
		o.HTTPClient = &http.Client{
			Timeout: 5 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) > 0 && req.URL.Host != via[0].URL.Host {
					return errors.New("jwksclient: cross-host redirect refused")
				}
				if len(via) >= 3 {
					return errors.New("jwksclient: too many redirects")
				}
				return nil
			},
		}
	}
	if o.MinRefetchInterval == 0 {
		o.MinRefetchInterval = 5 * time.Minute
	}
	if o.MaxResponseBytes == 0 {
		o.MaxResponseBytes = 256 * 1024
	}
	if o.MaxKeys == 0 {
		o.MaxKeys = 20
	}
	if o.Now == nil {
		o.Now = time.Now
	}
}

type entry struct {
	keys        map[string]crypto.PublicKey
	lastFetched time.Time
}

// Client caches one key set per JWKS URI.
type Client struct {
	opts  Options
	mu    sync.Mutex
	cache map[string]*entry
	// inflight collapses concurrent fetches of the same URI into one request.
	inflight map[string]chan struct{}
}

// New builds a Client, filling in the hardened defaults for any zero option.
func New(opts Options) *Client {
	opts.defaults()
	return &Client{
		opts:     opts,
		cache:    make(map[string]*entry),
		inflight: make(map[string]chan struct{}),
	}
}

// Key returns the public key for kid from the set at jwksURI. A cache miss
// triggers at most one fetch per MinRefetchInterval per URI, so an unknown
// kid cannot be used to drive outbound traffic.
func (c *Client) Key(ctx context.Context, jwksURI, kid string) (crypto.PublicKey, error) {
	if key, ok := c.cachedKey(jwksURI, kid); ok {
		return key, nil
	}
	if err := c.refresh(ctx, jwksURI); err != nil {
		return nil, err
	}
	if key, ok := c.cachedKey(jwksURI, kid); ok {
		return key, nil
	}
	return nil, ErrKeyNotFound
}

func (c *Client) cachedKey(jwksURI, kid string) (crypto.PublicKey, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.cache[jwksURI]
	if !ok {
		return nil, false
	}
	key, ok := e.keys[kid]
	return key, ok
}

// refresh fetches the key set, unless another goroutine is already doing so
// or the last fetch is more recent than MinRefetchInterval.
func (c *Client) refresh(ctx context.Context, jwksURI string) error {
	c.mu.Lock()
	if e, ok := c.cache[jwksURI]; ok &&
		c.opts.Now().Sub(e.lastFetched) < c.opts.MinRefetchInterval {
		c.mu.Unlock()
		return ErrKeyNotFound
	}
	if wait, ok := c.inflight[jwksURI]; ok {
		c.mu.Unlock()
		select {
		case <-wait:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	done := make(chan struct{})
	c.inflight[jwksURI] = done
	c.mu.Unlock()

	keys, err := c.fetch(ctx, jwksURI)

	c.mu.Lock()
	delete(c.inflight, jwksURI)
	if err == nil {
		c.cache[jwksURI] = &entry{keys: keys, lastFetched: c.opts.Now()}
	} else {
		// Negative caching: a failed fetch still stamps the clock so a broken
		// endpoint cannot be hammered on every inbound request.
		c.cache[jwksURI] = &entry{keys: map[string]crypto.PublicKey{}, lastFetched: c.opts.Now()}
	}
	c.mu.Unlock()
	close(done)
	return err
}

type jwk struct {
	KTY string `json:"kty"`
	KID string `json:"kid"`
	N   string `json:"n"`
	E   string `json:"e"`
	CRV string `json:"crv"`
	X   string `json:"x"`
	Y   string `json:"y"`
}

func (c *Client) fetch(ctx context.Context, jwksURI string) (map[string]crypto.PublicKey, error) {
	if err := ValidateURI(jwksURI); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, jwksURI, nil)
	if err != nil {
		return nil, fmt.Errorf("jwksclient: build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.opts.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("jwksclient: fetch: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck // response body close

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("jwksclient: fetch returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, c.opts.MaxResponseBytes+1))
	if err != nil {
		return nil, fmt.Errorf("jwksclient: read body: %w", err)
	}
	if int64(len(body)) > c.opts.MaxResponseBytes {
		return nil, errors.New("jwksclient: key set exceeds the size limit")
	}

	var doc struct {
		Keys []jwk `json:"keys"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("jwksclient: decode key set: %w", err)
	}
	if len(doc.Keys) > c.opts.MaxKeys {
		return nil, errors.New("jwksclient: key set holds too many keys")
	}

	out := make(map[string]crypto.PublicKey, len(doc.Keys))
	for _, k := range doc.Keys {
		pub, err := k.publicKey()
		if err != nil {
			continue // skip a key we cannot use, keep the rest
		}
		out[k.KID] = pub
	}
	return out, nil
}

func (k jwk) publicKey() (crypto.PublicKey, error) {
	switch k.KTY {
	case "RSA":
		n, err := base64.RawURLEncoding.DecodeString(k.N)
		if err != nil {
			return nil, err
		}
		e, err := base64.RawURLEncoding.DecodeString(k.E)
		if err != nil {
			return nil, err
		}
		return &rsa.PublicKey{
			N: new(big.Int).SetBytes(n),
			E: int(new(big.Int).SetBytes(e).Int64()),
		}, nil
	case "EC":
		var curve elliptic.Curve
		switch k.CRV {
		case "P-256":
			curve = elliptic.P256()
		case "P-384":
			curve = elliptic.P384()
		case "P-521":
			curve = elliptic.P521()
		default:
			return nil, fmt.Errorf("jwksclient: unsupported curve %q", k.CRV)
		}
		x, err := base64.RawURLEncoding.DecodeString(k.X)
		if err != nil {
			return nil, err
		}
		y, err := base64.RawURLEncoding.DecodeString(k.Y)
		if err != nil {
			return nil, err
		}
		return &ecdsa.PublicKey{
			Curve: curve,
			X:     new(big.Int).SetBytes(x),
			Y:     new(big.Int).SetBytes(y),
		}, nil
	default:
		return nil, fmt.Errorf("jwksclient: unsupported key type %q", k.KTY)
	}
}

// ValidateURI checks a JWKS URI before it is stored on a stream. It refuses
// plain HTTP and any host that is a literal private, loopback or link-local
// address, so an operator cannot point a stream at the metadata service.
func ValidateURI(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("jwksclient: parse jwks_uri: %w", err)
	}
	if u.Scheme != "https" {
		return errors.New("jwksclient: jwks_uri must use https")
	}
	if u.Host == "" {
		return errors.New("jwksclient: jwks_uri has no host")
	}
	host := u.Hostname()
	if ip := net.ParseIP(host); ip != nil {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
			ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
			return errors.New("jwksclient: jwks_uri points at a non-public address")
		}
	}
	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./plugins/sharedsignals/jwksclient/ -v`
Expected: PASS.

- [ ] **Step 5: Run with the race detector**

The cache and the single-flight map are shared, so this one matters.

Run: `go test ./plugins/sharedsignals/jwksclient/ -race -count=3`
Expected: PASS, no data races.

- [ ] **Step 6: Commit**

```bash
git add plugins/sharedsignals/jwksclient/
git commit -m "feat(sharedsignals): hardened JWKS client for inbound verification"
```

---

### Task 5: Core ID prefixes and plugin capability interfaces

**Files:**
- Modify: `id/id.go` (prefix block near line 26, type aliases near line 84, constructors near line 270)
- Modify: `plugin/plugin.go` (optional capability interface block, after `LedgerStoreProvider`)
- Test: `id/id_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces: `id.PrefixSSFStream`, `id.PrefixSSFLink`, `id.PrefixSSFEvent`, `id.PrefixSSFSignal`; types `id.SSFStreamID`, `id.SSFLinkID`, `id.SSFEventID`, `id.SSFSignalID`; constructors `id.NewSSFStreamID()`, `id.NewSSFLinkID()`, `id.NewSSFEventID()`, `id.NewSSFSignalID()`; `plugin.SessionRevoker` and `plugin.DispatcherProvider`.

`*authsome.Engine` already has `RevokeSession` (`service.go:694`) and `Dispatcher` (`engine.go:750`), so both interfaces are satisfied without touching the engine. These are declarations that let a plugin reach those methods without importing the concrete type.

- [ ] **Step 1: Write the failing test**

Append to `id/id_test.go`:

```go
func TestSSFPrefixes(t *testing.T) {
	cases := []struct {
		name   string
		got    ID
		prefix Prefix
	}{
		{"stream", NewSSFStreamID(), PrefixSSFStream},
		{"link", NewSSFLinkID(), PrefixSSFLink},
		{"event", NewSSFEventID(), PrefixSSFEvent},
		{"signal", NewSSFSignalID(), PrefixSSFSignal},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got.Prefix() != tc.prefix {
				t.Fatalf("prefix = %q, want %q", tc.got.Prefix(), tc.prefix)
			}
			parsed, err := ParseWithPrefix(tc.got.String(), tc.prefix)
			if err != nil {
				t.Fatalf("round trip: %v", err)
			}
			if parsed.String() != tc.got.String() {
				t.Fatalf("round trip = %q, want %q", parsed, tc.got)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./id/ -run TestSSFPrefixes -v`
Expected: FAIL, `NewSSFStreamID` and the prefix constants are undefined.

- [ ] **Step 3: Add the prefixes**

In `id/id.go`, add to the `const` block that ends with `PrefixUserEmail`:

```go
	PrefixSSFStream       Prefix = "assf"
	PrefixSSFLink         Prefix = "assl"
	PrefixSSFEvent        Prefix = "asse"
	PrefixSSFSignal       Prefix = "assg"
```

Add to the type alias block that ends with `AnyID`:

```go
// SSFStreamID identifies a Shared Signals inbound stream.
type SSFStreamID = ID

// SSFLinkID identifies a Shared Signals subject link.
type SSFLinkID = ID

// SSFEventID identifies a received Security Event Token.
type SSFEventID = ID

// SSFSignalID identifies a stored Shared Signals risk signal.
type SSFSignalID = ID
```

Add to the constructor block:

```go
// NewSSFStreamID generates a new Shared Signals stream ID.
func NewSSFStreamID() ID { return New(PrefixSSFStream) }

// NewSSFLinkID generates a new Shared Signals subject link ID.
func NewSSFLinkID() ID { return New(PrefixSSFLink) }

// NewSSFEventID generates a new received-event ID.
func NewSSFEventID() ID { return New(PrefixSSFEvent) }

// NewSSFSignalID generates a new Shared Signals risk signal ID.
func NewSSFSignalID() ID { return New(PrefixSSFSignal) }
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./id/ -run TestSSFPrefixes -v`
Expected: PASS, four subtests.

- [ ] **Step 5: Add the capability interfaces**

In `plugin/plugin.go`, after the `LedgerStoreProvider` declaration:

```go
// SessionRevoker is optionally implemented by engines that can revoke a
// single session by ID. Revoking through this rather than deleting rows
// directly keeps the AfterSessionRevoke hooks, the hook bus and the outbound
// relay firing. *authsome.Engine already satisfies it.
type SessionRevoker interface {
	RevokeSession(ctx context.Context, sessionID id.SessionID) error
}

// DispatcherProvider is optionally implemented by engines that expose a
// background job queue. A plugin that needs deferred work should fall back to
// its own goroutine when the host returns nil. *authsome.Engine already
// satisfies it.
type DispatcherProvider interface {
	Dispatcher() bridge.Dispatcher
}
```

- [ ] **Step 6: Verify the engine satisfies both**

Create `plugin/capability_test.go`:

```go
package plugin_test

import (
	"testing"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/plugin"
)

// The two capability interfaces exist so plugins can reach engine methods
// without importing the concrete type. If the engine ever loses either
// method, this fails at compile time rather than at runtime in a receiver.
func TestEngineSatisfiesCapabilities(t *testing.T) {
	var _ plugin.SessionRevoker = (*authsome.Engine)(nil)
	var _ plugin.DispatcherProvider = (*authsome.Engine)(nil)
}
```

Run: `go test ./plugin/ -run TestEngineSatisfiesCapabilities -v`
Expected: PASS. A compile error here means the engine method signature drifted from the interface.

- [ ] **Step 7: Commit**

```bash
git add id/id.go id/id_test.go plugin/plugin.go plugin/capability_test.go
git commit -m "feat(plugin): shared signals ID prefixes and engine capability interfaces"
```

---

### Task 6: Store domain types and the memory store

**Files:**
- Create: `plugins/sharedsignals/store.go`
- Create: `plugins/sharedsignals/store_memory.go`
- Test: `plugins/sharedsignals/store_memory_test.go`

**Interfaces:**
- Consumes: `id.NewSSFStreamID` and friends from Task 5.
- Produces: `InboundStream`, `SubjectLink`, `ReceivedEvent`, `Signal` structs; the `Store` interface; `ErrNotFound`, `ErrDuplicateJTI`; `NewMemoryStore() *MemoryStore`; the status, outcome and enforcement constants.

- [ ] **Step 1: Write the failing test**

Create `plugins/sharedsignals/store_memory_test.go`:

```go
package sharedsignals

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
)

func TestMemoryStore_InboundStreamRoundTrip(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	appID := id.NewAppID()

	in := &InboundStream{
		ID:                    id.NewSSFStreamID(),
		AppID:                 appID,
		EnvID:                 id.NewEnvironmentID(),
		Name:                  "okta-prod",
		Issuer:                "https://org.okta.com",
		Audience:              "https://authsome.example/ssf",
		JWKSURI:               "https://org.okta.com/oauth2/v1/keys",
		PushPathHash:          "hash-abc",
		PushTokenHash:         "hash-tok",
		AllowedEventTypes:     []string{"a", "b"},
		AllowedSubjectFormats: []string{"iss_sub"},
		VerifiedDomains:       []string{"corp.com"},
		ActionOverrides:       map[string]string{"x": "signal"},
		EnforcementMode:       EnforcementEnforce,
		Status:                StatusEnabled,
		MaxActionsPerHour:     100,
	}
	require.NoError(t, s.CreateInboundStream(ctx, in))

	got, err := s.GetInboundStream(ctx, in.ID)
	require.NoError(t, err)
	assert.Equal(t, "okta-prod", got.Name)
	assert.Equal(t, []string{"iss_sub"}, got.AllowedSubjectFormats)
	assert.Equal(t, map[string]string{"x": "signal"}, got.ActionOverrides)

	byHash, err := s.GetInboundStreamByPushPathHash(ctx, "hash-abc")
	require.NoError(t, err)
	assert.Equal(t, in.ID, byHash.ID)

	_, err = s.GetInboundStreamByPushPathHash(ctx, "nope")
	require.ErrorIs(t, err, ErrNotFound)

	got.Status = StatusPaused
	require.NoError(t, s.UpdateInboundStream(ctx, got))
	after, err := s.GetInboundStream(ctx, in.ID)
	require.NoError(t, err)
	assert.Equal(t, StatusPaused, after.Status)

	list, err := s.ListInboundStreams(ctx, appID)
	require.NoError(t, err)
	require.Len(t, list, 1)

	require.NoError(t, s.DeleteInboundStream(ctx, in.ID))
	_, err = s.GetInboundStream(ctx, in.ID)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestMemoryStore_SubjectLinkUpsert(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	appID, envID := id.NewAppID(), id.NewEnvironmentID()
	userID := id.NewUserID()

	link := &SubjectLink{
		ID: id.NewSSFLinkID(), AppID: appID, EnvID: envID,
		Issuer: "https://i", Subject: "u1", UserID: userID, Source: "sso",
	}
	require.NoError(t, s.UpsertSubjectLink(ctx, link))

	got, err := s.GetSubjectLink(ctx, appID, envID, "https://i", "u1")
	require.NoError(t, err)
	assert.Equal(t, userID, got.UserID)

	// Upsert on the same tuple replaces rather than duplicating.
	other := id.NewUserID()
	require.NoError(t, s.UpsertSubjectLink(ctx, &SubjectLink{
		ID: id.NewSSFLinkID(), AppID: appID, EnvID: envID,
		Issuer: "https://i", Subject: "u1", UserID: other, Source: "sso",
	}))
	got, err = s.GetSubjectLink(ctx, appID, envID, "https://i", "u1")
	require.NoError(t, err)
	assert.Equal(t, other, got.UserID)

	// A different app must not see it.
	_, err = s.GetSubjectLink(ctx, id.NewAppID(), envID, "https://i", "u1")
	require.ErrorIs(t, err, ErrNotFound)
}

// The unique (stream_id, jti) constraint is the replay guard, so the second
// insert of a jti must be distinguishable from any other failure.
func TestMemoryStore_ReceivedEventDedupe(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	streamID := id.NewSSFStreamID()

	first := &ReceivedEvent{
		ID: id.NewSSFEventID(), StreamID: streamID, JTI: "jti-1",
		EventType: "e", Outcome: OutcomePending, ReceivedAt: time.Now(),
	}
	require.NoError(t, s.InsertReceivedEvent(ctx, first))

	err := s.InsertReceivedEvent(ctx, &ReceivedEvent{
		ID: id.NewSSFEventID(), StreamID: streamID, JTI: "jti-1",
		EventType: "e", Outcome: OutcomePending, ReceivedAt: time.Now(),
	})
	require.ErrorIs(t, err, ErrDuplicateJTI)

	// The same jti on a different stream is a different event.
	require.NoError(t, s.InsertReceivedEvent(ctx, &ReceivedEvent{
		ID: id.NewSSFEventID(), StreamID: id.NewSSFStreamID(), JTI: "jti-1",
		EventType: "e", Outcome: OutcomePending, ReceivedAt: time.Now(),
	}))

	first.Outcome = OutcomeApplied
	first.ActionTaken = "revoked_all"
	require.NoError(t, s.UpdateReceivedEvent(ctx, first))
}

func TestMemoryStore_CountActionsSince(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	streamID := id.NewSSFStreamID()
	now := time.Now()

	for i := 0; i < 3; i++ {
		require.NoError(t, s.InsertReceivedEvent(ctx, &ReceivedEvent{
			ID: id.NewSSFEventID(), StreamID: streamID,
			JTI: "recent-" + string(rune('a'+i)), EventType: "e",
			Outcome: OutcomeApplied, ActionTaken: "revoked_all", ReceivedAt: now,
		}))
	}
	// Old, and one with no action taken. Neither counts.
	require.NoError(t, s.InsertReceivedEvent(ctx, &ReceivedEvent{
		ID: id.NewSSFEventID(), StreamID: streamID, JTI: "old", EventType: "e",
		Outcome: OutcomeApplied, ActionTaken: "revoked_all",
		ReceivedAt: now.Add(-2 * time.Hour),
	}))
	require.NoError(t, s.InsertReceivedEvent(ctx, &ReceivedEvent{
		ID: id.NewSSFEventID(), StreamID: streamID, JTI: "noop", EventType: "e",
		Outcome: OutcomeIgnored, ReceivedAt: now,
	}))

	count, err := s.CountActionsSince(ctx, streamID, now.Add(-time.Hour))
	require.NoError(t, err)
	assert.Equal(t, 3, count)
}

func TestMemoryStore_SignalsExpire(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	appID, userID := id.NewAppID(), id.NewUserID()
	now := time.Now()

	require.NoError(t, s.CreateSignal(ctx, &Signal{
		ID: id.NewSSFSignalID(), AppID: appID, UserID: userID,
		EventType: "live", Severity: 90,
		EventAt: now, ExpiresAt: now.Add(time.Hour), CreatedAt: now,
	}))
	require.NoError(t, s.CreateSignal(ctx, &Signal{
		ID: id.NewSSFSignalID(), AppID: appID, UserID: userID,
		EventType: "expired", Severity: 90,
		EventAt: now.Add(-2 * time.Hour), ExpiresAt: now.Add(-time.Hour),
		CreatedAt: now.Add(-2 * time.Hour),
	}))

	got, err := s.ListActiveSignals(ctx, appID, userID, now)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "live", got[0].EventType)

	// Another user's signals are not ours.
	got, err = s.ListActiveSignals(ctx, appID, id.NewUserID(), now)
	require.NoError(t, err)
	assert.Empty(t, got)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./plugins/sharedsignals/ -v`
Expected: FAIL, the package does not compile.

- [ ] **Step 3: Write the domain types and the Store interface**

Create `plugins/sharedsignals/store.go`:

```go
// Package sharedsignals implements the OpenID Shared Signals Framework and
// CAEP. It receives Security Event Tokens from an upstream identity provider
// and turns them into session revocations and durable risk signals.
package sharedsignals

import (
	"context"
	"errors"
	"time"

	"github.com/xraph/authsome/id"
)

// Store errors.
var (
	// ErrNotFound is returned when a row does not exist.
	ErrNotFound = errors.New("sharedsignals: not found")
	// ErrDuplicateJTI is returned when a (stream_id, jti) pair already
	// exists. Callers treat this as a replay and answer 202, so it must be
	// distinguishable from every other write failure.
	ErrDuplicateJTI = errors.New("sharedsignals: duplicate jti")
)

// Stream status values, matching the SSF stream status vocabulary.
const (
	StatusEnabled  = "enabled"
	StatusPaused   = "paused"
	StatusDisabled = "disabled"
)

// Enforcement modes. Observe records what would have happened without doing it.
const (
	EnforcementEnforce = "enforce"
	EnforcementObserve = "observe"
)

// Received-event outcomes.
const (
	OutcomePending    = "pending"
	OutcomeApplied    = "applied"
	OutcomeIgnored    = "ignored"
	OutcomeUnresolved = "unresolved"
	OutcomeRejected   = "rejected"
)

// Subject link sources.
const (
	SourceSSO    = "sso"
	SourceSocial = "social"
	SourceSCIM   = "scim"
	SourceManual = "manual"
)

// InboundStream is one identity provider we accept events from. Everything
// the receiver trusts about a SET comes from this row, never from the token.
type InboundStream struct {
	ID    id.SSFStreamID
	AppID id.AppID
	EnvID id.EnvironmentID
	Name  string

	// Issuer is the exact iss value a SET must carry.
	Issuer string
	// Audience is the aud value a SET must include.
	Audience string
	// JWKSURI is where this stream's verification keys live.
	JWKSURI string

	// PushPathHash is SHA-256 of the secret URL segment. The plaintext is
	// shown once at creation and never stored.
	PushPathHash string
	// PushTokenHash is SHA-256 of the bearer token the transmitter must send.
	PushTokenHash string

	AllowedEventTypes     []string
	AllowedSubjectFormats []string
	VerifiedDomains       []string
	ActionOverrides       map[string]string

	EnforcementMode   string
	Status            string
	MaxActionsPerHour int

	// PendingVerifyState is the state we sent to the transmitter's
	// verification endpoint and expect echoed back in a verification event.
	PendingVerifyState string
	LastVerifiedAt     *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}

// SubjectLink binds an upstream (issuer, subject) pair to an authsome user.
// This is what makes the iss_sub subject format resolvable.
type SubjectLink struct {
	ID         id.SSFLinkID
	AppID      id.AppID
	EnvID      id.EnvironmentID
	Issuer     string
	Subject    string
	UserID     id.UserID
	Source     string
	CreatedAt  time.Time
	LastSeenAt time.Time
}

// ReceivedEvent is both the replay guard and the audit trail for one SET.
type ReceivedEvent struct {
	ID             id.SSFEventID
	StreamID       id.SSFStreamID
	JTI            string
	EventType      string
	SubjectJSON    string
	ResolvedUserID id.UserID
	Outcome        string
	ActionTaken    string
	Error          string
	ReceivedAt     time.Time
}

// Signal is durable risk state. An event that arrives while nobody is signing
// in has to survive until the sign-in that cares about it.
type Signal struct {
	ID        id.SSFSignalID
	AppID     id.AppID
	EnvID     id.EnvironmentID
	UserID    id.UserID
	StreamID  id.SSFStreamID
	EventType string
	Severity  int
	Reason    string
	EventAt   time.Time
	ExpiresAt time.Time
	CreatedAt time.Time
}

// Store persists everything the receiver needs.
type Store interface {
	CreateInboundStream(ctx context.Context, s *InboundStream) error
	GetInboundStream(ctx context.Context, streamID id.SSFStreamID) (*InboundStream, error)
	GetInboundStreamByPushPathHash(ctx context.Context, hash string) (*InboundStream, error)
	ListInboundStreams(ctx context.Context, appID id.AppID) ([]*InboundStream, error)
	UpdateInboundStream(ctx context.Context, s *InboundStream) error
	DeleteInboundStream(ctx context.Context, streamID id.SSFStreamID) error

	UpsertSubjectLink(ctx context.Context, l *SubjectLink) error
	GetSubjectLink(ctx context.Context, appID id.AppID, envID id.EnvironmentID,
		issuer, subject string) (*SubjectLink, error)

	// InsertReceivedEvent returns ErrDuplicateJTI when (stream_id, jti)
	// already exists.
	InsertReceivedEvent(ctx context.Context, e *ReceivedEvent) error
	UpdateReceivedEvent(ctx context.Context, e *ReceivedEvent) error
	// CountActionsSince counts events on a stream that actually did
	// something since a cutoff. It backs the circuit breaker.
	CountActionsSince(ctx context.Context, streamID id.SSFStreamID, since time.Time) (int, error)

	CreateSignal(ctx context.Context, s *Signal) error
	ListActiveSignals(ctx context.Context, appID id.AppID, userID id.UserID,
		now time.Time) ([]*Signal, error)
}
```

- [ ] **Step 4: Write the memory store**

Create `plugins/sharedsignals/store_memory.go`:

```go
package sharedsignals

import (
	"context"
	"sync"
	"time"

	"github.com/xraph/authsome/id"
)

// MemoryStore is an in-memory Store, used in tests and in standalone mode
// when no database is configured.
type MemoryStore struct {
	mu      sync.RWMutex
	streams map[string]*InboundStream
	links   map[string]*SubjectLink
	events  map[string]*ReceivedEvent
	signals map[string]*Signal
}

// NewMemoryStore builds an empty in-memory store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		streams: make(map[string]*InboundStream),
		links:   make(map[string]*SubjectLink),
		events:  make(map[string]*ReceivedEvent),
		signals: make(map[string]*Signal),
	}
}

var _ Store = (*MemoryStore)(nil)

func linkKey(appID id.AppID, envID id.EnvironmentID, issuer, subject string) string {
	return appID.String() + "|" + envID.String() + "|" + issuer + "|" + subject
}

func eventKey(streamID id.SSFStreamID, jti string) string {
	return streamID.String() + "|" + jti
}

func cloneStream(s *InboundStream) *InboundStream {
	out := *s
	out.AllowedEventTypes = append([]string(nil), s.AllowedEventTypes...)
	out.AllowedSubjectFormats = append([]string(nil), s.AllowedSubjectFormats...)
	out.VerifiedDomains = append([]string(nil), s.VerifiedDomains...)
	if s.ActionOverrides != nil {
		out.ActionOverrides = make(map[string]string, len(s.ActionOverrides))
		for k, v := range s.ActionOverrides {
			out.ActionOverrides[k] = v
		}
	}
	return &out
}

func (m *MemoryStore) CreateInboundStream(_ context.Context, s *InboundStream) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	if s.CreatedAt.IsZero() {
		s.CreatedAt = now
	}
	s.UpdatedAt = now
	m.streams[s.ID.String()] = cloneStream(s)
	return nil
}

func (m *MemoryStore) GetInboundStream(_ context.Context, streamID id.SSFStreamID) (*InboundStream, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.streams[streamID.String()]
	if !ok {
		return nil, ErrNotFound
	}
	return cloneStream(s), nil
}

func (m *MemoryStore) GetInboundStreamByPushPathHash(_ context.Context, hash string) (*InboundStream, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, s := range m.streams {
		if s.PushPathHash == hash {
			return cloneStream(s), nil
		}
	}
	return nil, ErrNotFound
}

func (m *MemoryStore) ListInboundStreams(_ context.Context, appID id.AppID) ([]*InboundStream, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []*InboundStream
	for _, s := range m.streams {
		if s.AppID == appID {
			out = append(out, cloneStream(s))
		}
	}
	return out, nil
}

func (m *MemoryStore) UpdateInboundStream(_ context.Context, s *InboundStream) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.streams[s.ID.String()]; !ok {
		return ErrNotFound
	}
	s.UpdatedAt = time.Now()
	m.streams[s.ID.String()] = cloneStream(s)
	return nil
}

func (m *MemoryStore) DeleteInboundStream(_ context.Context, streamID id.SSFStreamID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.streams[streamID.String()]; !ok {
		return ErrNotFound
	}
	delete(m.streams, streamID.String())
	return nil
}

func (m *MemoryStore) UpsertSubjectLink(_ context.Context, l *SubjectLink) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	if l.CreatedAt.IsZero() {
		l.CreatedAt = now
	}
	l.LastSeenAt = now
	out := *l
	m.links[linkKey(l.AppID, l.EnvID, l.Issuer, l.Subject)] = &out
	return nil
}

func (m *MemoryStore) GetSubjectLink(_ context.Context, appID id.AppID,
	envID id.EnvironmentID, issuer, subject string) (*SubjectLink, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	l, ok := m.links[linkKey(appID, envID, issuer, subject)]
	if !ok {
		return nil, ErrNotFound
	}
	out := *l
	return &out, nil
}

func (m *MemoryStore) InsertReceivedEvent(_ context.Context, e *ReceivedEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	k := eventKey(e.StreamID, e.JTI)
	if _, ok := m.events[k]; ok {
		return ErrDuplicateJTI
	}
	if e.ReceivedAt.IsZero() {
		e.ReceivedAt = time.Now()
	}
	out := *e
	m.events[k] = &out
	return nil
}

func (m *MemoryStore) UpdateReceivedEvent(_ context.Context, e *ReceivedEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	k := eventKey(e.StreamID, e.JTI)
	if _, ok := m.events[k]; !ok {
		return ErrNotFound
	}
	out := *e
	m.events[k] = &out
	return nil
}

func (m *MemoryStore) CountActionsSince(_ context.Context, streamID id.SSFStreamID,
	since time.Time) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	count := 0
	for _, e := range m.events {
		if e.StreamID == streamID && e.ActionTaken != "" && e.ReceivedAt.After(since) {
			count++
		}
	}
	return count, nil
}

func (m *MemoryStore) CreateSignal(_ context.Context, s *Signal) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s.CreatedAt.IsZero() {
		s.CreatedAt = time.Now()
	}
	out := *s
	m.signals[s.ID.String()] = &out
	return nil
}

func (m *MemoryStore) ListActiveSignals(_ context.Context, appID id.AppID,
	userID id.UserID, now time.Time) ([]*Signal, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []*Signal
	for _, s := range m.signals {
		if s.AppID == appID && s.UserID == userID && s.ExpiresAt.After(now) {
			c := *s
			out = append(out, &c)
		}
	}
	return out, nil
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./plugins/sharedsignals/ -v`
Expected: PASS, five tests.

- [ ] **Step 6: Run with the race detector**

Run: `go test ./plugins/sharedsignals/ -race`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add plugins/sharedsignals/store.go plugins/sharedsignals/store_memory.go plugins/sharedsignals/store_memory_test.go
git commit -m "feat(sharedsignals): store domain types and in-memory backend"
```
