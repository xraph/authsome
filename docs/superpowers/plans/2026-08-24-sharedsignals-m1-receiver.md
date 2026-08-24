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
- Migration versions here start at `20260824000001`. A sibling plan moved off that number to avoid a collision, but that plan appends to the shared `authsome` group where versions must be unique. This one owns its own group, and version numbers are scoped per group: `plugins/social` and `plugins/sso` both already use `20240201000001` in theirs. Do not renumber.
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
2. Two core changes are needed that the spec did not name: four ID prefixes in `id/id.go`, following the convention every other plugin uses, and a `RateLimitConfig.SSFPushLimit` field so the push route can opt into the existing `authsome.PluginRateLimit` helper.

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
| `config.go` | `RateLimitConfig.SSFPushLimit`, defaulting to 60 |

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

---

### Task 7: Grove models and migrations

**Files:**
- Create: `plugins/sharedsignals/store_models.go`
- Create: `plugins/sharedsignals/migrations.go`
- Test: `plugins/sharedsignals/store_models_test.go`

**Interfaces:**
- Consumes: the domain types from Task 6.
- Produces: `inboundStreamModel`, `subjectLinkModel`, `receivedEventModel`, `signalModel`; mappers `fromInboundStream`/`toInboundStream`, `fromSubjectLink`/`toSubjectLink`, `fromReceivedEvent`/`toReceivedEvent`, `fromSignal`/`toSignal`; `PostgresMigrations`, `SqliteMigrations`, `MongoMigrations`.

String slices and the override map are stored as JSON text so the same model serves postgres and sqlite without a dialect-specific array type.

- [ ] **Step 1: Write the failing test**

Create `plugins/sharedsignals/store_models_test.go`:

```go
package sharedsignals

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
)

func TestInboundStreamModel_RoundTrip(t *testing.T) {
	verified := time.Now().UTC().Truncate(time.Second)
	in := &InboundStream{
		ID:                    id.NewSSFStreamID(),
		AppID:                 id.NewAppID(),
		EnvID:                 id.NewEnvironmentID(),
		Name:                  "okta",
		Issuer:                "https://org.okta.com",
		Audience:              "https://authsome.example/ssf",
		JWKSURI:               "https://org.okta.com/keys",
		PushPathHash:          "hp",
		PushTokenHash:         "ht",
		AllowedEventTypes:     []string{"a", "b"},
		AllowedSubjectFormats: []string{"iss_sub", "email"},
		VerifiedDomains:       []string{"corp.com"},
		ActionOverrides:       map[string]string{"a": "signal"},
		EnforcementMode:       EnforcementObserve,
		Status:                StatusEnabled,
		MaxActionsPerHour:     50,
		PendingVerifyState:    "st",
		LastVerifiedAt:        &verified,
		CreatedAt:             verified,
		UpdatedAt:             verified,
	}

	got, err := toInboundStream(fromInboundStream(in))
	require.NoError(t, err)
	assert.Equal(t, in.ID, got.ID)
	assert.Equal(t, in.AllowedEventTypes, got.AllowedEventTypes)
	assert.Equal(t, in.AllowedSubjectFormats, got.AllowedSubjectFormats)
	assert.Equal(t, in.VerifiedDomains, got.VerifiedDomains)
	assert.Equal(t, in.ActionOverrides, got.ActionOverrides)
	require.NotNil(t, got.LastVerifiedAt)
	assert.True(t, in.LastVerifiedAt.Equal(*got.LastVerifiedAt))
}

// A stream created without optional collections must not blow up on decode.
func TestInboundStreamModel_EmptyCollections(t *testing.T) {
	in := &InboundStream{
		ID: id.NewSSFStreamID(), AppID: id.NewAppID(), EnvID: id.NewEnvironmentID(),
		Status: StatusEnabled, EnforcementMode: EnforcementEnforce,
	}
	got, err := toInboundStream(fromInboundStream(in))
	require.NoError(t, err)
	assert.Empty(t, got.AllowedEventTypes)
	assert.Empty(t, got.ActionOverrides)
	assert.Nil(t, got.LastVerifiedAt)
}

func TestSubjectLinkModel_RoundTrip(t *testing.T) {
	in := &SubjectLink{
		ID: id.NewSSFLinkID(), AppID: id.NewAppID(), EnvID: id.NewEnvironmentID(),
		Issuer: "https://i", Subject: "u1", UserID: id.NewUserID(), Source: SourceSSO,
		CreatedAt: time.Now().UTC().Truncate(time.Second),
		LastSeenAt: time.Now().UTC().Truncate(time.Second),
	}
	got, err := toSubjectLink(fromSubjectLink(in))
	require.NoError(t, err)
	assert.Equal(t, in.UserID, got.UserID)
	assert.Equal(t, in.Subject, got.Subject)
}

func TestReceivedEventModel_RoundTrip(t *testing.T) {
	in := &ReceivedEvent{
		ID: id.NewSSFEventID(), StreamID: id.NewSSFStreamID(), JTI: "j1",
		EventType: "e", SubjectJSON: `{"format":"opaque","id":"u"}`,
		ResolvedUserID: id.NewUserID(), Outcome: OutcomeApplied,
		ActionTaken: "revoked_all", ReceivedAt: time.Now().UTC().Truncate(time.Second),
	}
	got, err := toReceivedEvent(fromReceivedEvent(in))
	require.NoError(t, err)
	assert.Equal(t, in.JTI, got.JTI)
	assert.Equal(t, in.ResolvedUserID, got.ResolvedUserID)
}

// An unresolved event has no user, and the zero ID must survive the trip
// rather than failing to parse.
func TestReceivedEventModel_NoResolvedUser(t *testing.T) {
	in := &ReceivedEvent{
		ID: id.NewSSFEventID(), StreamID: id.NewSSFStreamID(), JTI: "j1",
		EventType: "e", Outcome: OutcomeUnresolved,
		ReceivedAt: time.Now().UTC().Truncate(time.Second),
	}
	got, err := toReceivedEvent(fromReceivedEvent(in))
	require.NoError(t, err)
	assert.True(t, got.ResolvedUserID.IsNil())
}

func TestSignalModel_RoundTrip(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	in := &Signal{
		ID: id.NewSSFSignalID(), AppID: id.NewAppID(), EnvID: id.NewEnvironmentID(),
		UserID: id.NewUserID(), StreamID: id.NewSSFStreamID(),
		EventType: "e", Severity: 90, Reason: "why",
		EventAt: now, ExpiresAt: now.Add(time.Hour), CreatedAt: now,
	}
	got, err := toSignal(fromSignal(in))
	require.NoError(t, err)
	assert.Equal(t, 90, got.Severity)
	assert.Equal(t, in.UserID, got.UserID)
}

func TestMigrationGroups_Named(t *testing.T) {
	assert.Equal(t, "authsome-sharedsignals", PostgresMigrations.Name())
	assert.Equal(t, "authsome-sharedsignals", SqliteMigrations.Name())
	assert.Equal(t, "authsome-sharedsignals", MongoMigrations.Name())
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./plugins/sharedsignals/ -run 'Model|Migration' -v`
Expected: FAIL, the mappers and migration groups are undefined.

- [ ] **Step 3: Write the models and mappers**

Create `plugins/sharedsignals/store_models.go`:

```go
package sharedsignals

import (
	"encoding/json"
	"time"

	"github.com/xraph/grove"

	"github.com/xraph/authsome/id"
)

// ──────────────────────────────────────────────────
// Grove models
// ──────────────────────────────────────────────────

type inboundStreamModel struct {
	grove.BaseModel `grove:"table:authsome_ssf_inbound_streams,alias:sis"`

	ID    string `grove:"id,pk"`
	AppID string `grove:"app_id,notnull"`
	EnvID string `grove:"env_id,notnull"`
	Name  string `grove:"name,notnull"`

	Issuer   string `grove:"issuer,notnull"`
	Audience string `grove:"audience,notnull"`
	JWKSURI  string `grove:"jwks_uri,notnull"`

	PushPathHash  string `grove:"push_path_hash,notnull"`
	PushTokenHash string `grove:"push_token_hash,notnull"`

	AllowedEventTypes     string `grove:"allowed_event_types,notnull"`
	AllowedSubjectFormats string `grove:"allowed_subject_formats,notnull"`
	VerifiedDomains       string `grove:"verified_domains,notnull"`
	ActionOverrides       string `grove:"action_overrides,notnull"`

	EnforcementMode   string `grove:"enforcement_mode,notnull"`
	Status            string `grove:"status,notnull"`
	MaxActionsPerHour int    `grove:"max_actions_per_hour,notnull"`

	PendingVerifyState string     `grove:"pending_verify_state,notnull"`
	LastVerifiedAt     *time.Time `grove:"last_verified_at"`

	CreatedAt time.Time `grove:"created_at,notnull"`
	UpdatedAt time.Time `grove:"updated_at,notnull"`
}

type subjectLinkModel struct {
	grove.BaseModel `grove:"table:authsome_ssf_subject_links,alias:ssl"`

	ID         string    `grove:"id,pk"`
	AppID      string    `grove:"app_id,notnull"`
	EnvID      string    `grove:"env_id,notnull"`
	Issuer     string    `grove:"issuer,notnull"`
	Subject    string    `grove:"subject,notnull"`
	UserID     string    `grove:"user_id,notnull"`
	Source     string    `grove:"source,notnull"`
	CreatedAt  time.Time `grove:"created_at,notnull"`
	LastSeenAt time.Time `grove:"last_seen_at,notnull"`
}

type receivedEventModel struct {
	grove.BaseModel `grove:"table:authsome_ssf_received_events,alias:sre"`

	ID             string    `grove:"id,pk"`
	StreamID       string    `grove:"stream_id,notnull"`
	JTI            string    `grove:"jti,notnull"`
	EventType      string    `grove:"event_type,notnull"`
	SubjectJSON    string    `grove:"subject_json,notnull"`
	ResolvedUserID string    `grove:"resolved_user_id,notnull"`
	Outcome        string    `grove:"outcome,notnull"`
	ActionTaken    string    `grove:"action_taken,notnull"`
	Error          string    `grove:"error,notnull"`
	ReceivedAt     time.Time `grove:"received_at,notnull"`
}

type signalModel struct {
	grove.BaseModel `grove:"table:authsome_ssf_signals,alias:ssg"`

	ID        string    `grove:"id,pk"`
	AppID     string    `grove:"app_id,notnull"`
	EnvID     string    `grove:"env_id,notnull"`
	UserID    string    `grove:"user_id,notnull"`
	StreamID  string    `grove:"stream_id,notnull"`
	EventType string    `grove:"event_type,notnull"`
	Severity  int       `grove:"severity,notnull"`
	Reason    string    `grove:"reason,notnull"`
	EventAt   time.Time `grove:"event_at,notnull"`
	ExpiresAt time.Time `grove:"expires_at,notnull"`
	CreatedAt time.Time `grove:"created_at,notnull"`
}

// ──────────────────────────────────────────────────
// Mappers
// ──────────────────────────────────────────────────

// encodeJSON marshals a value to a JSON string, falling back to an empty
// container so a NOT NULL column always has something valid in it.
func encodeJSON(v any, empty string) string {
	if v == nil {
		return empty
	}
	b, err := json.Marshal(v)
	if err != nil {
		return empty
	}
	return string(b)
}

func decodeStrings(s string) ([]string, error) {
	if s == "" || s == "null" {
		return nil, nil
	}
	var out []string
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return nil, err
	}
	return out, nil
}

func decodeStringMap(s string) (map[string]string, error) {
	if s == "" || s == "null" {
		return nil, nil
	}
	var out map[string]string
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return nil, err
	}
	return out, nil
}

// parseOptionalID parses an ID that may legitimately be absent, returning the
// zero ID rather than an error for an empty column.
func parseOptionalID(s string) id.ID {
	if s == "" {
		return id.Nil
	}
	parsed, err := id.Parse(s)
	if err != nil {
		return id.Nil
	}
	return parsed
}

func fromInboundStream(s *InboundStream) *inboundStreamModel {
	return &inboundStreamModel{
		ID:                    s.ID.String(),
		AppID:                 s.AppID.String(),
		EnvID:                 s.EnvID.String(),
		Name:                  s.Name,
		Issuer:                s.Issuer,
		Audience:              s.Audience,
		JWKSURI:               s.JWKSURI,
		PushPathHash:          s.PushPathHash,
		PushTokenHash:         s.PushTokenHash,
		AllowedEventTypes:     encodeJSON(s.AllowedEventTypes, "[]"),
		AllowedSubjectFormats: encodeJSON(s.AllowedSubjectFormats, "[]"),
		VerifiedDomains:       encodeJSON(s.VerifiedDomains, "[]"),
		ActionOverrides:       encodeJSON(s.ActionOverrides, "{}"),
		EnforcementMode:       s.EnforcementMode,
		Status:                s.Status,
		MaxActionsPerHour:     s.MaxActionsPerHour,
		PendingVerifyState:    s.PendingVerifyState,
		LastVerifiedAt:        s.LastVerifiedAt,
		CreatedAt:             s.CreatedAt,
		UpdatedAt:             s.UpdatedAt,
	}
}

func toInboundStream(m *inboundStreamModel) (*InboundStream, error) {
	streamID, err := id.Parse(m.ID)
	if err != nil {
		return nil, err
	}
	appID, err := id.Parse(m.AppID)
	if err != nil {
		return nil, err
	}
	events, err := decodeStrings(m.AllowedEventTypes)
	if err != nil {
		return nil, err
	}
	formats, err := decodeStrings(m.AllowedSubjectFormats)
	if err != nil {
		return nil, err
	}
	domains, err := decodeStrings(m.VerifiedDomains)
	if err != nil {
		return nil, err
	}
	overrides, err := decodeStringMap(m.ActionOverrides)
	if err != nil {
		return nil, err
	}
	return &InboundStream{
		ID:                    streamID,
		AppID:                 appID,
		EnvID:                 parseOptionalID(m.EnvID),
		Name:                  m.Name,
		Issuer:                m.Issuer,
		Audience:              m.Audience,
		JWKSURI:               m.JWKSURI,
		PushPathHash:          m.PushPathHash,
		PushTokenHash:         m.PushTokenHash,
		AllowedEventTypes:     events,
		AllowedSubjectFormats: formats,
		VerifiedDomains:       domains,
		ActionOverrides:       overrides,
		EnforcementMode:       m.EnforcementMode,
		Status:                m.Status,
		MaxActionsPerHour:     m.MaxActionsPerHour,
		PendingVerifyState:    m.PendingVerifyState,
		LastVerifiedAt:        m.LastVerifiedAt,
		CreatedAt:             m.CreatedAt,
		UpdatedAt:             m.UpdatedAt,
	}, nil
}

func fromSubjectLink(l *SubjectLink) *subjectLinkModel {
	return &subjectLinkModel{
		ID:         l.ID.String(),
		AppID:      l.AppID.String(),
		EnvID:      l.EnvID.String(),
		Issuer:     l.Issuer,
		Subject:    l.Subject,
		UserID:     l.UserID.String(),
		Source:     l.Source,
		CreatedAt:  l.CreatedAt,
		LastSeenAt: l.LastSeenAt,
	}
}

func toSubjectLink(m *subjectLinkModel) (*SubjectLink, error) {
	linkID, err := id.Parse(m.ID)
	if err != nil {
		return nil, err
	}
	userID, err := id.Parse(m.UserID)
	if err != nil {
		return nil, err
	}
	return &SubjectLink{
		ID:         linkID,
		AppID:      parseOptionalID(m.AppID),
		EnvID:      parseOptionalID(m.EnvID),
		Issuer:     m.Issuer,
		Subject:    m.Subject,
		UserID:     userID,
		Source:     m.Source,
		CreatedAt:  m.CreatedAt,
		LastSeenAt: m.LastSeenAt,
	}, nil
}

func fromReceivedEvent(e *ReceivedEvent) *receivedEventModel {
	resolved := ""
	if !e.ResolvedUserID.IsNil() {
		resolved = e.ResolvedUserID.String()
	}
	return &receivedEventModel{
		ID:             e.ID.String(),
		StreamID:       e.StreamID.String(),
		JTI:            e.JTI,
		EventType:      e.EventType,
		SubjectJSON:    e.SubjectJSON,
		ResolvedUserID: resolved,
		Outcome:        e.Outcome,
		ActionTaken:    e.ActionTaken,
		Error:          e.Error,
		ReceivedAt:     e.ReceivedAt,
	}
}

func toReceivedEvent(m *receivedEventModel) (*ReceivedEvent, error) {
	eventID, err := id.Parse(m.ID)
	if err != nil {
		return nil, err
	}
	return &ReceivedEvent{
		ID:             eventID,
		StreamID:       parseOptionalID(m.StreamID),
		JTI:            m.JTI,
		EventType:      m.EventType,
		SubjectJSON:    m.SubjectJSON,
		ResolvedUserID: parseOptionalID(m.ResolvedUserID),
		Outcome:        m.Outcome,
		ActionTaken:    m.ActionTaken,
		Error:          m.Error,
		ReceivedAt:     m.ReceivedAt,
	}, nil
}

func fromSignal(s *Signal) *signalModel {
	return &signalModel{
		ID:        s.ID.String(),
		AppID:     s.AppID.String(),
		EnvID:     s.EnvID.String(),
		UserID:    s.UserID.String(),
		StreamID:  s.StreamID.String(),
		EventType: s.EventType,
		Severity:  s.Severity,
		Reason:    s.Reason,
		EventAt:   s.EventAt,
		ExpiresAt: s.ExpiresAt,
		CreatedAt: s.CreatedAt,
	}
}

func toSignal(m *signalModel) (*Signal, error) {
	signalID, err := id.Parse(m.ID)
	if err != nil {
		return nil, err
	}
	return &Signal{
		ID:        signalID,
		AppID:     parseOptionalID(m.AppID),
		EnvID:     parseOptionalID(m.EnvID),
		UserID:    parseOptionalID(m.UserID),
		StreamID:  parseOptionalID(m.StreamID),
		EventType: m.EventType,
		Severity:  m.Severity,
		Reason:    m.Reason,
		EventAt:   m.EventAt,
		ExpiresAt: m.ExpiresAt,
		CreatedAt: m.CreatedAt,
	}, nil
}
```

- [ ] **Step 4: Write the migrations**

Create `plugins/sharedsignals/migrations.go`:

```go
package sharedsignals

import (
	"context"

	"github.com/xraph/grove/migrate"
)

// Migration groups, one per driver. All depend on the core authsome group
// because the stream tables reference authsome_apps.
var (
	PostgresMigrations = migrate.NewGroup("authsome-sharedsignals", migrate.DependsOn("authsome"))
	SqliteMigrations   = migrate.NewGroup("authsome-sharedsignals", migrate.DependsOn("authsome"))
	MongoMigrations    = migrate.NewGroup("authsome-sharedsignals", migrate.DependsOn("authsome"))
)

// Mongo collection names.
const (
	colInboundStreams = "authsome_ssf_inbound_streams"
	colSubjectLinks   = "authsome_ssf_subject_links"
	colReceivedEvents = "authsome_ssf_received_events"
	colSignals        = "authsome_ssf_signals"
)

const pgSchema = `
CREATE TABLE IF NOT EXISTS authsome_ssf_inbound_streams (
    id                      TEXT PRIMARY KEY,
    app_id                  TEXT NOT NULL REFERENCES authsome_apps(id),
    env_id                  TEXT NOT NULL DEFAULT '',
    name                    TEXT NOT NULL DEFAULT '',
    issuer                  TEXT NOT NULL,
    audience                TEXT NOT NULL,
    jwks_uri                TEXT NOT NULL,
    push_path_hash          TEXT NOT NULL,
    push_token_hash         TEXT NOT NULL,
    allowed_event_types     TEXT NOT NULL DEFAULT '[]',
    allowed_subject_formats TEXT NOT NULL DEFAULT '[]',
    verified_domains        TEXT NOT NULL DEFAULT '[]',
    action_overrides        TEXT NOT NULL DEFAULT '{}',
    enforcement_mode        TEXT NOT NULL DEFAULT 'enforce',
    status                  TEXT NOT NULL DEFAULT 'enabled',
    max_actions_per_hour    INTEGER NOT NULL DEFAULT 100,
    pending_verify_state    TEXT NOT NULL DEFAULT '',
    last_verified_at        TIMESTAMPTZ,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- The push path is how an inbound request finds its stream, so the lookup
-- must be unique and indexed.
CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_ssf_streams_push_path
    ON authsome_ssf_inbound_streams (push_path_hash);
CREATE INDEX IF NOT EXISTS idx_authsome_ssf_streams_app
    ON authsome_ssf_inbound_streams (app_id, created_at DESC);

CREATE TABLE IF NOT EXISTS authsome_ssf_subject_links (
    id           TEXT PRIMARY KEY,
    app_id       TEXT NOT NULL,
    env_id       TEXT NOT NULL DEFAULT '',
    issuer       TEXT NOT NULL,
    subject      TEXT NOT NULL,
    user_id      TEXT NOT NULL,
    source       TEXT NOT NULL DEFAULT 'sso',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_ssf_links_tuple
    ON authsome_ssf_subject_links (app_id, env_id, issuer, subject);
CREATE INDEX IF NOT EXISTS idx_authsome_ssf_links_user
    ON authsome_ssf_subject_links (user_id);

CREATE TABLE IF NOT EXISTS authsome_ssf_received_events (
    id               TEXT PRIMARY KEY,
    stream_id        TEXT NOT NULL,
    jti              TEXT NOT NULL,
    event_type       TEXT NOT NULL,
    subject_json     TEXT NOT NULL DEFAULT '',
    resolved_user_id TEXT NOT NULL DEFAULT '',
    outcome          TEXT NOT NULL DEFAULT 'pending',
    action_taken     TEXT NOT NULL DEFAULT '',
    error            TEXT NOT NULL DEFAULT '',
    received_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- This unique constraint IS the replay guard. Without it a replayed SET
-- revokes sessions twice.
CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_ssf_events_jti
    ON authsome_ssf_received_events (stream_id, jti);
CREATE INDEX IF NOT EXISTS idx_authsome_ssf_events_stream_time
    ON authsome_ssf_received_events (stream_id, received_at DESC);

CREATE TABLE IF NOT EXISTS authsome_ssf_signals (
    id         TEXT PRIMARY KEY,
    app_id     TEXT NOT NULL,
    env_id     TEXT NOT NULL DEFAULT '',
    user_id    TEXT NOT NULL,
    stream_id  TEXT NOT NULL DEFAULT '',
    event_type TEXT NOT NULL,
    severity   INTEGER NOT NULL DEFAULT 0,
    reason     TEXT NOT NULL DEFAULT '',
    event_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_authsome_ssf_signals_lookup
    ON authsome_ssf_signals (app_id, user_id, expires_at DESC);
`

// sqliteSchema is the same shape with SQLite's type names. TIMESTAMPTZ and
// NOW() do not exist there.
const sqliteSchema = `
CREATE TABLE IF NOT EXISTS authsome_ssf_inbound_streams (
    id                      TEXT PRIMARY KEY,
    app_id                  TEXT NOT NULL,
    env_id                  TEXT NOT NULL DEFAULT '',
    name                    TEXT NOT NULL DEFAULT '',
    issuer                  TEXT NOT NULL,
    audience                TEXT NOT NULL,
    jwks_uri                TEXT NOT NULL,
    push_path_hash          TEXT NOT NULL,
    push_token_hash         TEXT NOT NULL,
    allowed_event_types     TEXT NOT NULL DEFAULT '[]',
    allowed_subject_formats TEXT NOT NULL DEFAULT '[]',
    verified_domains        TEXT NOT NULL DEFAULT '[]',
    action_overrides        TEXT NOT NULL DEFAULT '{}',
    enforcement_mode        TEXT NOT NULL DEFAULT 'enforce',
    status                  TEXT NOT NULL DEFAULT 'enabled',
    max_actions_per_hour    INTEGER NOT NULL DEFAULT 100,
    pending_verify_state    TEXT NOT NULL DEFAULT '',
    last_verified_at        DATETIME,
    created_at              DATETIME NOT NULL,
    updated_at              DATETIME NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_ssf_streams_push_path
    ON authsome_ssf_inbound_streams (push_path_hash);
CREATE INDEX IF NOT EXISTS idx_authsome_ssf_streams_app
    ON authsome_ssf_inbound_streams (app_id, created_at DESC);

CREATE TABLE IF NOT EXISTS authsome_ssf_subject_links (
    id           TEXT PRIMARY KEY,
    app_id       TEXT NOT NULL,
    env_id       TEXT NOT NULL DEFAULT '',
    issuer       TEXT NOT NULL,
    subject      TEXT NOT NULL,
    user_id      TEXT NOT NULL,
    source       TEXT NOT NULL DEFAULT 'sso',
    created_at   DATETIME NOT NULL,
    last_seen_at DATETIME NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_ssf_links_tuple
    ON authsome_ssf_subject_links (app_id, env_id, issuer, subject);
CREATE INDEX IF NOT EXISTS idx_authsome_ssf_links_user
    ON authsome_ssf_subject_links (user_id);

CREATE TABLE IF NOT EXISTS authsome_ssf_received_events (
    id               TEXT PRIMARY KEY,
    stream_id        TEXT NOT NULL,
    jti              TEXT NOT NULL,
    event_type       TEXT NOT NULL,
    subject_json     TEXT NOT NULL DEFAULT '',
    resolved_user_id TEXT NOT NULL DEFAULT '',
    outcome          TEXT NOT NULL DEFAULT 'pending',
    action_taken     TEXT NOT NULL DEFAULT '',
    error            TEXT NOT NULL DEFAULT '',
    received_at      DATETIME NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_ssf_events_jti
    ON authsome_ssf_received_events (stream_id, jti);
CREATE INDEX IF NOT EXISTS idx_authsome_ssf_events_stream_time
    ON authsome_ssf_received_events (stream_id, received_at DESC);

CREATE TABLE IF NOT EXISTS authsome_ssf_signals (
    id         TEXT PRIMARY KEY,
    app_id     TEXT NOT NULL,
    env_id     TEXT NOT NULL DEFAULT '',
    user_id    TEXT NOT NULL,
    stream_id  TEXT NOT NULL DEFAULT '',
    event_type TEXT NOT NULL,
    severity   INTEGER NOT NULL DEFAULT 0,
    reason     TEXT NOT NULL DEFAULT '',
    event_at   DATETIME NOT NULL,
    expires_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_authsome_ssf_signals_lookup
    ON authsome_ssf_signals (app_id, user_id, expires_at DESC);
`

const dropSchema = `
DROP TABLE IF EXISTS authsome_ssf_signals;
DROP TABLE IF EXISTS authsome_ssf_received_events;
DROP TABLE IF EXISTS authsome_ssf_subject_links;
DROP TABLE IF EXISTS authsome_ssf_inbound_streams;
`

func init() {
	PostgresMigrations.MustRegister(&migrate.Migration{
		Name:    "create_sharedsignals_tables",
		Version: "20260824000001",
		Up: func(ctx context.Context, exec migrate.Executor) error {
			_, err := exec.Exec(ctx, pgSchema)
			return err
		},
		Down: func(ctx context.Context, exec migrate.Executor) error {
			_, err := exec.Exec(ctx, dropSchema)
			return err
		},
	})

	SqliteMigrations.MustRegister(&migrate.Migration{
		Name:    "create_sharedsignals_tables",
		Version: "20260824000001",
		Up: func(ctx context.Context, exec migrate.Executor) error {
			_, err := exec.Exec(ctx, sqliteSchema)
			return err
		},
		Down: func(ctx context.Context, exec migrate.Executor) error {
			_, err := exec.Exec(ctx, dropSchema)
			return err
		},
	})

}
```

The `MongoMigrations` group is declared here but registered in Task 8, because
`mongomigrate.Executor.CreateCollection` takes a model value rather than a
collection name, and those doc types do not exist yet.

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./plugins/sharedsignals/ -run 'Model|Migration' -v`
Expected: PASS, seven tests.

- [ ] **Step 6: Commit**

```bash
git add plugins/sharedsignals/store_models.go plugins/sharedsignals/migrations.go plugins/sharedsignals/store_models_test.go
git commit -m "feat(sharedsignals): grove models and migrations for pg, sqlite and mongo"
```

---

### Task 8: Postgres and SQLite stores, plus the conformance suite

**Files:**
- Create: `plugins/sharedsignals/store_postgres.go`
- Create: `plugins/sharedsignals/store_sqlite.go`
- Create: `plugins/sharedsignals/store_conformance_test.go`
- Test: `plugins/sharedsignals/store_conformance_test.go`

**Interfaces:**
- Consumes: the `Store` interface and domain types from Task 6, the models and mappers from Task 7.
- Produces: `NewPostgresStore(db *grove.DB) *PostgresStore`, `NewSqliteStore(db *grove.DB) *SqliteStore`, and `runStoreConformance(t *testing.T, newStore func(*testing.T) Store)`.

The two SQL stores are near-identical because grove's query builders return driver-specific concrete types, so a shared implementation would need matching method sets on every query type. `plugins/sso` duplicates them for the same reason; follow that rather than inventing an abstraction.

- [ ] **Step 1: Write the failing conformance suite**

Create `plugins/sharedsignals/store_conformance_test.go`:

```go
package sharedsignals

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/grove"
	"github.com/xraph/grove/drivers/sqlitedriver"
	_ "github.com/xraph/grove/drivers/sqlitedriver/sqlitemigrate"

	"github.com/xraph/authsome/id"
	sqlitestore "github.com/xraph/authsome/store/sqlite"
)

// The SQLite store shares the models and mappers with Postgres, so running
// the suite against embedded SQLite exercises the same column mapping and
// the same JSON round trip without needing Docker.

func newMemoryConformanceStore(_ *testing.T) Store { return NewMemoryStore() }

func newSQLiteConformanceStore(t *testing.T) Store {
	t.Helper()
	ctx := context.Background()
	dsn := "file:" + filepath.Join(t.TempDir(), "ssf-conformance.db") + "?cache=shared"
	sdb := sqlitedriver.New()
	require.NoError(t, sdb.Open(ctx, dsn))
	db, err := grove.Open(sdb)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	// The core migrations satisfy the group's DependsOn("authsome").
	require.NoError(t, sqlitestore.New(db).Migrate(ctx, SqliteMigrations))
	return NewSqliteStore(db)
}

func TestStoreConformance_Memory(t *testing.T) { runStoreConformance(t, newMemoryConformanceStore) }
func TestStoreConformance_SQLite(t *testing.T) { runStoreConformance(t, newSQLiteConformanceStore) }

func runStoreConformance(t *testing.T, newStore func(*testing.T) Store) {
	t.Run("inbound stream round trip", func(t *testing.T) {
		ctx := context.Background()
		s := newStore(t)
		appID := id.NewAppID()
		now := time.Now().UTC().Truncate(time.Second)

		in := &InboundStream{
			ID: id.NewSSFStreamID(), AppID: appID, EnvID: id.NewEnvironmentID(),
			Name: "okta", Issuer: "https://org.okta.com",
			Audience: "https://authsome.example/ssf",
			JWKSURI:  "https://org.okta.com/keys",
			PushPathHash: "hash-a", PushTokenHash: "tok-a",
			AllowedEventTypes:     []string{"a", "b"},
			AllowedSubjectFormats: []string{"iss_sub"},
			VerifiedDomains:       []string{"corp.com"},
			ActionOverrides:       map[string]string{"a": "signal"},
			EnforcementMode:       EnforcementEnforce,
			Status:                StatusEnabled,
			MaxActionsPerHour:     100,
			CreatedAt:             now, UpdatedAt: now,
		}
		require.NoError(t, s.CreateInboundStream(ctx, in))

		got, err := s.GetInboundStream(ctx, in.ID)
		require.NoError(t, err)
		assert.Equal(t, "okta", got.Name)
		assert.Equal(t, []string{"a", "b"}, got.AllowedEventTypes)
		assert.Equal(t, map[string]string{"a": "signal"}, got.ActionOverrides)

		byHash, err := s.GetInboundStreamByPushPathHash(ctx, "hash-a")
		require.NoError(t, err)
		assert.Equal(t, in.ID, byHash.ID)

		_, err = s.GetInboundStreamByPushPathHash(ctx, "does-not-exist")
		require.ErrorIs(t, err, ErrNotFound)

		got.Status = StatusPaused
		got.MaxActionsPerHour = 5
		require.NoError(t, s.UpdateInboundStream(ctx, got))
		after, err := s.GetInboundStream(ctx, in.ID)
		require.NoError(t, err)
		assert.Equal(t, StatusPaused, after.Status)
		assert.Equal(t, 5, after.MaxActionsPerHour)

		list, err := s.ListInboundStreams(ctx, appID)
		require.NoError(t, err)
		require.Len(t, list, 1)

		require.NoError(t, s.DeleteInboundStream(ctx, in.ID))
		_, err = s.GetInboundStream(ctx, in.ID)
		require.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("subject link upsert is scoped", func(t *testing.T) {
		ctx := context.Background()
		s := newStore(t)
		appID, envID := id.NewAppID(), id.NewEnvironmentID()
		userID := id.NewUserID()
		now := time.Now().UTC().Truncate(time.Second)

		require.NoError(t, s.UpsertSubjectLink(ctx, &SubjectLink{
			ID: id.NewSSFLinkID(), AppID: appID, EnvID: envID,
			Issuer: "https://i", Subject: "u1", UserID: userID,
			Source: SourceSSO, CreatedAt: now, LastSeenAt: now,
		}))

		got, err := s.GetSubjectLink(ctx, appID, envID, "https://i", "u1")
		require.NoError(t, err)
		assert.Equal(t, userID, got.UserID)

		// Same tuple, different user: replace, do not duplicate.
		other := id.NewUserID()
		require.NoError(t, s.UpsertSubjectLink(ctx, &SubjectLink{
			ID: id.NewSSFLinkID(), AppID: appID, EnvID: envID,
			Issuer: "https://i", Subject: "u1", UserID: other,
			Source: SourceSSO, CreatedAt: now, LastSeenAt: now,
		}))
		got, err = s.GetSubjectLink(ctx, appID, envID, "https://i", "u1")
		require.NoError(t, err)
		assert.Equal(t, other, got.UserID)

		// A link belongs to one app only.
		_, err = s.GetSubjectLink(ctx, id.NewAppID(), envID, "https://i", "u1")
		require.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("received event dedupe", func(t *testing.T) {
		ctx := context.Background()
		s := newStore(t)
		streamID := id.NewSSFStreamID()
		now := time.Now().UTC().Truncate(time.Second)

		ev := &ReceivedEvent{
			ID: id.NewSSFEventID(), StreamID: streamID, JTI: "jti-1",
			EventType: "e", Outcome: OutcomePending, ReceivedAt: now,
		}
		require.NoError(t, s.InsertReceivedEvent(ctx, ev))

		err := s.InsertReceivedEvent(ctx, &ReceivedEvent{
			ID: id.NewSSFEventID(), StreamID: streamID, JTI: "jti-1",
			EventType: "e", Outcome: OutcomePending, ReceivedAt: now,
		})
		require.ErrorIs(t, err, ErrDuplicateJTI,
			"a replayed jti must be reported as a duplicate, not a generic write error")

		require.NoError(t, s.InsertReceivedEvent(ctx, &ReceivedEvent{
			ID: id.NewSSFEventID(), StreamID: id.NewSSFStreamID(), JTI: "jti-1",
			EventType: "e", Outcome: OutcomePending, ReceivedAt: now,
		}))

		ev.Outcome = OutcomeApplied
		ev.ActionTaken = "revoked_all"
		require.NoError(t, s.UpdateReceivedEvent(ctx, ev))

		count, err := s.CountActionsSince(ctx, streamID, now.Add(-time.Hour))
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("signals expire", func(t *testing.T) {
		ctx := context.Background()
		s := newStore(t)
		appID, userID := id.NewAppID(), id.NewUserID()
		now := time.Now().UTC().Truncate(time.Second)

		require.NoError(t, s.CreateSignal(ctx, &Signal{
			ID: id.NewSSFSignalID(), AppID: appID, EnvID: id.NewEnvironmentID(),
			UserID: userID, StreamID: id.NewSSFStreamID(),
			EventType: "live", Severity: 90, Reason: "compromised",
			EventAt: now, ExpiresAt: now.Add(time.Hour), CreatedAt: now,
		}))
		require.NoError(t, s.CreateSignal(ctx, &Signal{
			ID: id.NewSSFSignalID(), AppID: appID, EnvID: id.NewEnvironmentID(),
			UserID: userID, StreamID: id.NewSSFStreamID(),
			EventType: "stale", Severity: 90, Reason: "old",
			EventAt: now.Add(-2 * time.Hour), ExpiresAt: now.Add(-time.Hour),
			CreatedAt: now.Add(-2 * time.Hour),
		}))

		got, err := s.ListActiveSignals(ctx, appID, userID, now)
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, "live", got[0].EventType)
		assert.Equal(t, 90, got[0].Severity)

		got, err = s.ListActiveSignals(ctx, appID, id.NewUserID(), now)
		require.NoError(t, err)
		assert.Empty(t, got)
	})
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./plugins/sharedsignals/ -run TestStoreConformance -v`
Expected: FAIL, `NewSqliteStore` is undefined.

- [ ] **Step 3: Write the postgres store**

Create `plugins/sharedsignals/store_postgres.go`:

```go
package sharedsignals

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/xraph/grove"
	"github.com/xraph/grove/drivers/pgdriver"

	"github.com/xraph/authsome/id"
)

// PostgresStore implements Store on the Grove ORM with PostgreSQL.
type PostgresStore struct {
	db *grove.DB
	pg *pgdriver.PgDB
}

// NewPostgresStore builds a PostgreSQL-backed store.
func NewPostgresStore(db *grove.DB) *PostgresStore {
	return &PostgresStore{db: db, pg: pgdriver.Unwrap(db)}
}

var _ Store = (*PostgresStore)(nil)

// isDuplicateKey reports whether a driver error is a unique-constraint
// violation. The dedupe path depends on telling that apart from a real
// failure, so both SQL stores route through this.
func isDuplicateKey(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate key") ||
		strings.Contains(msg, "unique constraint") ||
		strings.Contains(msg, "unique violation") ||
		strings.Contains(msg, "23505")
}

func sqlErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func (s *PostgresStore) CreateInboundStream(ctx context.Context, in *InboundStream) error {
	now := time.Now()
	if in.CreatedAt.IsZero() {
		in.CreatedAt = now
	}
	in.UpdatedAt = now
	_, err := s.pg.NewInsert(fromInboundStream(in)).Exec(ctx)
	return sqlErr(err)
}

func (s *PostgresStore) GetInboundStream(ctx context.Context, streamID id.SSFStreamID) (*InboundStream, error) {
	m := new(inboundStreamModel)
	if err := s.pg.NewSelect(m).Where("id = ?", streamID.String()).Scan(ctx); err != nil {
		return nil, sqlErr(err)
	}
	return toInboundStream(m)
}

func (s *PostgresStore) GetInboundStreamByPushPathHash(ctx context.Context, hash string) (*InboundStream, error) {
	m := new(inboundStreamModel)
	if err := s.pg.NewSelect(m).Where("push_path_hash = ?", hash).Scan(ctx); err != nil {
		return nil, sqlErr(err)
	}
	return toInboundStream(m)
}

func (s *PostgresStore) ListInboundStreams(ctx context.Context, appID id.AppID) ([]*InboundStream, error) {
	var models []*inboundStreamModel
	if err := s.pg.NewSelect(&models).
		Where("app_id = ?", appID.String()).
		Order("created_at DESC").
		Scan(ctx); err != nil {
		return nil, sqlErr(err)
	}
	out := make([]*InboundStream, 0, len(models))
	for _, m := range models {
		converted, err := toInboundStream(m)
		if err != nil {
			return nil, err
		}
		out = append(out, converted)
	}
	return out, nil
}

func (s *PostgresStore) UpdateInboundStream(ctx context.Context, in *InboundStream) error {
	in.UpdatedAt = time.Now()
	res, err := s.pg.NewUpdate(fromInboundStream(in)).
		Where("id = ?", in.ID.String()).Exec(ctx)
	if err != nil {
		return sqlErr(err)
	}
	if n, rerr := res.RowsAffected(); rerr == nil && n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) DeleteInboundStream(ctx context.Context, streamID id.SSFStreamID) error {
	res, err := s.pg.NewDelete((*inboundStreamModel)(nil)).
		Where("id = ?", streamID.String()).Exec(ctx)
	if err != nil {
		return sqlErr(err)
	}
	if n, rerr := res.RowsAffected(); rerr == nil && n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) UpsertSubjectLink(ctx context.Context, l *SubjectLink) error {
	now := time.Now()
	if l.CreatedAt.IsZero() {
		l.CreatedAt = now
	}
	l.LastSeenAt = now

	existing, err := s.GetSubjectLink(ctx, l.AppID, l.EnvID, l.Issuer, l.Subject)
	switch {
	case err == nil:
		l.ID = existing.ID
		l.CreatedAt = existing.CreatedAt
		_, uerr := s.pg.NewUpdate(fromSubjectLink(l)).
			Where("id = ?", l.ID.String()).Exec(ctx)
		return sqlErr(uerr)
	case errors.Is(err, ErrNotFound):
		_, ierr := s.pg.NewInsert(fromSubjectLink(l)).Exec(ctx)
		return sqlErr(ierr)
	default:
		return err
	}
}

func (s *PostgresStore) GetSubjectLink(ctx context.Context, appID id.AppID,
	envID id.EnvironmentID, issuer, subject string) (*SubjectLink, error) {
	m := new(subjectLinkModel)
	if err := s.pg.NewSelect(m).
		Where("app_id = ?", appID.String()).
		Where("env_id = ?", envID.String()).
		Where("issuer = ?", issuer).
		Where("subject = ?", subject).
		Scan(ctx); err != nil {
		return nil, sqlErr(err)
	}
	return toSubjectLink(m)
}

func (s *PostgresStore) InsertReceivedEvent(ctx context.Context, e *ReceivedEvent) error {
	if e.ReceivedAt.IsZero() {
		e.ReceivedAt = time.Now()
	}
	_, err := s.pg.NewInsert(fromReceivedEvent(e)).Exec(ctx)
	if isDuplicateKey(err) {
		return ErrDuplicateJTI
	}
	return sqlErr(err)
}

func (s *PostgresStore) UpdateReceivedEvent(ctx context.Context, e *ReceivedEvent) error {
	res, err := s.pg.NewUpdate(fromReceivedEvent(e)).
		Where("id = ?", e.ID.String()).Exec(ctx)
	if err != nil {
		return sqlErr(err)
	}
	if n, rerr := res.RowsAffected(); rerr == nil && n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) CountActionsSince(ctx context.Context,
	streamID id.SSFStreamID, since time.Time) (int, error) {
	count, err := s.pg.NewSelect((*receivedEventModel)(nil)).
		Where("stream_id = ?", streamID.String()).
		Where("action_taken <> ?", "").
		Where("received_at > ?", since).
		Count(ctx)
	if err != nil {
		return 0, sqlErr(err)
	}
	return count, nil
}

func (s *PostgresStore) CreateSignal(ctx context.Context, sig *Signal) error {
	if sig.CreatedAt.IsZero() {
		sig.CreatedAt = time.Now()
	}
	_, err := s.pg.NewInsert(fromSignal(sig)).Exec(ctx)
	return sqlErr(err)
}

func (s *PostgresStore) ListActiveSignals(ctx context.Context, appID id.AppID,
	userID id.UserID, now time.Time) ([]*Signal, error) {
	var models []*signalModel
	if err := s.pg.NewSelect(&models).
		Where("app_id = ?", appID.String()).
		Where("user_id = ?", userID.String()).
		Where("expires_at > ?", now).
		Order("severity DESC").
		Scan(ctx); err != nil {
		return nil, sqlErr(err)
	}
	out := make([]*Signal, 0, len(models))
	for _, m := range models {
		converted, err := toSignal(m)
		if err != nil {
			return nil, err
		}
		out = append(out, converted)
	}
	return out, nil
}
```

- [ ] **Step 4: Write the SQLite store**

Create `plugins/sharedsignals/store_sqlite.go`. It is the postgres file with
`pgdriver.PgDB` swapped for `sqlitedriver.SqliteDB`. `isDuplicateKey` and
`sqlErr` already live in `store_postgres.go` and are reused here, so do not
redeclare them.

```go
package sharedsignals

import (
	"context"
	"errors"
	"time"

	"github.com/xraph/grove"
	"github.com/xraph/grove/drivers/sqlitedriver"

	"github.com/xraph/authsome/id"
)

// SqliteStore implements Store on the Grove ORM with SQLite.
type SqliteStore struct {
	db  *grove.DB
	sdb *sqlitedriver.SqliteDB
}

// NewSqliteStore builds a SQLite-backed store.
func NewSqliteStore(db *grove.DB) *SqliteStore {
	return &SqliteStore{db: db, sdb: sqlitedriver.Unwrap(db)}
}

var _ Store = (*SqliteStore)(nil)

func (s *SqliteStore) CreateInboundStream(ctx context.Context, in *InboundStream) error {
	now := time.Now()
	if in.CreatedAt.IsZero() {
		in.CreatedAt = now
	}
	in.UpdatedAt = now
	_, err := s.sdb.NewInsert(fromInboundStream(in)).Exec(ctx)
	return sqlErr(err)
}

func (s *SqliteStore) GetInboundStream(ctx context.Context, streamID id.SSFStreamID) (*InboundStream, error) {
	m := new(inboundStreamModel)
	if err := s.sdb.NewSelect(m).Where("id = ?", streamID.String()).Scan(ctx); err != nil {
		return nil, sqlErr(err)
	}
	return toInboundStream(m)
}

func (s *SqliteStore) GetInboundStreamByPushPathHash(ctx context.Context, hash string) (*InboundStream, error) {
	m := new(inboundStreamModel)
	if err := s.sdb.NewSelect(m).Where("push_path_hash = ?", hash).Scan(ctx); err != nil {
		return nil, sqlErr(err)
	}
	return toInboundStream(m)
}

func (s *SqliteStore) ListInboundStreams(ctx context.Context, appID id.AppID) ([]*InboundStream, error) {
	var models []*inboundStreamModel
	if err := s.sdb.NewSelect(&models).
		Where("app_id = ?", appID.String()).
		Order("created_at DESC").
		Scan(ctx); err != nil {
		return nil, sqlErr(err)
	}
	out := make([]*InboundStream, 0, len(models))
	for _, m := range models {
		converted, err := toInboundStream(m)
		if err != nil {
			return nil, err
		}
		out = append(out, converted)
	}
	return out, nil
}

func (s *SqliteStore) UpdateInboundStream(ctx context.Context, in *InboundStream) error {
	in.UpdatedAt = time.Now()
	res, err := s.sdb.NewUpdate(fromInboundStream(in)).
		Where("id = ?", in.ID.String()).Exec(ctx)
	if err != nil {
		return sqlErr(err)
	}
	if n, rerr := res.RowsAffected(); rerr == nil && n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *SqliteStore) DeleteInboundStream(ctx context.Context, streamID id.SSFStreamID) error {
	res, err := s.sdb.NewDelete((*inboundStreamModel)(nil)).
		Where("id = ?", streamID.String()).Exec(ctx)
	if err != nil {
		return sqlErr(err)
	}
	if n, rerr := res.RowsAffected(); rerr == nil && n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *SqliteStore) UpsertSubjectLink(ctx context.Context, l *SubjectLink) error {
	now := time.Now()
	if l.CreatedAt.IsZero() {
		l.CreatedAt = now
	}
	l.LastSeenAt = now

	existing, err := s.GetSubjectLink(ctx, l.AppID, l.EnvID, l.Issuer, l.Subject)
	switch {
	case err == nil:
		l.ID = existing.ID
		l.CreatedAt = existing.CreatedAt
		_, uerr := s.sdb.NewUpdate(fromSubjectLink(l)).
			Where("id = ?", l.ID.String()).Exec(ctx)
		return sqlErr(uerr)
	case errors.Is(err, ErrNotFound):
		_, ierr := s.sdb.NewInsert(fromSubjectLink(l)).Exec(ctx)
		return sqlErr(ierr)
	default:
		return err
	}
}

func (s *SqliteStore) GetSubjectLink(ctx context.Context, appID id.AppID,
	envID id.EnvironmentID, issuer, subject string) (*SubjectLink, error) {
	m := new(subjectLinkModel)
	if err := s.sdb.NewSelect(m).
		Where("app_id = ?", appID.String()).
		Where("env_id = ?", envID.String()).
		Where("issuer = ?", issuer).
		Where("subject = ?", subject).
		Scan(ctx); err != nil {
		return nil, sqlErr(err)
	}
	return toSubjectLink(m)
}

func (s *SqliteStore) InsertReceivedEvent(ctx context.Context, e *ReceivedEvent) error {
	if e.ReceivedAt.IsZero() {
		e.ReceivedAt = time.Now()
	}
	_, err := s.sdb.NewInsert(fromReceivedEvent(e)).Exec(ctx)
	if isDuplicateKey(err) {
		return ErrDuplicateJTI
	}
	return sqlErr(err)
}

func (s *SqliteStore) UpdateReceivedEvent(ctx context.Context, e *ReceivedEvent) error {
	res, err := s.sdb.NewUpdate(fromReceivedEvent(e)).
		Where("id = ?", e.ID.String()).Exec(ctx)
	if err != nil {
		return sqlErr(err)
	}
	if n, rerr := res.RowsAffected(); rerr == nil && n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *SqliteStore) CountActionsSince(ctx context.Context,
	streamID id.SSFStreamID, since time.Time) (int, error) {
	count, err := s.sdb.NewSelect((*receivedEventModel)(nil)).
		Where("stream_id = ?", streamID.String()).
		Where("action_taken <> ?", "").
		Where("received_at > ?", since).
		Count(ctx)
	if err != nil {
		return 0, sqlErr(err)
	}
	return count, nil
}

func (s *SqliteStore) CreateSignal(ctx context.Context, sig *Signal) error {
	if sig.CreatedAt.IsZero() {
		sig.CreatedAt = time.Now()
	}
	_, err := s.sdb.NewInsert(fromSignal(sig)).Exec(ctx)
	return sqlErr(err)
}

func (s *SqliteStore) ListActiveSignals(ctx context.Context, appID id.AppID,
	userID id.UserID, now time.Time) ([]*Signal, error) {
	var models []*signalModel
	if err := s.sdb.NewSelect(&models).
		Where("app_id = ?", appID.String()).
		Where("user_id = ?", userID.String()).
		Where("expires_at > ?", now).
		Order("severity DESC").
		Scan(ctx); err != nil {
		return nil, sqlErr(err)
	}
	out := make([]*Signal, 0, len(models))
	for _, m := range models {
		converted, err := toSignal(m)
		if err != nil {
			return nil, err
		}
		out = append(out, converted)
	}
	return out, nil
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./plugins/sharedsignals/ -run TestStoreConformance -v`
Expected: PASS, both backends, four subtests each.

If `TestStoreConformance_SQLite/received_event_dedupe` fails with a generic
error rather than `ErrDuplicateJTI`, the SQLite driver's duplicate message is
not covered by `isDuplicateKey`. Print the raw error, then add its wording
(SQLite says `UNIQUE constraint failed`) to the match list.

- [ ] **Step 6: Commit**

```bash
git add plugins/sharedsignals/store_postgres.go plugins/sharedsignals/store_sqlite.go plugins/sharedsignals/store_conformance_test.go
git commit -m "feat(sharedsignals): postgres and sqlite stores with a conformance suite"
```

---

### Task 9: Mongo store and mongo migrations

**Files:**
- Create: `plugins/sharedsignals/store_mongo.go`
- Modify: `plugins/sharedsignals/migrations.go` (register the `MongoMigrations` group deferred from Task 7)
- Test: `plugins/sharedsignals/store_mongo_test.go`

**Interfaces:**
- Consumes: the `Store` interface and domain types from Task 6.
- Produces: `NewMongoStore(db *grove.DB) *MongoStore`, and the four bson doc types the mongo migration needs.

Mongo keeps its own doc types with `bson` tags, following `plugins/sso/store_mongo.go`. The collection constants already exist in `migrations.go` from Task 7.

- [ ] **Step 1: Write the failing test**

Create `plugins/sharedsignals/store_mongo_test.go`:

```go
package sharedsignals

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
)

// The mongo store needs a live server, so the conformance suite does not run
// it by default. What we can check without one is that the doc converters
// preserve every field, which is where mongo backends usually drift from the
// SQL ones.

func TestMongoDocs_InboundStreamRoundTrip(t *testing.T) {
	verified := time.Now().UTC().Truncate(time.Millisecond)
	in := &InboundStream{
		ID: id.NewSSFStreamID(), AppID: id.NewAppID(), EnvID: id.NewEnvironmentID(),
		Name: "okta", Issuer: "https://org.okta.com",
		Audience: "https://authsome.example/ssf", JWKSURI: "https://org.okta.com/keys",
		PushPathHash: "hp", PushTokenHash: "ht",
		AllowedEventTypes:     []string{"a"},
		AllowedSubjectFormats: []string{"iss_sub"},
		VerifiedDomains:       []string{"corp.com"},
		ActionOverrides:       map[string]string{"a": "signal"},
		EnforcementMode:       EnforcementObserve, Status: StatusPaused,
		MaxActionsPerHour: 42, PendingVerifyState: "st",
		LastVerifiedAt: &verified, CreatedAt: verified, UpdatedAt: verified,
	}

	got, err := docToInboundStream(inboundStreamToDoc(in))
	require.NoError(t, err)
	assert.Equal(t, in.ID, got.ID)
	assert.Equal(t, in.AllowedSubjectFormats, got.AllowedSubjectFormats)
	assert.Equal(t, in.ActionOverrides, got.ActionOverrides)
	assert.Equal(t, 42, got.MaxActionsPerHour)
	assert.Equal(t, StatusPaused, got.Status)
	require.NotNil(t, got.LastVerifiedAt)
}

func TestMongoDocs_SubjectLinkRoundTrip(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Millisecond)
	in := &SubjectLink{
		ID: id.NewSSFLinkID(), AppID: id.NewAppID(), EnvID: id.NewEnvironmentID(),
		Issuer: "https://i", Subject: "u1", UserID: id.NewUserID(),
		Source: SourceSocial, CreatedAt: now, LastSeenAt: now,
	}
	got, err := docToSubjectLink(subjectLinkToDoc(in))
	require.NoError(t, err)
	assert.Equal(t, in.UserID, got.UserID)
	assert.Equal(t, SourceSocial, got.Source)
}

func TestMongoDocs_ReceivedEventRoundTrip(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Millisecond)
	in := &ReceivedEvent{
		ID: id.NewSSFEventID(), StreamID: id.NewSSFStreamID(), JTI: "j",
		EventType: "e", SubjectJSON: `{"format":"opaque","id":"u"}`,
		Outcome: OutcomeUnresolved, ReceivedAt: now,
	}
	got, err := docToReceivedEvent(receivedEventToDoc(in))
	require.NoError(t, err)
	assert.Equal(t, "j", got.JTI)
	assert.True(t, got.ResolvedUserID.IsNil())
}

func TestMongoDocs_SignalRoundTrip(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Millisecond)
	in := &Signal{
		ID: id.NewSSFSignalID(), AppID: id.NewAppID(), EnvID: id.NewEnvironmentID(),
		UserID: id.NewUserID(), StreamID: id.NewSSFStreamID(),
		EventType: "e", Severity: 75, Reason: "why",
		EventAt: now, ExpiresAt: now.Add(time.Hour), CreatedAt: now,
	}
	got, err := docToSignal(signalToDoc(in))
	require.NoError(t, err)
	assert.Equal(t, 75, got.Severity)
	assert.Equal(t, in.StreamID, got.StreamID)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./plugins/sharedsignals/ -run TestMongoDocs -v`
Expected: FAIL, the doc converters are undefined.

- [ ] **Step 3: Write the mongo store**

Create `plugins/sharedsignals/store_mongo.go`:

```go
package sharedsignals

import (
	"context"
	"errors"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/xraph/grove"
	"github.com/xraph/grove/drivers/mongodriver"

	"github.com/xraph/authsome/id"
)

// MongoStore implements Store on MongoDB.
type MongoStore struct {
	db  *grove.DB
	mdb *mongodriver.MongoDB
}

// NewMongoStore builds a MongoDB-backed store.
func NewMongoStore(db *grove.DB) *MongoStore {
	return &MongoStore{db: db, mdb: mongodriver.Unwrap(db)}
}

var _ Store = (*MongoStore)(nil)

// ──────────────────────────────────────────────────
// Documents
// ──────────────────────────────────────────────────

type inboundStreamDoc struct {
	grove.BaseModel `grove:"table:authsome_ssf_inbound_streams"`

	ID                    string     `bson:"_id"`
	AppID                 string     `bson:"app_id"`
	EnvID                 string     `bson:"env_id"`
	Name                  string     `bson:"name"`
	Issuer                string     `bson:"issuer"`
	Audience              string     `bson:"audience"`
	JWKSURI               string     `bson:"jwks_uri"`
	PushPathHash          string     `bson:"push_path_hash"`
	PushTokenHash         string     `bson:"push_token_hash"`
	AllowedEventTypes     []string   `bson:"allowed_event_types"`
	AllowedSubjectFormats []string   `bson:"allowed_subject_formats"`
	VerifiedDomains       []string   `bson:"verified_domains"`
	ActionOverrides       string     `bson:"action_overrides"`
	EnforcementMode       string     `bson:"enforcement_mode"`
	Status                string     `bson:"status"`
	MaxActionsPerHour     int        `bson:"max_actions_per_hour"`
	PendingVerifyState    string     `bson:"pending_verify_state"`
	LastVerifiedAt        *time.Time `bson:"last_verified_at,omitempty"`
	CreatedAt             time.Time  `bson:"created_at"`
	UpdatedAt             time.Time  `bson:"updated_at"`
}

type subjectLinkDoc struct {
	grove.BaseModel `grove:"table:authsome_ssf_subject_links"`

	ID         string    `bson:"_id"`
	AppID      string    `bson:"app_id"`
	EnvID      string    `bson:"env_id"`
	Issuer     string    `bson:"issuer"`
	Subject    string    `bson:"subject"`
	UserID     string    `bson:"user_id"`
	Source     string    `bson:"source"`
	CreatedAt  time.Time `bson:"created_at"`
	LastSeenAt time.Time `bson:"last_seen_at"`
}

type receivedEventDoc struct {
	grove.BaseModel `grove:"table:authsome_ssf_received_events"`

	ID             string    `bson:"_id"`
	StreamID       string    `bson:"stream_id"`
	JTI            string    `bson:"jti"`
	EventType      string    `bson:"event_type"`
	SubjectJSON    string    `bson:"subject_json"`
	ResolvedUserID string    `bson:"resolved_user_id"`
	Outcome        string    `bson:"outcome"`
	ActionTaken    string    `bson:"action_taken"`
	Error          string    `bson:"error"`
	ReceivedAt     time.Time `bson:"received_at"`
}

type signalDoc struct {
	grove.BaseModel `grove:"table:authsome_ssf_signals"`

	ID        string    `bson:"_id"`
	AppID     string    `bson:"app_id"`
	EnvID     string    `bson:"env_id"`
	UserID    string    `bson:"user_id"`
	StreamID  string    `bson:"stream_id"`
	EventType string    `bson:"event_type"`
	Severity  int       `bson:"severity"`
	Reason    string    `bson:"reason"`
	EventAt   time.Time `bson:"event_at"`
	ExpiresAt time.Time `bson:"expires_at"`
	CreatedAt time.Time `bson:"created_at"`
}

// ──────────────────────────────────────────────────
// Converters
// ──────────────────────────────────────────────────

func inboundStreamToDoc(s *InboundStream) *inboundStreamDoc {
	return &inboundStreamDoc{
		ID: s.ID.String(), AppID: s.AppID.String(), EnvID: s.EnvID.String(),
		Name: s.Name, Issuer: s.Issuer, Audience: s.Audience, JWKSURI: s.JWKSURI,
		PushPathHash: s.PushPathHash, PushTokenHash: s.PushTokenHash,
		AllowedEventTypes:     s.AllowedEventTypes,
		AllowedSubjectFormats: s.AllowedSubjectFormats,
		VerifiedDomains:       s.VerifiedDomains,
		ActionOverrides:       encodeJSON(s.ActionOverrides, "{}"),
		EnforcementMode:       s.EnforcementMode, Status: s.Status,
		MaxActionsPerHour: s.MaxActionsPerHour,
		PendingVerifyState: s.PendingVerifyState, LastVerifiedAt: s.LastVerifiedAt,
		CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt,
	}
}

func docToInboundStream(d *inboundStreamDoc) (*InboundStream, error) {
	streamID, err := id.Parse(d.ID)
	if err != nil {
		return nil, err
	}
	overrides, err := decodeStringMap(d.ActionOverrides)
	if err != nil {
		return nil, err
	}
	return &InboundStream{
		ID: streamID, AppID: parseOptionalID(d.AppID), EnvID: parseOptionalID(d.EnvID),
		Name: d.Name, Issuer: d.Issuer, Audience: d.Audience, JWKSURI: d.JWKSURI,
		PushPathHash: d.PushPathHash, PushTokenHash: d.PushTokenHash,
		AllowedEventTypes:     d.AllowedEventTypes,
		AllowedSubjectFormats: d.AllowedSubjectFormats,
		VerifiedDomains:       d.VerifiedDomains,
		ActionOverrides:       overrides,
		EnforcementMode:       d.EnforcementMode, Status: d.Status,
		MaxActionsPerHour: d.MaxActionsPerHour,
		PendingVerifyState: d.PendingVerifyState, LastVerifiedAt: d.LastVerifiedAt,
		CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt,
	}, nil
}

func subjectLinkToDoc(l *SubjectLink) *subjectLinkDoc {
	return &subjectLinkDoc{
		ID: l.ID.String(), AppID: l.AppID.String(), EnvID: l.EnvID.String(),
		Issuer: l.Issuer, Subject: l.Subject, UserID: l.UserID.String(),
		Source: l.Source, CreatedAt: l.CreatedAt, LastSeenAt: l.LastSeenAt,
	}
}

func docToSubjectLink(d *subjectLinkDoc) (*SubjectLink, error) {
	linkID, err := id.Parse(d.ID)
	if err != nil {
		return nil, err
	}
	return &SubjectLink{
		ID: linkID, AppID: parseOptionalID(d.AppID), EnvID: parseOptionalID(d.EnvID),
		Issuer: d.Issuer, Subject: d.Subject, UserID: parseOptionalID(d.UserID),
		Source: d.Source, CreatedAt: d.CreatedAt, LastSeenAt: d.LastSeenAt,
	}, nil
}

func receivedEventToDoc(e *ReceivedEvent) *receivedEventDoc {
	resolved := ""
	if !e.ResolvedUserID.IsNil() {
		resolved = e.ResolvedUserID.String()
	}
	return &receivedEventDoc{
		ID: e.ID.String(), StreamID: e.StreamID.String(), JTI: e.JTI,
		EventType: e.EventType, SubjectJSON: e.SubjectJSON, ResolvedUserID: resolved,
		Outcome: e.Outcome, ActionTaken: e.ActionTaken, Error: e.Error,
		ReceivedAt: e.ReceivedAt,
	}
}

func docToReceivedEvent(d *receivedEventDoc) (*ReceivedEvent, error) {
	eventID, err := id.Parse(d.ID)
	if err != nil {
		return nil, err
	}
	return &ReceivedEvent{
		ID: eventID, StreamID: parseOptionalID(d.StreamID), JTI: d.JTI,
		EventType: d.EventType, SubjectJSON: d.SubjectJSON,
		ResolvedUserID: parseOptionalID(d.ResolvedUserID),
		Outcome: d.Outcome, ActionTaken: d.ActionTaken, Error: d.Error,
		ReceivedAt: d.ReceivedAt,
	}, nil
}

func signalToDoc(s *Signal) *signalDoc {
	return &signalDoc{
		ID: s.ID.String(), AppID: s.AppID.String(), EnvID: s.EnvID.String(),
		UserID: s.UserID.String(), StreamID: s.StreamID.String(),
		EventType: s.EventType, Severity: s.Severity, Reason: s.Reason,
		EventAt: s.EventAt, ExpiresAt: s.ExpiresAt, CreatedAt: s.CreatedAt,
	}
}

func docToSignal(d *signalDoc) (*Signal, error) {
	signalID, err := id.Parse(d.ID)
	if err != nil {
		return nil, err
	}
	return &Signal{
		ID: signalID, AppID: parseOptionalID(d.AppID), EnvID: parseOptionalID(d.EnvID),
		UserID: parseOptionalID(d.UserID), StreamID: parseOptionalID(d.StreamID),
		EventType: d.EventType, Severity: d.Severity, Reason: d.Reason,
		EventAt: d.EventAt, ExpiresAt: d.ExpiresAt, CreatedAt: d.CreatedAt,
	}, nil
}

// ──────────────────────────────────────────────────
// Store methods
// ──────────────────────────────────────────────────

func mongoErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, mongo.ErrNoDocuments) {
		return ErrNotFound
	}
	return err
}

func isMongoDuplicate(err error) bool {
	if err == nil {
		return false
	}
	if mongo.IsDuplicateKeyError(err) {
		return true
	}
	return strings.Contains(strings.ToLower(err.Error()), "duplicate key")
}

func (s *MongoStore) CreateInboundStream(ctx context.Context, in *InboundStream) error {
	now := time.Now()
	if in.CreatedAt.IsZero() {
		in.CreatedAt = now
	}
	in.UpdatedAt = now
	_, err := s.mdb.Collection(colInboundStreams).InsertOne(ctx, inboundStreamToDoc(in))
	return mongoErr(err)
}

func (s *MongoStore) GetInboundStream(ctx context.Context, streamID id.SSFStreamID) (*InboundStream, error) {
	d := new(inboundStreamDoc)
	if err := s.mdb.Collection(colInboundStreams).
		FindOne(ctx, bson.M{"_id": streamID.String()}).Decode(d); err != nil {
		return nil, mongoErr(err)
	}
	return docToInboundStream(d)
}

func (s *MongoStore) GetInboundStreamByPushPathHash(ctx context.Context, hash string) (*InboundStream, error) {
	d := new(inboundStreamDoc)
	if err := s.mdb.Collection(colInboundStreams).
		FindOne(ctx, bson.M{"push_path_hash": hash}).Decode(d); err != nil {
		return nil, mongoErr(err)
	}
	return docToInboundStream(d)
}

func (s *MongoStore) ListInboundStreams(ctx context.Context, appID id.AppID) ([]*InboundStream, error) {
	cur, err := s.mdb.Collection(colInboundStreams).Find(ctx,
		bson.M{"app_id": appID.String()},
		options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}))
	if err != nil {
		return nil, mongoErr(err)
	}
	defer cur.Close(ctx) //nolint:errcheck // cursor close

	var out []*InboundStream
	for cur.Next(ctx) {
		d := new(inboundStreamDoc)
		if derr := cur.Decode(d); derr != nil {
			return nil, derr
		}
		converted, cerr := docToInboundStream(d)
		if cerr != nil {
			return nil, cerr
		}
		out = append(out, converted)
	}
	return out, cur.Err()
}

func (s *MongoStore) UpdateInboundStream(ctx context.Context, in *InboundStream) error {
	in.UpdatedAt = time.Now()
	res, err := s.mdb.Collection(colInboundStreams).ReplaceOne(ctx,
		bson.M{"_id": in.ID.String()}, inboundStreamToDoc(in))
	if err != nil {
		return mongoErr(err)
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *MongoStore) DeleteInboundStream(ctx context.Context, streamID id.SSFStreamID) error {
	res, err := s.mdb.Collection(colInboundStreams).
		DeleteOne(ctx, bson.M{"_id": streamID.String()})
	if err != nil {
		return mongoErr(err)
	}
	if res.DeletedCount == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *MongoStore) UpsertSubjectLink(ctx context.Context, l *SubjectLink) error {
	now := time.Now()
	if l.CreatedAt.IsZero() {
		l.CreatedAt = now
	}
	l.LastSeenAt = now
	filter := bson.M{
		"app_id": l.AppID.String(), "env_id": l.EnvID.String(),
		"issuer": l.Issuer, "subject": l.Subject,
	}
	_, err := s.mdb.Collection(colSubjectLinks).ReplaceOne(ctx, filter,
		subjectLinkToDoc(l), options.Replace().SetUpsert(true))
	return mongoErr(err)
}

func (s *MongoStore) GetSubjectLink(ctx context.Context, appID id.AppID,
	envID id.EnvironmentID, issuer, subject string) (*SubjectLink, error) {
	d := new(subjectLinkDoc)
	if err := s.mdb.Collection(colSubjectLinks).FindOne(ctx, bson.M{
		"app_id": appID.String(), "env_id": envID.String(),
		"issuer": issuer, "subject": subject,
	}).Decode(d); err != nil {
		return nil, mongoErr(err)
	}
	return docToSubjectLink(d)
}

func (s *MongoStore) InsertReceivedEvent(ctx context.Context, e *ReceivedEvent) error {
	if e.ReceivedAt.IsZero() {
		e.ReceivedAt = time.Now()
	}
	_, err := s.mdb.Collection(colReceivedEvents).InsertOne(ctx, receivedEventToDoc(e))
	if isMongoDuplicate(err) {
		return ErrDuplicateJTI
	}
	return mongoErr(err)
}

func (s *MongoStore) UpdateReceivedEvent(ctx context.Context, e *ReceivedEvent) error {
	res, err := s.mdb.Collection(colReceivedEvents).ReplaceOne(ctx,
		bson.M{"_id": e.ID.String()}, receivedEventToDoc(e))
	if err != nil {
		return mongoErr(err)
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *MongoStore) CountActionsSince(ctx context.Context,
	streamID id.SSFStreamID, since time.Time) (int, error) {
	n, err := s.mdb.Collection(colReceivedEvents).CountDocuments(ctx, bson.M{
		"stream_id":    streamID.String(),
		"action_taken": bson.M{"$ne": ""},
		"received_at":  bson.M{"$gt": since},
	})
	if err != nil {
		return 0, mongoErr(err)
	}
	return int(n), nil
}

func (s *MongoStore) CreateSignal(ctx context.Context, sig *Signal) error {
	if sig.CreatedAt.IsZero() {
		sig.CreatedAt = time.Now()
	}
	_, err := s.mdb.Collection(colSignals).InsertOne(ctx, signalToDoc(sig))
	return mongoErr(err)
}

func (s *MongoStore) ListActiveSignals(ctx context.Context, appID id.AppID,
	userID id.UserID, now time.Time) ([]*Signal, error) {
	cur, err := s.mdb.Collection(colSignals).Find(ctx, bson.M{
		"app_id": appID.String(), "user_id": userID.String(),
		"expires_at": bson.M{"$gt": now},
	}, options.Find().SetSort(bson.D{{Key: "severity", Value: -1}}))
	if err != nil {
		return nil, mongoErr(err)
	}
	defer cur.Close(ctx) //nolint:errcheck // cursor close

	var out []*Signal
	for cur.Next(ctx) {
		d := new(signalDoc)
		if derr := cur.Decode(d); derr != nil {
			return nil, derr
		}
		converted, cerr := docToSignal(d)
		if cerr != nil {
			return nil, cerr
		}
		out = append(out, converted)
	}
	return out, cur.Err()
}
```

- [ ] **Step 4: Register the mongo migration**

Now that the doc types exist, append to the `init()` in `plugins/sharedsignals/migrations.go`, and add the imports it needs (`fmt`, `go.mongodb.org/mongo-driver/v2/bson`, `go.mongodb.org/mongo-driver/v2/mongo`, `go.mongodb.org/mongo-driver/v2/mongo/options`, `github.com/xraph/grove/drivers/mongodriver/mongomigrate`):

```go
	MongoMigrations.MustRegister(&migrate.Migration{
		Name:    "create_sharedsignals_collections",
		Version: "20260824000001",
		Up: func(ctx context.Context, exec migrate.Executor) error {
			mexec, ok := exec.(*mongomigrate.Executor)
			if !ok {
				return fmt.Errorf("sharedsignals: expected mongomigrate executor, got %T", exec)
			}
			for _, model := range []any{
				(*inboundStreamDoc)(nil), (*subjectLinkDoc)(nil),
				(*receivedEventDoc)(nil), (*signalDoc)(nil),
			} {
				if err := mexec.CreateCollection(ctx, model); err != nil {
					return err
				}
			}
			if err := mexec.CreateIndexes(ctx, colInboundStreams, []mongo.IndexModel{
				{
					Keys:    bson.D{{Key: "push_path_hash", Value: 1}},
					Options: options.Index().SetUnique(true),
				},
				{Keys: bson.D{{Key: "app_id", Value: 1}, {Key: "created_at", Value: -1}}},
			}); err != nil {
				return err
			}
			if err := mexec.CreateIndexes(ctx, colSubjectLinks, []mongo.IndexModel{
				{
					Keys: bson.D{
						{Key: "app_id", Value: 1}, {Key: "env_id", Value: 1},
						{Key: "issuer", Value: 1}, {Key: "subject", Value: 1},
					},
					Options: options.Index().SetUnique(true),
				},
				{Keys: bson.D{{Key: "user_id", Value: 1}}},
			}); err != nil {
				return err
			}
			// This unique index is the replay guard on mongo. Without it a
			// replayed SET revokes sessions a second time.
			if err := mexec.CreateIndexes(ctx, colReceivedEvents, []mongo.IndexModel{
				{
					Keys:    bson.D{{Key: "stream_id", Value: 1}, {Key: "jti", Value: 1}},
					Options: options.Index().SetUnique(true),
				},
				{Keys: bson.D{{Key: "stream_id", Value: 1}, {Key: "received_at", Value: -1}}},
			}); err != nil {
				return err
			}
			return mexec.CreateIndexes(ctx, colSignals, []mongo.IndexModel{
				{Keys: bson.D{
					{Key: "app_id", Value: 1}, {Key: "user_id", Value: 1},
					{Key: "expires_at", Value: -1},
				}},
			})
		},
		Down: func(ctx context.Context, exec migrate.Executor) error {
			mexec, ok := exec.(*mongomigrate.Executor)
			if !ok {
				return fmt.Errorf("sharedsignals: expected mongomigrate executor, got %T", exec)
			}
			for _, model := range []any{
				(*signalDoc)(nil), (*receivedEventDoc)(nil),
				(*subjectLinkDoc)(nil), (*inboundStreamDoc)(nil),
			} {
				if err := mexec.DropCollection(ctx, model); err != nil {
					return err
				}
			}
			return nil
		},
	})
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./plugins/sharedsignals/ -run 'TestMongoDocs|TestMigrationGroups' -v`
Expected: PASS, five tests.

- [ ] **Step 6: Commit**

```bash
git add plugins/sharedsignals/store_mongo.go plugins/sharedsignals/store_mongo_test.go plugins/sharedsignals/migrations.go
git commit -m "feat(sharedsignals): mongo store and collection migrations"
```

---

### Task 10: Plugin skeleton, config and settings

**Files:**
- Create: `plugins/sharedsignals/plugin.go`
- Test: `plugins/sharedsignals/plugin_test.go`

**Interfaces:**
- Consumes: the stores from Tasks 6, 8 and 9; the migration groups from Task 7; `jwksclient.New` from Task 4.
- Produces: `Plugin` struct; `New(cfg ...Config) *Plugin`; `Config{Audience string, SignalTTL time.Duration, MaxActionsPerHour int, ClockSkew, MaxSETAge time.Duration, MaxBodyBytes int64}`; `(*Plugin).Name() string` returning `"sharedsignals"`; `(*Plugin).OnInit`, `(*Plugin).MigrationGroups`, `(*Plugin).DeclareSettings`; `(*Plugin).SetStore(Store)` for tests; the settings `SettingEnabled`, `SettingSignalTTLHours`, `SettingMaxActionsPerHour`, `SettingCAEPSignalWeight`.

- [ ] **Step 1: Write the failing test**

Create `plugins/sharedsignals/plugin_test.go`:

```go
package sharedsignals

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/plugin"
	"github.com/xraph/authsome/settings"
)

func TestPlugin_Name(t *testing.T) {
	assert.Equal(t, "sharedsignals", New().Name())
}

func TestPlugin_ConfigDefaults(t *testing.T) {
	p := New()
	assert.Equal(t, 24*time.Hour, p.config.SignalTTL)
	assert.Equal(t, 100, p.config.MaxActionsPerHour)
	assert.Equal(t, 5*time.Minute, p.config.ClockSkew)
	assert.Equal(t, 24*time.Hour, p.config.MaxSETAge)
	assert.Equal(t, int64(64*1024), p.config.MaxBodyBytes)
}

func TestPlugin_ConfigOverridesSurvive(t *testing.T) {
	p := New(Config{Audience: "https://a/ssf", SignalTTL: time.Hour, MaxBodyBytes: 1234})
	assert.Equal(t, "https://a/ssf", p.config.Audience)
	assert.Equal(t, time.Hour, p.config.SignalTTL)
	assert.Equal(t, int64(1234), p.config.MaxBodyBytes)
	// Unset fields still get defaults.
	assert.Equal(t, 100, p.config.MaxActionsPerHour)
}

func TestPlugin_MigrationGroups(t *testing.T) {
	p := New()
	assert.Len(t, p.MigrationGroups("pg"), 1)
	assert.Len(t, p.MigrationGroups("sqlite"), 1)
	assert.Len(t, p.MigrationGroups("mongo"), 1)
	assert.Empty(t, p.MigrationGroups("memory"))
}

func TestPlugin_DeclareSettings(t *testing.T) {
	m := settings.NewManager()
	require.NoError(t, New().DeclareSettings(m))
}

// Without a database the plugin must still come up on the memory store rather
// than leaving a nil store for the receiver to panic on.
func TestPlugin_OnInitFallsBackToMemory(t *testing.T) {
	p := New()
	require.NoError(t, p.OnInit(context.Background(), stubEngine{}))
	require.NotNil(t, p.store)
	_, err := p.store.GetInboundStreamByPushPathHash(context.Background(), "x")
	require.ErrorIs(t, err, ErrNotFound)
	require.NotNil(t, p.jwks)
}

// Compile-time proof that the plugin implements what the engine looks for.
func TestPlugin_ImplementsHooks(t *testing.T) {
	var p any = New()
	_, ok := p.(plugin.Plugin)
	assert.True(t, ok)
	_, ok = p.(plugin.OnInit)
	assert.True(t, ok)
	_, ok = p.(plugin.MigrationProvider)
	assert.True(t, ok)
	_, ok = p.(plugin.SettingsProvider)
	assert.True(t, ok)
	_, ok = p.(plugin.RouteProvider)
	assert.True(t, ok)
}
```

You also need a `stubEngine` that satisfies `plugin.Engine` with zero values.
Create `plugins/sharedsignals/stub_engine_test.go`:

```go
package sharedsignals

import (
	"context"

	log "github.com/xraph/go-utils/log"
	"github.com/xraph/grove"

	"github.com/xraph/forge"
	"github.com/xraph/forge/extensions/auth"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/apikey"
	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/ceremony"
	"github.com/xraph/authsome/hook"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugin"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/settings"
	"github.com/xraph/authsome/store"
	"github.com/xraph/authsome/tokenformat"
	"github.com/xraph/authsome/user"
)

// stubEngine is the smallest thing that satisfies plugin.Engine. It exists so
// OnInit can be tested without standing up a real engine.
type stubEngine struct{}

var _ plugin.Engine = stubEngine{}

func (stubEngine) Store() store.Store                  { return nil }
func (stubEngine) DB() *grove.DB                       { return nil }
func (stubEngine) Plugins() *plugin.Registry           { return nil }
func (stubEngine) Plugin(string) plugin.Plugin         { return nil }
func (stubEngine) Hooks() *hook.Bus                    { return nil }
func (stubEngine) Logger() log.Logger                  { return log.NewNoopLogger() }
func (stubEngine) Settings() *settings.Manager         { return nil }
func (stubEngine) Chronicle() bridge.Chronicle         { return nil }
func (stubEngine) Relay() bridge.EventRelay            { return nil }
func (stubEngine) Herald() bridge.Herald               { return nil }
func (stubEngine) Mailer() bridge.Mailer               { return nil }
func (stubEngine) SMSSender() bridge.SMSSender         { return nil }
func (stubEngine) Ledger() bridge.Ledger               { return nil }
func (stubEngine) TokenEncryptor() bridge.Encryptor    { return nil }
func (stubEngine) CeremonyStore() ceremony.Store       { return nil }
func (stubEngine) APIKeyStore() apikey.Store           { return nil }
func (stubEngine) AuthMiddleware() forge.Middleware    { return nil }
func (stubEngine) AuthRegistry() auth.Registry         { return nil }
func (stubEngine) PlatformAppID() id.AppID             { return id.Nil }
func (stubEngine) DefaultAppID() string                { return "" }
func (stubEngine) BasePath() string                    { return "" }
func (stubEngine) TokenFormatForApp(string) tokenformat.Format { return nil }

func (stubEngine) SessionConfigForApp(context.Context, id.AppID, ...id.EnvironmentID) account.SessionConfig {
	return account.SessionConfig{}
}
func (stubEngine) ResolveSessionByToken(string) (*session.Session, error) { return nil, nil }
func (stubEngine) ResolveUser(string) (*user.User, error)                 { return nil, nil }
func (stubEngine) GetUser(context.Context, id.UserID) (*user.User, error) { return nil, nil }
func (stubEngine) EnsureDefaultRole(context.Context, id.AppID, id.UserID)  {}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./plugins/sharedsignals/ -run TestPlugin -v`
Expected: FAIL, `New` is undefined.

If `stubEngine` does not compile, `plugin.Engine` has drifted. Run
`go doc github.com/xraph/authsome/plugin.Engine` and add the missing methods
rather than changing the interface.

- [ ] **Step 3: Write the plugin**

Create `plugins/sharedsignals/plugin.go`:

```go
package sharedsignals

import (
	"context"
	"time"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/grove/migrate"

	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/formconfig"
	"github.com/xraph/authsome/hook"
	"github.com/xraph/authsome/plugin"
	"github.com/xraph/authsome/plugins/sharedsignals/jwksclient"
	"github.com/xraph/authsome/settings"
	"github.com/xraph/authsome/store"
)

// Compile-time interface checks.
var (
	_ plugin.Plugin            = (*Plugin)(nil)
	_ plugin.OnInit            = (*Plugin)(nil)
	_ plugin.MigrationProvider = (*Plugin)(nil)
	_ plugin.SettingsProvider  = (*Plugin)(nil)
	_ plugin.RouteProvider     = (*Plugin)(nil)
)

func intPtr(v int) *int { return &v }

// ──────────────────────────────────────────────────
// Dynamic settings
// ──────────────────────────────────────────────────

var (
	// SettingEnabled turns the receiver on and off without deleting streams.
	SettingEnabled = settings.Define("sharedsignals.enabled", true,
		settings.WithDisplayName("Shared Signals Enabled"),
		settings.WithDescription("Accept inbound CAEP events from configured streams"),
		settings.WithCategory("Shared Signals"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithEnforceable(),
		settings.WithHelpText("Turn the receiver off to stop acting on inbound events without deleting any stream"),
		settings.WithOrder(10),
	)

	// SettingSignalTTLHours is how long a received signal keeps affecting risk.
	SettingSignalTTLHours = settings.Define("sharedsignals.signal_ttl_hours", 24,
		settings.WithDisplayName("Signal TTL (hours)"),
		settings.WithDescription("How long a received CAEP signal keeps influencing the risk score"),
		settings.WithCategory("Shared Signals"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithInputType(formconfig.FieldNumber),
		settings.WithUIValidation(formconfig.Validation{Min: intPtr(1), Max: intPtr(720)}),
		settings.WithHelpText("Signals decay to zero over this window. Default: 24"),
		settings.WithOrder(20),
	)

	// SettingMaxActionsPerHour is the circuit breaker default for new streams.
	SettingMaxActionsPerHour = settings.Define("sharedsignals.max_actions_per_hour", 100,
		settings.WithDisplayName("Max Actions Per Hour"),
		settings.WithDescription("Actions one stream may take in an hour before it is paused"),
		settings.WithCategory("Shared Signals"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithInputType(formconfig.FieldNumber),
		settings.WithUIValidation(formconfig.Validation{Min: intPtr(1), Max: intPtr(100000)}),
		settings.WithHelpText("A transmitter that crosses this is paused and an alert is raised"),
		settings.WithOrder(30),
	)

	// SettingCAEPSignalWeight is the weight the risk engine gives our signals.
	SettingCAEPSignalWeight = settings.Define("sharedsignals.risk_weight", 2,
		settings.WithDisplayName("Risk Weight"),
		settings.WithDescription("Weight the risk engine applies to CAEP signals"),
		settings.WithCategory("Shared Signals"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithInputType(formconfig.FieldNumber),
		settings.WithUIValidation(formconfig.Validation{Min: intPtr(1), Max: intPtr(10)}),
		settings.WithHelpText("A CAEP event is higher confidence than an IP heuristic. Default: 2"),
		settings.WithOrder(40),
	)
)

// Config configures the plugin. Every field has a sane default.
type Config struct {
	// Audience is the aud value inbound SETs must carry. Streams inherit it
	// at creation when they do not set their own.
	Audience string

	// SignalTTL is how long a stored signal stays active.
	SignalTTL time.Duration

	// MaxActionsPerHour is the circuit breaker default for new streams.
	MaxActionsPerHour int

	// ClockSkew is the tolerance for an iat slightly in the future.
	ClockSkew time.Duration

	// MaxSETAge is how far in the past an iat may be. Transmitters retry for
	// a long time, so this is generous.
	MaxSETAge time.Duration

	// MaxBodyBytes bounds the push request body.
	MaxBodyBytes int64
}

func (c *Config) defaults() {
	if c.SignalTTL == 0 {
		c.SignalTTL = 24 * time.Hour
	}
	if c.MaxActionsPerHour == 0 {
		c.MaxActionsPerHour = 100
	}
	if c.ClockSkew == 0 {
		c.ClockSkew = 5 * time.Minute
	}
	if c.MaxSETAge == 0 {
		c.MaxSETAge = 24 * time.Hour
	}
	if c.MaxBodyBytes == 0 {
		c.MaxBodyBytes = 64 * 1024
	}
}

// Plugin receives Shared Signals events and turns them into session
// revocations and durable risk signals.
type Plugin struct {
	config Config

	store     Store
	authStore store.Store
	jwks      *jwksclient.Client

	engine      plugin.Engine
	revoker     plugin.SessionRevoker
	logger      log.Logger
	chronicle   bridge.Chronicle
	relay       bridge.EventRelay
	hooks       *hook.Bus
	settingsMgr *settings.Manager
}

// New builds the plugin. Config is optional.
func New(cfg ...Config) *Plugin {
	var c Config
	if len(cfg) > 0 {
		c = cfg[0]
	}
	c.defaults()
	return &Plugin{config: c}
}

// Name returns the plugin name.
func (p *Plugin) Name() string { return "sharedsignals" }

// SetStore overrides the backing store. Tests use it; production wiring goes
// through OnInit.
func (p *Plugin) SetStore(s Store) { p.store = s }

// OnInit captures engine references and picks a store for the driver in use.
func (p *Plugin) OnInit(_ context.Context, engine plugin.Engine) error {
	p.engine = engine
	p.authStore = engine.Store()
	p.chronicle = engine.Chronicle()
	p.relay = engine.Relay()
	p.hooks = engine.Hooks()
	p.settingsMgr = engine.Settings()

	p.logger = engine.Logger()
	if p.logger == nil {
		p.logger = log.NewNoopLogger()
	}

	// Revoking through the engine keeps the AfterSessionRevoke hooks, the
	// hook bus and the outbound relay firing. An engine that cannot revoke
	// leaves us able to record signals but not to act, which the receiver
	// reports rather than hiding.
	if r, ok := engine.(plugin.SessionRevoker); ok {
		p.revoker = r
	}

	if p.store == nil {
		if db := engine.DB(); db != nil {
			switch db.Driver().Name() {
			case "pg":
				p.store = NewPostgresStore(db)
			case "sqlite":
				p.store = NewSqliteStore(db)
			case "mongo":
				p.store = NewMongoStore(db)
			}
		}
	}
	if p.store == nil {
		p.store = NewMemoryStore()
	}

	p.jwks = jwksclient.New(jwksclient.Options{})

	return nil
}

// MigrationGroups implements plugin.MigrationProvider.
func (p *Plugin) MigrationGroups(driverName string) []*migrate.Group {
	switch driverName {
	case "pg", "postgres", "postgresql":
		return []*migrate.Group{PostgresMigrations}
	case "sqlite", "sqlite3":
		return []*migrate.Group{SqliteMigrations}
	case "mongo", "mongodb":
		return []*migrate.Group{MongoMigrations}
	default:
		return nil
	}
}

// DeclareSettings implements plugin.SettingsProvider.
func (p *Plugin) DeclareSettings(m *settings.Manager) error {
	if err := settings.RegisterTyped(m, "sharedsignals", SettingEnabled); err != nil {
		return err
	}
	if err := settings.RegisterTyped(m, "sharedsignals", SettingSignalTTLHours); err != nil {
		return err
	}
	if err := settings.RegisterTyped(m, "sharedsignals", SettingMaxActionsPerHour); err != nil {
		return err
	}
	return settings.RegisterTyped(m, "sharedsignals", SettingCAEPSignalWeight)
}
```

- [ ] **Step 4: Add a placeholder RegisterRoutes so the interface check compiles**

The `plugin.RouteProvider` assertion needs the method now; Task 14 fills in the
body. Append to `plugins/sharedsignals/plugin.go`:

```go
// RegisterRoutes implements plugin.RouteProvider. The receiver endpoint lands
// in receiver.go and the admin CRUD in admin.go.
func (p *Plugin) RegisterRoutes(router forge.Router) error {
	if err := p.registerReceiverRoutes(router); err != nil {
		return err
	}
	return p.registerAdminRoutes(router)
}
```

Add `"github.com/xraph/forge"` to the import block, and create two temporary
stubs at the bottom of `plugin.go` that Tasks 14 and 16 replace:

```go
func (p *Plugin) registerReceiverRoutes(_ forge.Router) error { return nil }
func (p *Plugin) registerAdminRoutes(_ forge.Router) error    { return nil }
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./plugins/sharedsignals/ -run TestPlugin -v`
Expected: PASS, six tests.

- [ ] **Step 6: Commit**

```bash
git add plugins/sharedsignals/plugin.go plugins/sharedsignals/plugin_test.go plugins/sharedsignals/stub_engine_test.go
git commit -m "feat(sharedsignals): plugin lifecycle, config and settings"
```

---

### Task 11: Subject resolution

**Files:**
- Create: `plugins/sharedsignals/subject.go`
- Test: `plugins/sharedsignals/subject_test.go`

**Interfaces:**
- Consumes: `caep.SubjectID` from Task 1, `Store` from Task 6, `Plugin` from Task 10.
- Produces: `Resolution{UserID id.UserID, SessionID id.SessionID, Outcome string}`; `(*Plugin).resolveSubject(ctx context.Context, s *InboundStream, subj caep.SubjectID) (Resolution, error)`.

This is where a stranger's chosen identifier becomes one of our users, so it is the second half of the security story. Read the tests as the specification.

- [ ] **Step 1: Write the failing test**

Create `plugins/sharedsignals/subject_test.go`:

```go
package sharedsignals

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/sharedsignals/caep"
	"github.com/xraph/authsome/store/memory"
	"github.com/xraph/authsome/user"
)

type resolveFixture struct {
	plugin *Plugin
	stream *InboundStream
	user   *user.User
	appID  id.AppID
	envID  id.EnvironmentID
}

func newResolveFixture(t *testing.T, verifiedEmail bool) resolveFixture {
	t.Helper()
	ctx := context.Background()
	appID, envID := id.NewAppID(), id.NewEnvironmentID()

	authStore := memory.New()
	u := &user.User{
		ID: id.NewUserID(), AppID: appID, EnvID: envID,
		Email: "target@corp.com", EmailVerified: verifiedEmail,
		Phone: "+15551234567", PhoneVerified: verifiedEmail,
	}
	require.NoError(t, authStore.CreateUserWithPrimaryEmail(ctx, u, &user.UserEmail{
		ID: id.NewUserEmailID(), UserID: u.ID, AppID: appID, EnvID: envID,
		Email: u.Email, Verified: verifiedEmail, Primary: true,
	}))

	p := New()
	p.store = NewMemoryStore()
	p.authStore = authStore

	stream := &InboundStream{
		ID: id.NewSSFStreamID(), AppID: appID, EnvID: envID,
		Issuer: "https://org.okta.com",
		AllowedSubjectFormats: []string{
			caep.FormatIssSub, caep.FormatOpaque, caep.FormatEmail,
			caep.FormatPhoneNumber, caep.FormatAliases,
		},
		VerifiedDomains: []string{"corp.com"},
		Status:          StatusEnabled,
	}
	require.NoError(t, p.store.CreateInboundStream(ctx, stream))

	require.NoError(t, p.store.UpsertSubjectLink(ctx, &SubjectLink{
		ID: id.NewSSFLinkID(), AppID: appID, EnvID: envID,
		Issuer: "https://org.okta.com", Subject: "okta-user-1",
		UserID: u.ID, Source: SourceSSO,
	}))

	return resolveFixture{plugin: p, stream: stream, user: u, appID: appID, envID: envID}
}

func TestResolveSubject_IssSub(t *testing.T) {
	f := newResolveFixture(t, true)
	got, err := f.plugin.resolveSubject(context.Background(), f.stream, caep.SubjectID{
		Format: caep.FormatIssSub, Issuer: "https://org.okta.com", Subject: "okta-user-1",
	})
	require.NoError(t, err)
	assert.Equal(t, OutcomeApplied, got.Outcome)
	assert.Equal(t, f.user.ID, got.UserID)
}

// An IdP may only speak for its own subjects. A SET from Okta claiming an
// Entra subject is a lateral-movement attempt.
func TestResolveSubject_IssSubRejectsForeignIssuer(t *testing.T) {
	f := newResolveFixture(t, true)
	got, err := f.plugin.resolveSubject(context.Background(), f.stream, caep.SubjectID{
		Format: caep.FormatIssSub, Issuer: "https://login.microsoftonline.com", Subject: "okta-user-1",
	})
	require.NoError(t, err)
	assert.Equal(t, OutcomeRejected, got.Outcome)
	assert.True(t, got.UserID.IsNil())
}

func TestResolveSubject_UnknownSubjectIsUnresolved(t *testing.T) {
	f := newResolveFixture(t, true)
	got, err := f.plugin.resolveSubject(context.Background(), f.stream, caep.SubjectID{
		Format: caep.FormatIssSub, Issuer: "https://org.okta.com", Subject: "nobody",
	})
	require.NoError(t, err)
	assert.Equal(t, OutcomeUnresolved, got.Outcome)
}

func TestResolveSubject_EmailInVerifiedDomain(t *testing.T) {
	f := newResolveFixture(t, true)
	got, err := f.plugin.resolveSubject(context.Background(), f.stream, caep.SubjectID{
		Format: caep.FormatEmail, Email: "target@corp.com",
	})
	require.NoError(t, err)
	assert.Equal(t, OutcomeApplied, got.Outcome)
	assert.Equal(t, f.user.ID, got.UserID)
}

func TestResolveSubject_EmailOutsideVerifiedDomainRejected(t *testing.T) {
	f := newResolveFixture(t, true)
	got, err := f.plugin.resolveSubject(context.Background(), f.stream, caep.SubjectID{
		Format: caep.FormatEmail, Email: "target@notours.com",
	})
	require.NoError(t, err)
	assert.Equal(t, OutcomeRejected, got.Outcome)
}

// The subtle one. If we matched an unverified address, anyone could attach
// ceo@corp.com to their own account and quietly absorb the CEO's revocation
// events, leaving the real account signed in.
func TestResolveSubject_EmailMustBeVerified(t *testing.T) {
	f := newResolveFixture(t, false)
	got, err := f.plugin.resolveSubject(context.Background(), f.stream, caep.SubjectID{
		Format: caep.FormatEmail, Email: "target@corp.com",
	})
	require.NoError(t, err)
	assert.Equal(t, OutcomeRejected, got.Outcome,
		"an unverified address must never resolve a subject")
}

func TestResolveSubject_EmailFormatNotAllowedOnStream(t *testing.T) {
	f := newResolveFixture(t, true)
	f.stream.AllowedSubjectFormats = []string{caep.FormatIssSub}
	got, err := f.plugin.resolveSubject(context.Background(), f.stream, caep.SubjectID{
		Format: caep.FormatEmail, Email: "target@corp.com",
	})
	require.NoError(t, err)
	assert.Equal(t, OutcomeRejected, got.Outcome)
}

func TestResolveSubject_UnsupportedFormatsRejected(t *testing.T) {
	f := newResolveFixture(t, true)
	for _, subj := range []caep.SubjectID{
		{Format: caep.FormatAccount, URI: "acct:target@corp.com"},
		{Format: caep.FormatURI, URI: "https://corp.com/u/1"},
		{Format: caep.FormatDID, URL: "did:example:1"},
		{Format: "something-invented"},
	} {
		got, err := f.plugin.resolveSubject(context.Background(), f.stream, subj)
		require.NoError(t, err)
		assert.Equal(t, OutcomeRejected, got.Outcome, "format %q", subj.Format)
	}
}

func TestResolveSubject_ComplexSubjectUsesUserMember(t *testing.T) {
	f := newResolveFixture(t, true)
	got, err := f.plugin.resolveSubject(context.Background(), f.stream, caep.SubjectID{
		Members: map[string]caep.SubjectID{
			"user": {Format: caep.FormatIssSub, Issuer: "https://org.okta.com", Subject: "okta-user-1"},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, OutcomeApplied, got.Outcome)
	assert.Equal(t, f.user.ID, got.UserID)
}

func TestResolveSubject_ComplexSubjectWithoutUserMemberIsRejected(t *testing.T) {
	f := newResolveFixture(t, true)
	got, err := f.plugin.resolveSubject(context.Background(), f.stream, caep.SubjectID{
		Members: map[string]caep.SubjectID{
			"tenant": {Format: caep.FormatOpaque, ID: "t1"},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, OutcomeRejected, got.Outcome)
}

func TestResolveSubject_AliasesAgreeing(t *testing.T) {
	f := newResolveFixture(t, true)
	got, err := f.plugin.resolveSubject(context.Background(), f.stream, caep.SubjectID{
		Format: caep.FormatAliases,
		Identifiers: []caep.SubjectID{
			{Format: caep.FormatIssSub, Issuer: "https://org.okta.com", Subject: "okta-user-1"},
			{Format: caep.FormatEmail, Email: "target@corp.com"},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, OutcomeApplied, got.Outcome)
	assert.Equal(t, f.user.ID, got.UserID)
}

// Two aliases naming two different users is a contradiction, and guessing
// which one was meant is how the wrong person gets signed out.
func TestResolveSubject_AliasesDisagreeingRejected(t *testing.T) {
	ctx := context.Background()
	f := newResolveFixture(t, true)

	other := &user.User{
		ID: id.NewUserID(), AppID: f.appID, EnvID: f.envID,
		Email: "other@corp.com", EmailVerified: true,
	}
	authStore, ok := f.plugin.authStore.(*memory.Store)
	require.True(t, ok)
	require.NoError(t, authStore.CreateUserWithPrimaryEmail(ctx, other, &user.UserEmail{
		ID: id.NewUserEmailID(), UserID: other.ID, AppID: f.appID, EnvID: f.envID,
		Email: other.Email, Verified: true, Primary: true,
	}))

	got, err := f.plugin.resolveSubject(ctx, f.stream, caep.SubjectID{
		Format: caep.FormatAliases,
		Identifiers: []caep.SubjectID{
			{Format: caep.FormatIssSub, Issuer: "https://org.okta.com", Subject: "okta-user-1"},
			{Format: caep.FormatEmail, Email: "other@corp.com"},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, OutcomeRejected, got.Outcome)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./plugins/sharedsignals/ -run TestResolveSubject -v`
Expected: FAIL, `resolveSubject` is undefined.

- [ ] **Step 3: Write the resolver**

Create `plugins/sharedsignals/subject.go`:

```go
package sharedsignals

import (
	"context"
	"errors"
	"strings"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/sharedsignals/caep"
)

// Resolution is the outcome of turning a subject identifier into one of our
// users. Outcome is OutcomeApplied when UserID is set, OutcomeUnresolved when
// the identifier was acceptable but named nobody we know, and OutcomeRejected
// when the identifier was not one this stream may assert.
type Resolution struct {
	UserID    id.UserID
	SessionID id.SessionID
	Outcome   string
}

func rejected() Resolution   { return Resolution{Outcome: OutcomeRejected} }
func unresolved() Resolution { return Resolution{Outcome: OutcomeUnresolved} }

func allowsFormat(s *InboundStream, format string) bool {
	for _, f := range s.AllowedSubjectFormats {
		if f == format {
			return true
		}
	}
	return false
}

func domainAllowed(s *InboundStream, email string) bool {
	at := strings.LastIndex(email, "@")
	if at < 0 || at == len(email)-1 {
		return false
	}
	domain := strings.ToLower(email[at+1:])
	for _, d := range s.VerifiedDomains {
		if strings.ToLower(d) == domain {
			return true
		}
	}
	return false
}

// resolveSubject maps a subject identifier onto an authsome user, scoped to
// the stream's app and environment. Nothing here trusts the identifier's own
// claim about who may assert it.
func (p *Plugin) resolveSubject(ctx context.Context, s *InboundStream,
	subj caep.SubjectID) (Resolution, error) {
	// A complex subject describes one principal through named members. The
	// user member is the identity; the session member narrows the action.
	if subj.IsComplex() {
		userMember, ok := subj.Member("user")
		if !ok {
			return rejected(), nil
		}
		res, err := p.resolveSimpleSubject(ctx, s, userMember)
		if err != nil || res.Outcome != OutcomeApplied {
			return res, err
		}
		if sessionMember, ok := subj.Member("session"); ok {
			if sessionID, perr := id.ParseSessionID(sessionMember.ID); perr == nil {
				res.SessionID = sessionID
			}
		}
		return res, nil
	}

	if subj.Format == caep.FormatAliases {
		return p.resolveAliases(ctx, s, subj)
	}

	return p.resolveSimpleSubject(ctx, s, subj)
}

// resolveAliases requires every resolvable member to name the same user.
// Members that resolve to nobody are ignored; members that contradict each
// other kill the event.
func (p *Plugin) resolveAliases(ctx context.Context, s *InboundStream,
	subj caep.SubjectID) (Resolution, error) {
	if !allowsFormat(s, caep.FormatAliases) {
		return rejected(), nil
	}

	var found Resolution
	for _, alias := range subj.Identifiers {
		res, err := p.resolveSimpleSubject(ctx, s, alias)
		if err != nil {
			return Resolution{}, err
		}
		if res.Outcome != OutcomeApplied {
			continue
		}
		if found.Outcome != OutcomeApplied {
			found = res
			continue
		}
		if found.UserID != res.UserID {
			return rejected(), nil
		}
	}
	if found.Outcome != OutcomeApplied {
		return unresolved(), nil
	}
	return found, nil
}

func (p *Plugin) resolveSimpleSubject(ctx context.Context, s *InboundStream,
	subj caep.SubjectID) (Resolution, error) {
	if !allowsFormat(s, subj.Format) {
		return rejected(), nil
	}

	switch subj.Format {
	case caep.FormatIssSub:
		// An identity provider speaks only for subjects it issued.
		if subj.Issuer != s.Issuer {
			return rejected(), nil
		}
		return p.resolveViaLink(ctx, s, subj.Subject)

	case caep.FormatOpaque:
		return p.resolveViaLink(ctx, s, subj.ID)

	case caep.FormatEmail:
		if !domainAllowed(s, subj.Email) {
			return rejected(), nil
		}
		u, err := p.authStore.GetUserByAnyEmail(ctx, s.AppID, s.EnvID, subj.Email)
		if err != nil {
			return unresolved(), nil //nolint:nilerr // a miss is not a failure
		}
		// An unverified address proves nothing about who owns it.
		record, err := p.authStore.GetUserEmailRecord(ctx, s.AppID, s.EnvID, subj.Email)
		if err != nil || record == nil || !record.Verified {
			return rejected(), nil //nolint:nilerr // refuse rather than guess
		}
		return Resolution{UserID: u.ID, Outcome: OutcomeApplied}, nil

	case caep.FormatPhoneNumber:
		u, err := p.authStore.GetUserByPhone(ctx, s.AppID, subj.PhoneNumber)
		if err != nil {
			return unresolved(), nil //nolint:nilerr // a miss is not a failure
		}
		if !u.PhoneVerified {
			return rejected(), nil
		}
		return Resolution{UserID: u.ID, Outcome: OutcomeApplied}, nil

	default:
		// account, uri, did and anything we do not recognise.
		return rejected(), nil
	}
}

func (p *Plugin) resolveViaLink(ctx context.Context, s *InboundStream,
	subject string) (Resolution, error) {
	if subject == "" {
		return rejected(), nil
	}
	link, err := p.store.GetSubjectLink(ctx, s.AppID, s.EnvID, s.Issuer, subject)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return unresolved(), nil
		}
		return Resolution{}, err
	}
	return Resolution{UserID: link.UserID, Outcome: OutcomeApplied}, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./plugins/sharedsignals/ -run TestResolveSubject -v`
Expected: PASS, twelve tests.

If `GetUserEmailRecord` or `user.UserEmail` field names differ, run
`go doc github.com/xraph/authsome/user.Store` and adjust the calls; the
verified check itself must stay.

- [ ] **Step 5: Commit**

```bash
git add plugins/sharedsignals/subject.go plugins/sharedsignals/subject_test.go
git commit -m "feat(sharedsignals): subject identifier resolution with domain and verification gating"
```

---

### Task 12: Subject link stamping

**Files:**
- Create: `plugins/sharedsignals/links.go`
- Modify: `plugins/sso/plugin.go` (after a successful OIDC sign-in)
- Test: `plugins/sharedsignals/links_test.go`

**Interfaces:**
- Consumes: `Store` and `SubjectLink` from Task 6, `Plugin` from Task 10.
- Produces: `(*Plugin).LinkSubject(ctx context.Context, appID id.AppID, envID id.EnvironmentID, issuer, subject string, userID id.UserID, source string) error`; the exported `SubjectLinker` interface other plugins assert against.

Without links, `iss_sub` resolves to nobody and the receiver is a very well tested no-op. SSO is where they come from, because Okta's `sub` in the SET is the same value Okta puts in the OIDC id_token.

- [ ] **Step 1: Write the failing test**

Create `plugins/sharedsignals/links_test.go`:

```go
package sharedsignals

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
)

func TestLinkSubject_CreatesAndResolves(t *testing.T) {
	ctx := context.Background()
	p := New()
	p.store = NewMemoryStore()
	appID, envID := id.NewAppID(), id.NewEnvironmentID()
	userID := id.NewUserID()

	require.NoError(t, p.LinkSubject(ctx, appID, envID,
		"https://org.okta.com", "okta-user-1", userID, SourceSSO))

	got, err := p.store.GetSubjectLink(ctx, appID, envID, "https://org.okta.com", "okta-user-1")
	require.NoError(t, err)
	assert.Equal(t, userID, got.UserID)
	assert.Equal(t, SourceSSO, got.Source)
}

// Signing in twice must refresh the link rather than pile up rows.
func TestLinkSubject_IsIdempotent(t *testing.T) {
	ctx := context.Background()
	p := New()
	p.store = NewMemoryStore()
	appID, envID := id.NewAppID(), id.NewEnvironmentID()
	userID := id.NewUserID()

	for i := 0; i < 3; i++ {
		require.NoError(t, p.LinkSubject(ctx, appID, envID,
			"https://i", "u1", userID, SourceSSO))
	}
	got, err := p.store.GetSubjectLink(ctx, appID, envID, "https://i", "u1")
	require.NoError(t, err)
	assert.Equal(t, userID, got.UserID)
}

func TestLinkSubject_RejectsEmptyArguments(t *testing.T) {
	ctx := context.Background()
	p := New()
	p.store = NewMemoryStore()
	appID, envID := id.NewAppID(), id.NewEnvironmentID()

	require.Error(t, p.LinkSubject(ctx, appID, envID, "", "u1", id.NewUserID(), SourceSSO))
	require.Error(t, p.LinkSubject(ctx, appID, envID, "https://i", "", id.NewUserID(), SourceSSO))
	require.Error(t, p.LinkSubject(ctx, appID, envID, "https://i", "u1", id.Nil, SourceSSO))
}

// The interface is what sso asserts against, so it has to be satisfied by the
// concrete plugin or the wiring silently does nothing.
func TestPlugin_IsSubjectLinker(t *testing.T) {
	var _ SubjectLinker = New()
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./plugins/sharedsignals/ -run 'LinkSubject|SubjectLinker' -v`
Expected: FAIL, `LinkSubject` is undefined.

- [ ] **Step 3: Write the linker**

Create `plugins/sharedsignals/links.go`:

```go
package sharedsignals

import (
	"context"
	"errors"

	"github.com/xraph/authsome/id"
)

// SubjectLinker is implemented by this plugin so other plugins can record the
// upstream identity they just authenticated without importing the concrete
// type. Callers reach it through engine.Plugin("sharedsignals") and a type
// assertion, the same way risk contributors are wired.
type SubjectLinker interface {
	LinkSubject(ctx context.Context, appID id.AppID, envID id.EnvironmentID,
		issuer, subject string, userID id.UserID, source string) error
}

var _ SubjectLinker = (*Plugin)(nil)

// LinkSubject records that (issuer, subject) is this user, so a later CAEP
// event naming that pair resolves. Calling it repeatedly is safe: the store
// upserts on the tuple.
func (p *Plugin) LinkSubject(ctx context.Context, appID id.AppID, envID id.EnvironmentID,
	issuer, subject string, userID id.UserID, source string) error {
	if issuer == "" {
		return errors.New("sharedsignals: link subject: issuer is required")
	}
	if subject == "" {
		return errors.New("sharedsignals: link subject: subject is required")
	}
	if userID.IsNil() {
		return errors.New("sharedsignals: link subject: user is required")
	}
	if source == "" {
		source = SourceManual
	}
	if p.store == nil {
		return errors.New("sharedsignals: link subject: no store configured")
	}

	return p.store.UpsertSubjectLink(ctx, &SubjectLink{
		ID:      id.NewSSFLinkID(),
		AppID:   appID,
		EnvID:   envID,
		Issuer:  issuer,
		Subject: subject,
		UserID:  userID,
		Source:  source,
	})
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./plugins/sharedsignals/ -run 'LinkSubject|SubjectLinker' -v`
Expected: PASS, four tests.

- [ ] **Step 5: Wire sso to call it**

Find where `plugins/sso/plugin.go` completes an OIDC sign-in and has both the
resolved user and the id_token `sub` in scope. Search for the call that
creates the session after `handleOIDCRedirect` or `handleCallback` resolves a
user:

```bash
grep -n "func (p \*Plugin) handleCallback\|func (p \*Plugin) handleOIDCRedirect" -A 60 plugins/sso/plugin.go
```

Add this helper to `plugins/sso/plugin.go`:

```go
// subjectLinker is the slice of the sharedsignals plugin sso needs. Declared
// here rather than imported so sso does not depend on that plugin being
// compiled in.
type subjectLinker interface {
	LinkSubject(ctx context.Context, appID id.AppID, envID id.EnvironmentID,
		issuer, subject string, userID id.UserID, source string) error
}

// linkSharedSignalsSubject records the IdP subject we just authenticated so a
// later CAEP event naming (issuer, sub) can find this user. A no-op when the
// sharedsignals plugin is not registered, and never fatal to a sign-in: a
// failure here costs a future signal, not this login.
func (p *Plugin) linkSharedSignalsSubject(ctx context.Context, u *user.User,
	issuer, subject string) {
	if p.engine == nil || issuer == "" || subject == "" {
		return
	}
	target := p.engine.Plugin("sharedsignals")
	if target == nil {
		return
	}
	linker, ok := target.(subjectLinker)
	if !ok {
		return
	}
	if err := linker.LinkSubject(ctx, u.AppID, u.EnvID, issuer, subject, u.ID, "sso"); err != nil {
		p.logger.Warn("sso: record shared signals subject link",
			log.String("issuer", issuer),
			log.String("error", err.Error()),
		)
	}
}
```

Then call it immediately after the user is resolved on the OIDC path, passing
the connection's `Issuer` and the id_token's `sub`.

- [ ] **Step 6: Test the sso wiring**

Create `plugins/sso/sharedsignals_link_test.go`:

```go
package sso

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/user"
)

type recordingLinker struct {
	calls   int
	issuer  string
	subject string
	userID  id.UserID
	err     error
}

func (r *recordingLinker) Name() string { return "sharedsignals" }

func (r *recordingLinker) LinkSubject(_ context.Context, _ id.AppID, _ id.EnvironmentID,
	issuer, subject string, userID id.UserID, _ string) error {
	r.calls++
	r.issuer, r.subject, r.userID = issuer, subject, userID
	return r.err
}

func TestLinkSharedSignalsSubject_SkipsWhenPluginAbsent(t *testing.T) {
	p := &Plugin{}
	// No engine means no plugin registry; this must not panic.
	p.linkSharedSignalsSubject(context.Background(),
		&user.User{ID: id.NewUserID()}, "https://i", "u1")
}

func TestLinkSharedSignalsSubject_SkipsEmptyValues(t *testing.T) {
	r := &recordingLinker{}
	p := newSSOPluginWithPlugin(t, r)
	u := &user.User{ID: id.NewUserID(), AppID: id.NewAppID(), EnvID: id.NewEnvironmentID()}

	p.linkSharedSignalsSubject(context.Background(), u, "", "u1")
	p.linkSharedSignalsSubject(context.Background(), u, "https://i", "")
	assert.Zero(t, r.calls)
}

func TestLinkSharedSignalsSubject_RecordsLink(t *testing.T) {
	r := &recordingLinker{}
	p := newSSOPluginWithPlugin(t, r)
	u := &user.User{ID: id.NewUserID(), AppID: id.NewAppID(), EnvID: id.NewEnvironmentID()}

	p.linkSharedSignalsSubject(context.Background(), u, "https://org.okta.com", "okta-user-1")
	assert.Equal(t, 1, r.calls)
	assert.Equal(t, "https://org.okta.com", r.issuer)
	assert.Equal(t, "okta-user-1", r.subject)
	assert.Equal(t, u.ID, r.userID)
}
```

Build `newSSOPluginWithPlugin` in that file using the same engine test double
`plugins/sso` already uses in `routes_test.go`. Read that file first:

```bash
grep -n "func new.*Plugin\|plugin.Engine" plugins/sso/routes_test.go | head
```

If no double exists there, add one whose `Plugin(name string)` returns the
recorder when `name == "sharedsignals"` and nil otherwise, and whose
`Logger()` returns `log.NewNoopLogger()`.

- [ ] **Step 7: Run both suites**

Run: `go test ./plugins/sharedsignals/ ./plugins/sso/ -v`
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add plugins/sharedsignals/links.go plugins/sharedsignals/links_test.go plugins/sso/plugin.go plugins/sso/sharedsignals_link_test.go
git commit -m "feat(sharedsignals): record IdP subject links on SSO sign-in"
```

---

### Task 13: The action matrix and the circuit breaker

**Files:**
- Create: `plugins/sharedsignals/actions.go`
- Test: `plugins/sharedsignals/actions_test.go`

**Interfaces:**
- Consumes: `caep.Event` from Task 2, `Resolution` from Task 11, `Store` from Task 6, `plugin.SessionRevoker` from Task 5.
- Produces: action constants `ActionRevokeAll`, `ActionRevokeSession`, `ActionSignal`, `ActionLog`, `ActionNone`; `(*Plugin).actionFor(s *InboundStream, ev caep.Event) string`; `(*Plugin).severityFor(ev caep.Event) int`; `(*Plugin).applyEvent(ctx context.Context, s *InboundStream, ev caep.Event, res Resolution) (actionTaken string, err error)`; `(*Plugin).checkCircuitBreaker(ctx context.Context, s *InboundStream) (bool, error)`.

- [ ] **Step 1: Write the failing test**

Create `plugins/sharedsignals/actions_test.go`:

```go
package sharedsignals

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/sharedsignals/caep"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/store/memory"
)

type recordingRevoker struct {
	revoked []id.SessionID
	err     error
}

func (r *recordingRevoker) RevokeSession(_ context.Context, sessionID id.SessionID) error {
	if r.err != nil {
		return r.err
	}
	r.revoked = append(r.revoked, sessionID)
	return nil
}

type actionFixture struct {
	plugin   *Plugin
	stream   *InboundStream
	revoker  *recordingRevoker
	userID   id.UserID
	sessions []id.SessionID
}

func newActionFixture(t *testing.T) actionFixture {
	t.Helper()
	ctx := context.Background()
	appID, envID := id.NewAppID(), id.NewEnvironmentID()
	userID := id.NewUserID()

	authStore := memory.New()
	var sessions []id.SessionID
	for i := 0; i < 3; i++ {
		s := &session.Session{
			ID: id.NewSessionID(), AppID: appID, EnvID: envID, UserID: userID,
			Token: "tok-" + string(rune('a'+i)), ExpiresAt: time.Now().Add(time.Hour),
		}
		require.NoError(t, authStore.CreateSession(ctx, s))
		sessions = append(sessions, s.ID)
	}

	rev := &recordingRevoker{}
	p := New()
	p.store = NewMemoryStore()
	p.authStore = authStore
	p.revoker = rev

	stream := &InboundStream{
		ID: id.NewSSFStreamID(), AppID: appID, EnvID: envID,
		Issuer: "https://org.okta.com", Status: StatusEnabled,
		EnforcementMode: EnforcementEnforce, MaxActionsPerHour: 100,
	}
	require.NoError(t, p.store.CreateInboundStream(ctx, stream))

	return actionFixture{plugin: p, stream: stream, revoker: rev,
		userID: userID, sessions: sessions}
}

func TestActionFor_Defaults(t *testing.T) {
	p := New()
	s := &InboundStream{}
	cases := []struct {
		ev   caep.Event
		want string
	}{
		{caep.Event{Type: caep.EventSessionRevoked}, ActionRevokeAll},
		{caep.Event{Type: caep.EventTokenClaimsChange}, ActionRevokeAll},
		{caep.Event{Type: caep.EventCredentialChange, ChangeType: "revoke"}, ActionRevokeAll},
		{caep.Event{Type: caep.EventCredentialChange, ChangeType: "delete"}, ActionRevokeAll},
		{caep.Event{Type: caep.EventCredentialChange, ChangeType: "create"}, ActionSignal},
		{caep.Event{Type: caep.EventAssuranceLevelChange, ChangeDirection: "decrease"}, ActionSignal},
		{caep.Event{Type: caep.EventAssuranceLevelChange, ChangeDirection: "increase"}, ActionSignal},
		{caep.Event{Type: caep.EventDeviceComplianceChange, CurrentStatus: "not-compliant"}, ActionSignal},
		{caep.Event{Type: caep.EventRiskLevelChange, CurrentLevel: "HIGH"}, ActionSignal},
		{caep.Event{Type: caep.EventVerification}, ActionNone},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, p.actionFor(s, tc.ev), "event %s", tc.ev.Type)
	}
}

func TestActionFor_StreamOverrideWins(t *testing.T) {
	p := New()
	s := &InboundStream{ActionOverrides: map[string]string{
		caep.EventSessionRevoked: ActionSignal,
	}}
	assert.Equal(t, ActionSignal, p.actionFor(s, caep.Event{Type: caep.EventSessionRevoked}))
}

func TestSeverityFor(t *testing.T) {
	p := New()
	assert.Equal(t, 100, p.severityFor(caep.Event{Type: caep.EventSessionRevoked}))
	assert.Equal(t, 90, p.severityFor(caep.Event{
		Type: caep.EventCredentialChange, ChangeType: "revoke"}))
	assert.Equal(t, 20, p.severityFor(caep.Event{
		Type: caep.EventCredentialChange, ChangeType: "create"}))
	assert.Equal(t, 70, p.severityFor(caep.Event{
		Type: caep.EventAssuranceLevelChange, ChangeDirection: "decrease"}))
	assert.Equal(t, 10, p.severityFor(caep.Event{
		Type: caep.EventAssuranceLevelChange, ChangeDirection: "increase"}))
	assert.Equal(t, 60, p.severityFor(caep.Event{
		Type: caep.EventDeviceComplianceChange, CurrentStatus: "not-compliant"}))
	assert.Equal(t, 10, p.severityFor(caep.Event{
		Type: caep.EventDeviceComplianceChange, CurrentStatus: "compliant"}))
	assert.Equal(t, 80, p.severityFor(caep.Event{
		Type: caep.EventRiskLevelChange, CurrentLevel: "HIGH"}))
}

func TestApplyEvent_SessionRevokedRevokesEverySession(t *testing.T) {
	f := newActionFixture(t)
	action, err := f.plugin.applyEvent(context.Background(), f.stream,
		caep.Event{Type: caep.EventSessionRevoked},
		Resolution{UserID: f.userID, Outcome: OutcomeApplied})
	require.NoError(t, err)
	assert.Equal(t, ActionRevokeAll, action)
	assert.Len(t, f.revoker.revoked, 3)
}

// A session member in a complex subject narrows the blast radius to one
// session instead of signing the user out everywhere.
func TestApplyEvent_TargetedSessionRevoke(t *testing.T) {
	f := newActionFixture(t)
	action, err := f.plugin.applyEvent(context.Background(), f.stream,
		caep.Event{Type: caep.EventSessionRevoked},
		Resolution{UserID: f.userID, SessionID: f.sessions[1], Outcome: OutcomeApplied})
	require.NoError(t, err)
	assert.Equal(t, ActionRevokeSession, action)
	require.Len(t, f.revoker.revoked, 1)
	assert.Equal(t, f.sessions[1], f.revoker.revoked[0])
}

// Observe mode must record the signal and skip the revocation, so an operator
// can watch a new stream before trusting it.
func TestApplyEvent_ObserveModeDoesNotRevoke(t *testing.T) {
	f := newActionFixture(t)
	f.stream.EnforcementMode = EnforcementObserve

	action, err := f.plugin.applyEvent(context.Background(), f.stream,
		caep.Event{Type: caep.EventSessionRevoked},
		Resolution{UserID: f.userID, Outcome: OutcomeApplied})
	require.NoError(t, err)
	assert.Equal(t, ActionLog, action)
	assert.Empty(t, f.revoker.revoked)

	signals, err := f.plugin.store.ListActiveSignals(context.Background(),
		f.stream.AppID, f.userID, time.Now())
	require.NoError(t, err)
	require.Len(t, signals, 1, "observe mode still records the signal")
}

func TestApplyEvent_AlwaysWritesASignal(t *testing.T) {
	f := newActionFixture(t)
	_, err := f.plugin.applyEvent(context.Background(), f.stream,
		caep.Event{Type: caep.EventDeviceComplianceChange, CurrentStatus: "not-compliant"},
		Resolution{UserID: f.userID, Outcome: OutcomeApplied})
	require.NoError(t, err)

	signals, err := f.plugin.store.ListActiveSignals(context.Background(),
		f.stream.AppID, f.userID, time.Now())
	require.NoError(t, err)
	require.Len(t, signals, 1)
	assert.Equal(t, 60, signals[0].Severity)
	assert.Empty(t, f.revoker.revoked)
}

func TestApplyEvent_VerificationTakesNoUserAction(t *testing.T) {
	f := newActionFixture(t)
	action, err := f.plugin.applyEvent(context.Background(), f.stream,
		caep.Event{Type: caep.EventVerification, State: "abc"},
		Resolution{Outcome: OutcomeApplied})
	require.NoError(t, err)
	assert.Equal(t, "", action)
	assert.Empty(t, f.revoker.revoked)
}

func TestCircuitBreaker_TripsAndPausesStream(t *testing.T) {
	ctx := context.Background()
	f := newActionFixture(t)
	f.stream.MaxActionsPerHour = 2
	require.NoError(t, f.plugin.store.UpdateInboundStream(ctx, f.stream))

	now := time.Now()
	for i := 0; i < 2; i++ {
		require.NoError(t, f.plugin.store.InsertReceivedEvent(ctx, &ReceivedEvent{
			ID: id.NewSSFEventID(), StreamID: f.stream.ID,
			JTI: "prior-" + string(rune('a'+i)), EventType: caep.EventSessionRevoked,
			Outcome: OutcomeApplied, ActionTaken: ActionRevokeAll, ReceivedAt: now,
		}))
	}

	ok, err := f.plugin.checkCircuitBreaker(ctx, f.stream)
	require.NoError(t, err)
	assert.False(t, ok, "the breaker must trip at the limit")

	after, err := f.plugin.store.GetInboundStream(ctx, f.stream.ID)
	require.NoError(t, err)
	assert.Equal(t, StatusPaused, after.Status,
		"a tripped breaker pauses the stream so it stops acting until a human looks")
}

func TestCircuitBreaker_AllowsUnderLimit(t *testing.T) {
	f := newActionFixture(t)
	ok, err := f.plugin.checkCircuitBreaker(context.Background(), f.stream)
	require.NoError(t, err)
	assert.True(t, ok)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./plugins/sharedsignals/ -run 'Action|Severity|ApplyEvent|CircuitBreaker' -v`
Expected: FAIL, the action helpers are undefined.

- [ ] **Step 3: Write the action matrix**

Create `plugins/sharedsignals/actions.go`:

```go
package sharedsignals

import (
	"context"
	"fmt"
	"time"

	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/sharedsignals/caep"
)

// Actions an event can produce.
const (
	ActionRevokeAll     = "revoke_all"
	ActionRevokeSession = "revoke_session"
	ActionSignal        = "signal"
	ActionLog           = "log"
	ActionNone          = "none"
)

// actionFor decides what an event does. A stream override always wins over
// the default, so an operator can quiet a noisy transmitter per event type
// without turning the whole stream off.
func (p *Plugin) actionFor(s *InboundStream, ev caep.Event) string {
	if override, ok := s.ActionOverrides[ev.Type]; ok && override != "" {
		return override
	}

	switch ev.Type {
	case caep.EventSessionRevoked:
		return ActionRevokeAll

	case caep.EventTokenClaimsChange:
		// session.Roles is stamped at issue time and never re-resolved, so a
		// claims change cannot reach a live session. Ending it is the only
		// way to pick up the new claims.
		return ActionRevokeAll

	case caep.EventCredentialChange:
		if ev.ChangeType == "revoke" || ev.ChangeType == "delete" {
			return ActionRevokeAll
		}
		return ActionSignal

	case caep.EventAssuranceLevelChange,
		caep.EventDeviceComplianceChange,
		caep.EventRiskLevelChange:
		return ActionSignal

	case caep.EventVerification:
		return ActionNone

	default:
		return ActionSignal
	}
}

// severityFor scores an event 0 to 100 for the risk engine.
func (p *Plugin) severityFor(ev caep.Event) int {
	switch ev.Type {
	case caep.EventSessionRevoked:
		return 100

	case caep.EventTokenClaimsChange:
		return 50

	case caep.EventCredentialChange:
		if ev.ChangeType == "revoke" || ev.ChangeType == "delete" {
			return 90
		}
		return 20

	case caep.EventAssuranceLevelChange:
		if ev.ChangeDirection == "decrease" {
			return 70
		}
		return 10

	case caep.EventDeviceComplianceChange:
		if ev.CurrentStatus == "not-compliant" {
			return 60
		}
		return 10

	case caep.EventRiskLevelChange:
		switch ev.CurrentLevel {
		case "HIGH":
			return 80
		case "MEDIUM":
			return 50
		default:
			return 10
		}

	default:
		return 30
	}
}

// applyEvent runs the matrix for one resolved event and returns the action it
// actually took. A signal is always recorded first, so even an action that is
// skipped or fails still leaves the next sign-in something to score.
func (p *Plugin) applyEvent(ctx context.Context, s *InboundStream,
	ev caep.Event, res Resolution) (string, error) {
	if ev.Type == caep.EventVerification {
		return "", p.completeVerification(ctx, s, ev)
	}

	action := p.actionFor(s, ev)

	if err := p.recordSignal(ctx, s, ev, res); err != nil {
		return "", err
	}

	if action == ActionSignal || action == ActionNone || action == ActionLog {
		return "", nil
	}

	// Observe mode records everything and changes nothing.
	if s.EnforcementMode == EnforcementObserve {
		p.audit(ctx, s, ev, res, bridge.SeverityWarning, "would_"+action)
		return ActionLog, nil
	}

	if p.revoker == nil {
		return "", fmt.Errorf("sharedsignals: cannot revoke, engine does not support it")
	}

	// A session member in the subject narrows this to one session.
	if !res.SessionID.IsNil() && action == ActionRevokeAll {
		action = ActionRevokeSession
	}

	switch action {
	case ActionRevokeSession:
		if err := p.revoker.RevokeSession(ctx, res.SessionID); err != nil {
			return "", err
		}
	case ActionRevokeAll:
		sessions, err := p.authStore.ListUserSessions(ctx, res.UserID)
		if err != nil {
			return "", err
		}
		for _, sess := range sessions {
			// Stay inside this stream's app and environment. A stream never
			// reaches a session it was not scoped to.
			if sess.AppID != s.AppID {
				continue
			}
			if err := p.revoker.RevokeSession(ctx, sess.ID); err != nil {
				return "", err
			}
		}
	}

	p.audit(ctx, s, ev, res, bridge.SeverityCritical, action)
	return action, nil
}

func (p *Plugin) recordSignal(ctx context.Context, s *InboundStream,
	ev caep.Event, res Resolution) error {
	now := time.Now()
	eventAt := now
	if ev.EventTimestamp > 0 {
		// CAEP timestamps are seconds in the spec but Okta sends
		// milliseconds, so treat anything implausibly large as millis.
		if ev.EventTimestamp > 1e11 {
			eventAt = time.UnixMilli(ev.EventTimestamp)
		} else {
			eventAt = time.Unix(ev.EventTimestamp, 0)
		}
	}

	reason := ""
	if ev.ReasonAdmin != nil {
		reason = ev.ReasonAdmin["en"]
	}

	return p.store.CreateSignal(ctx, &Signal{
		ID:        id.NewSSFSignalID(),
		AppID:     s.AppID,
		EnvID:     s.EnvID,
		UserID:    res.UserID,
		StreamID:  s.ID,
		EventType: ev.Type,
		Severity:  p.severityFor(ev),
		Reason:    reason,
		EventAt:   eventAt,
		ExpiresAt: now.Add(p.config.SignalTTL),
		CreatedAt: now,
	})
}

// completeVerification matches the echoed state against what we sent. A
// mismatch is not an error to the transmitter, it just does not mark the
// stream verified.
func (p *Plugin) completeVerification(ctx context.Context, s *InboundStream,
	ev caep.Event) error {
	if s.PendingVerifyState == "" || ev.State != s.PendingVerifyState {
		return nil
	}
	now := time.Now()
	s.LastVerifiedAt = &now
	s.PendingVerifyState = ""
	return p.store.UpdateInboundStream(ctx, s)
}

// checkCircuitBreaker reports whether the stream may still act. Crossing the
// limit pauses the stream and raises an alert, because a transmitter asking
// for thousands of revocations is either compromised or misconfigured and
// both want the same answer.
func (p *Plugin) checkCircuitBreaker(ctx context.Context, s *InboundStream) (bool, error) {
	limit := s.MaxActionsPerHour
	if limit <= 0 {
		limit = p.config.MaxActionsPerHour
	}

	count, err := p.store.CountActionsSince(ctx, s.ID, time.Now().Add(-time.Hour))
	if err != nil {
		return false, err
	}
	if count < limit {
		return true, nil
	}

	s.Status = StatusPaused
	if err := p.store.UpdateInboundStream(ctx, s); err != nil {
		return false, err
	}

	if p.chronicle != nil {
		_ = p.chronicle.Record(ctx, &bridge.AuditEvent{ //nolint:errcheck // best-effort audit
			Action:   "ssf_circuit_breaker_tripped",
			Resource: "sharedsignals_stream",
			Tenant:   s.AppID.String(),
			Outcome:  bridge.OutcomeFailure,
			Severity: bridge.SeverityCritical,
			Metadata: map[string]string{
				"stream_id": s.ID.String(),
				"issuer":    s.Issuer,
				"limit":     fmt.Sprintf("%d", limit),
				"count":     fmt.Sprintf("%d", count),
			},
		})
	}
	if p.relay != nil {
		_ = p.relay.Send(ctx, &bridge.WebhookEvent{ //nolint:errcheck // best-effort webhook
			Type:     "security.ssf.circuit_breaker_tripped",
			TenantID: s.AppID.String(),
			Data: map[string]string{
				"stream_id": s.ID.String(),
				"issuer":    s.Issuer,
			},
		})
	}

	p.logger.Warn("sharedsignals: circuit breaker tripped, stream paused",
		logString("stream_id", s.ID.String()),
		logString("issuer", s.Issuer),
	)
	return false, nil
}

func (p *Plugin) audit(ctx context.Context, s *InboundStream, ev caep.Event,
	res Resolution, severity bridge.Severity, action string) {
	if p.chronicle == nil {
		return
	}
	_ = p.chronicle.Record(ctx, &bridge.AuditEvent{ //nolint:errcheck // best-effort audit
		Action:   "ssf_event_applied",
		Resource: "session",
		ActorID:  res.UserID.String(),
		Tenant:   s.AppID.String(),
		Outcome:  bridge.OutcomeSuccess,
		Severity: severity,
		Metadata: map[string]string{
			"stream_id":  s.ID.String(),
			"issuer":     s.Issuer,
			"event_type": ev.Type,
			"action":     action,
		},
	})
}
```

Add this helper to `plugins/sharedsignals/plugin.go` so the logging calls read
the same everywhere:

```go
// logString is a tiny alias so call sites do not repeat the import path of
// the logging package in every field.
func logString(key, value string) log.Field { return log.String(key, value) }
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./plugins/sharedsignals/ -run 'Action|Severity|ApplyEvent|CircuitBreaker' -v`
Expected: PASS, eleven tests.

If `bridge.Severity` or `log.Field` do not resolve, check the real names with
`go doc github.com/xraph/authsome/bridge.AuditEvent` and
`go doc github.com/xraph/go-utils/log.String`.

- [ ] **Step 5: Commit**

```bash
git add plugins/sharedsignals/actions.go plugins/sharedsignals/actions_test.go plugins/sharedsignals/plugin.go
git commit -m "feat(sharedsignals): action matrix, signal recording and the blast-radius breaker"
```

---

### Task 14: The push endpoint

**Files:**
- Create: `plugins/sharedsignals/receiver.go`
- Modify: `plugins/sharedsignals/plugin.go` (delete the `registerReceiverRoutes` stub from Task 10)
- Test: `plugins/sharedsignals/receiver_test.go`

**Interfaces:**
- Consumes: everything from Tasks 1, 2, 3, 4, 6, 10, 11 and 13.
- Produces: `(*Plugin).registerReceiverRoutes(router forge.Router) error`; `(*Plugin).handlePush(ctx forge.Context) error`; `HashSecret(raw string) string`; `NewPushSecret() (raw, hash string, err error)`.

This is the endpoint an external party talks to, so the gates run cheapest and least trusting first. The test file is the specification for the ordering.

- [ ] **Step 1: Write the failing test**

Create `plugins/sharedsignals/receiver_test.go`:

```go
package sharedsignals

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
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/sharedsignals/caep"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/store/memory"
)

type receiverFixture struct {
	plugin    *Plugin
	stream    *InboundStream
	revoker   *recordingRevoker
	key       *rsa.PrivateKey
	jwksSrv   *httptest.Server
	pushPath  string
	pushToken string
	userID    id.UserID
}

const (
	fixtureIssuer = "https://org.okta.com"
	fixtureAud    = "https://authsome.example/ssf"
	fixtureKID    = "kid-1"
	fixtureSub    = "okta-user-1"
)

func newReceiverFixture(t *testing.T) *receiverFixture {
	t.Helper()
	ctx := context.Background()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// A fake transmitter serving its own JWKS, which is the only way to test
	// the verification path the way it actually runs.
	jwksSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := base64.RawURLEncoding.EncodeToString(key.PublicKey.N.Bytes())
		e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.PublicKey.E)).Bytes())
		fmt.Fprintf(w, `{"keys":[{"kty":"RSA","use":"sig","alg":"RS256","kid":%q,"n":%q,"e":%q}]}`,
			fixtureKID, n, e)
	}))
	t.Cleanup(jwksSrv.Close)

	appID, envID := id.NewAppID(), id.NewEnvironmentID()
	userID := id.NewUserID()

	authStore := memory.New()
	require.NoError(t, authStore.CreateSession(ctx, &session.Session{
		ID: id.NewSessionID(), AppID: appID, EnvID: envID, UserID: userID,
		Token: "tok-1", ExpiresAt: time.Now().Add(time.Hour),
	}))

	rev := &recordingRevoker{}
	p := New(Config{Audience: fixtureAud})
	p.store = NewMemoryStore()
	p.authStore = authStore
	p.revoker = rev
	require.NoError(t, p.OnInit(ctx, stubEngine{}))
	// OnInit installs a memory store and a jwks client; keep ours.
	p.store = NewMemoryStore()
	p.authStore = authStore
	p.revoker = rev

	pushPath, pushPathHash, err := NewPushSecret()
	require.NoError(t, err)
	pushToken, pushTokenHash, err := NewPushSecret()
	require.NoError(t, err)

	stream := &InboundStream{
		ID: id.NewSSFStreamID(), AppID: appID, EnvID: envID, Name: "okta",
		Issuer: fixtureIssuer, Audience: fixtureAud, JWKSURI: jwksSrv.URL,
		PushPathHash: pushPathHash, PushTokenHash: pushTokenHash,
		AllowedEventTypes: []string{
			caep.EventSessionRevoked, caep.EventCredentialChange, caep.EventVerification,
		},
		AllowedSubjectFormats: []string{caep.FormatIssSub},
		EnforcementMode:       EnforcementEnforce,
		Status:                StatusEnabled,
		MaxActionsPerHour:     100,
	}
	require.NoError(t, p.store.CreateInboundStream(ctx, stream))
	require.NoError(t, p.store.UpsertSubjectLink(ctx, &SubjectLink{
		ID: id.NewSSFLinkID(), AppID: appID, EnvID: envID,
		Issuer: fixtureIssuer, Subject: fixtureSub, UserID: userID, Source: SourceSSO,
	}))

	return &receiverFixture{
		plugin: p, stream: stream, revoker: rev, key: key, jwksSrv: jwksSrv,
		pushPath: pushPath, pushToken: pushToken, userID: userID,
	}
}

// signSET builds a SET the fixture's fake transmitter would send.
func (f *receiverFixture) signSET(t *testing.T, mutate func(jwt.MapClaims)) string {
	t.Helper()
	claims := jwt.MapClaims{
		"iss": fixtureIssuer,
		"aud": fixtureAud,
		"jti": "jti-" + id.NewSSFEventID().String(),
		"iat": time.Now().Unix(),
		"events": map[string]any{
			caep.EventSessionRevoked: map[string]any{
				// Okta ships "subject", not "sub_id".
				"subject": map[string]any{
					"format": "iss_sub", "iss": fixtureIssuer, "sub": fixtureSub,
				},
				"reason_admin": map[string]any{"en": "Account compromised"},
			},
		},
	}
	if mutate != nil {
		mutate(claims)
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tok.Header["typ"] = "secevent+jwt"
	tok.Header["kid"] = fixtureKID
	signed, err := tok.SignedString(f.key)
	require.NoError(t, err)
	return signed
}

// post drives the handler through the plugin's own router so the route
// pattern and the parameter binding are exercised, not bypassed.
func (f *receiverFixture) post(t *testing.T, path, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost,
		"/v1/ssf/streams/"+path+"/events", stringReader(body))
	req.Header.Set("Content-Type", "application/secevent+jwt")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	f.plugin.servePushForTest(rec, req, path)
	return rec
}

func errBody(t *testing.T, rec *httptest.ResponseRecorder) map[string]string {
	t.Helper()
	var out map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
	return out
}

func TestPush_ValidSETRevokesSessions(t *testing.T) {
	f := newReceiverFixture(t)
	rec := f.post(t, f.pushPath, f.pushToken, f.signSET(t, nil))

	assert.Equal(t, http.StatusAccepted, rec.Code)
	assert.Empty(t, rec.Body.String(), "a 202 carries no body")
	assert.Len(t, f.revoker.revoked, 1)
}

func TestPush_UnknownPathIs404(t *testing.T) {
	f := newReceiverFixture(t)
	rec := f.post(t, "not-a-real-path", f.pushToken, f.signSET(t, nil))
	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Empty(t, f.revoker.revoked)
}

func TestPush_WrongTokenIs401(t *testing.T) {
	f := newReceiverFixture(t)
	rec := f.post(t, f.pushPath, "wrong-token", f.signSET(t, nil))
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, "authentication_failed", errBody(t, rec)["err"])
	assert.Empty(t, f.revoker.revoked)
}

func TestPush_MissingTokenIs401(t *testing.T) {
	f := newReceiverFixture(t)
	rec := f.post(t, f.pushPath, "", f.signSET(t, nil))
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestPush_PausedStreamIs403(t *testing.T) {
	f := newReceiverFixture(t)
	f.stream.Status = StatusPaused
	require.NoError(t, f.plugin.store.UpdateInboundStream(context.Background(), f.stream))

	rec := f.post(t, f.pushPath, f.pushToken, f.signSET(t, nil))
	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.Equal(t, "access_denied", errBody(t, rec)["err"])
}

func TestPush_WrongIssuerIs400(t *testing.T) {
	f := newReceiverFixture(t)
	body := f.signSET(t, func(c jwt.MapClaims) { c["iss"] = "https://evil.example" })
	rec := f.post(t, f.pushPath, f.pushToken, body)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "invalid_issuer", errBody(t, rec)["err"])
	assert.Empty(t, f.revoker.revoked)
}

func TestPush_WrongAudienceIs400(t *testing.T) {
	f := newReceiverFixture(t)
	body := f.signSET(t, func(c jwt.MapClaims) { c["aud"] = "https://elsewhere" })
	rec := f.post(t, f.pushPath, f.pushToken, body)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "invalid_audience", errBody(t, rec)["err"])
}

func TestPush_UnsignedSETIs400(t *testing.T) {
	f := newReceiverFixture(t)
	tok := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
		"iss": fixtureIssuer, "aud": fixtureAud, "jti": "j", "iat": time.Now().Unix(),
		"events": map[string]any{caep.EventSessionRevoked: map[string]any{
			"subject": map[string]any{"format": "iss_sub", "iss": fixtureIssuer, "sub": fixtureSub},
		}},
	})
	tok.Header["typ"] = "secevent+jwt"
	body, err := tok.SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)

	rec := f.post(t, f.pushPath, f.pushToken, body)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "invalid_key", errBody(t, rec)["err"])
	assert.Empty(t, f.revoker.revoked)
}

// A replayed SET must be accepted and ignored. Answering with an error would
// make the transmitter retry forever.
func TestPush_ReplayedJTIIsAcceptedOnceOnly(t *testing.T) {
	f := newReceiverFixture(t)
	body := f.signSET(t, nil)

	first := f.post(t, f.pushPath, f.pushToken, body)
	assert.Equal(t, http.StatusAccepted, first.Code)
	assert.Len(t, f.revoker.revoked, 1)

	second := f.post(t, f.pushPath, f.pushToken, body)
	assert.Equal(t, http.StatusAccepted, second.Code)
	assert.Len(t, f.revoker.revoked, 1, "a replay must not revoke a second time")
}

// A subject we cannot place returns 202. An error would tell the transmitter
// which of its users have accounts here.
func TestPush_UnknownSubjectIs202AndDoesNothing(t *testing.T) {
	f := newReceiverFixture(t)
	body := f.signSET(t, func(c jwt.MapClaims) {
		c["events"] = map[string]any{
			caep.EventSessionRevoked: map[string]any{
				"subject": map[string]any{
					"format": "iss_sub", "iss": fixtureIssuer, "sub": "nobody-here",
				},
			},
		}
	})

	rec := f.post(t, f.pushPath, f.pushToken, body)
	assert.Equal(t, http.StatusAccepted, rec.Code)
	assert.Empty(t, f.revoker.revoked)
}

// An event type the stream did not subscribe to is recorded and dropped.
func TestPush_EventTypeNotAllowedIsIgnored(t *testing.T) {
	f := newReceiverFixture(t)
	body := f.signSET(t, func(c jwt.MapClaims) {
		c["events"] = map[string]any{
			caep.EventDeviceComplianceChange: map[string]any{
				"subject": map[string]any{
					"format": "iss_sub", "iss": fixtureIssuer, "sub": fixtureSub,
				},
				"current_status": "not-compliant", "previous_status": "compliant",
			},
		}
	})

	rec := f.post(t, f.pushPath, f.pushToken, body)
	assert.Equal(t, http.StatusAccepted, rec.Code)
	assert.Empty(t, f.revoker.revoked)
}

func TestPush_OversizedBodyIs400(t *testing.T) {
	f := newReceiverFixture(t)
	huge := make([]byte, f.plugin.config.MaxBodyBytes+1024)
	for i := range huge {
		huge[i] = 'A'
	}
	rec := f.post(t, f.pushPath, f.pushToken, string(huge))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// The whole point of pinning keys per stream: a genuine SET aimed at one
// tenant must not work against another tenant's URL.
func TestPush_CrossStreamReplayFails(t *testing.T) {
	ctx := context.Background()
	f := newReceiverFixture(t)

	otherPath, otherPathHash, err := NewPushSecret()
	require.NoError(t, err)
	otherToken, otherTokenHash, err := NewPushSecret()
	require.NoError(t, err)
	require.NoError(t, f.plugin.store.CreateInboundStream(ctx, &InboundStream{
		ID: id.NewSSFStreamID(), AppID: id.NewAppID(), EnvID: id.NewEnvironmentID(),
		Issuer: "https://other-idp.example", Audience: fixtureAud,
		JWKSURI:      f.jwksSrv.URL,
		PushPathHash: otherPathHash, PushTokenHash: otherTokenHash,
		AllowedEventTypes:     []string{caep.EventSessionRevoked},
		AllowedSubjectFormats: []string{caep.FormatIssSub},
		EnforcementMode:       EnforcementEnforce, Status: StatusEnabled,
		MaxActionsPerHour:     100,
	}))

	rec := f.post(t, otherPath, otherToken, f.signSET(t, nil))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "invalid_issuer", errBody(t, rec)["err"])
	assert.Empty(t, f.revoker.revoked)
}

func TestHashSecret_IsStableAndNotThePlaintext(t *testing.T) {
	raw, hash, err := NewPushSecret()
	require.NoError(t, err)
	assert.NotEqual(t, raw, hash)
	assert.Equal(t, hash, HashSecret(raw))
	assert.NotEqual(t, hash, HashSecret(raw+"x"))
}
```

Add the small reader helper at the bottom of the same file:

```go
func stringReader(s string) *strings.Reader { return strings.NewReader(s) }
```

and add `"strings"` to its imports.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./plugins/sharedsignals/ -run 'TestPush|TestHashSecret' -v`
Expected: FAIL, `NewPushSecret` and `servePushForTest` are undefined.

- [ ] **Step 3: Write the receiver**

Create `plugins/sharedsignals/receiver.go`:

```go
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

	"github.com/xraph/forge"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/sharedsignals/caep"
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

	// A raw handler: the body is a compact JWS rather than JSON, success is a
	// bodiless 202, and failure is the RFC 8935 error object.
	return g.POST("/streams/:push_path/events", p.handlePush,
		forge.WithSummary("Receive Security Event Tokens"),
		forge.WithOperationID("receiveSecurityEventTokens"),
	)
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
```

Add `"time"` and the `jwksclient` import to that file, and delete the
`registerReceiverRoutes` stub added at the bottom of `plugin.go` in Task 10.

- [ ] **Step 4: Add the per-IP rate limit**

The spec's first gate is a per-IP rate limit, and the repo already has the
idiom for a plugin-owned route cap. Add a field to `RateLimitConfig` in
`config.go`, next to `VerifyEmailLimit`:

```go
	// SSFPushLimit is the max Shared Signals push deliveries accepted per
	// window per client (default: 60). Defence in depth only: rate limiting
	// is off unless Enabled is set, so the real controls on this endpoint are
	// the secret path, the bearer token, the signature, and the per-stream
	// circuit breaker.
	SSFPushLimit int `json:"ssf_push_limit"`
```

Set its default of 60 wherever the other rate-limit defaults are applied:

```bash
grep -n "VerifyEmailLimit" config.go
```

Then apply it to the route in `registerReceiverRoutes`:

```go
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
```

Import the root package as `authsome "github.com/xraph/authsome"`. That
direction is fine: the root imports `plugin`, never `plugins/*`, and
`plugins/riskengine/contract.go` already does the same thing.

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./plugins/sharedsignals/ -run 'TestPush|TestHashSecret' -v`
Expected: PASS, thirteen tests.

If `forge.Context` has no `Param` method for a raw handler, check how
`plugins/sso` reads `ctx.Param("provider")` in `handleACS` and match it.

- [ ] **Step 6: Run the whole package with the race detector**

Run: `go test ./plugins/sharedsignals/... -race`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add plugins/sharedsignals/receiver.go plugins/sharedsignals/receiver_test.go plugins/sharedsignals/plugin.go config.go
git commit -m "feat(sharedsignals): SSF push endpoint with ordered validation gates"
```

---

### Task 15: Risk contributor, and fixing riskengine's empty UserID

**Files:**
- Create: `plugins/sharedsignals/risk.go`
- Modify: `plugins/riskengine/plugin.go` (the `RiskRequest` struct and `OnBeforeSignIn`)
- Test: `plugins/sharedsignals/risk_test.go`
- Test: `plugins/riskengine/plugin_test.go`

**Interfaces:**
- Consumes: `Store` and `Signal` from Task 6, `Plugin` from Task 10.
- Produces: `(*Plugin).EvaluateRisk(ctx context.Context, req *riskengine.RiskRequest) (*riskengine.RiskSignal, error)`; a new `Email` field and a populated `UserID` on `riskengine.RiskRequest`.

`riskengine.OnBeforeSignIn` currently builds `RiskRequest{IPAddress, UserAgent, AppID}` and never sets `UserID`, so the field is empty on every call. Every contributor shipped so far scores an IP, which is why nobody noticed. Fix that first or this contributor scores nothing forever.

- [ ] **Step 1: Write the failing riskengine test**

Append to `plugins/riskengine/plugin_test.go`:

```go
// capturingContributor records the request it was handed so we can assert on
// what the engine actually populates.
type capturingContributor struct {
	got *RiskRequest
}

func (c *capturingContributor) Name() string { return "capturing" }

func (c *capturingContributor) EvaluateRisk(_ context.Context, req *RiskRequest) (*RiskSignal, error) {
	c.got = req
	return &RiskSignal{Source: "capturing", Score: 0, Weight: 1}, nil
}

// A user-scoped contributor needs something to identify the user by. Before
// this fix the engine passed neither, so the whole class of contributor was
// dead on arrival.
func TestOnBeforeSignIn_PassesIdentifierToContributors(t *testing.T) {
	c := &capturingContributor{}
	p := New(c)
	p.logger = log.NewNoopLogger()

	appID := id.NewAppID()
	err := p.OnBeforeSignIn(context.Background(), &account.SignInRequest{
		AppID:     appID,
		Email:     "target@corp.com",
		IPAddress: "203.0.113.10",
		UserAgent: "test-agent",
	})
	require.NoError(t, err)

	require.NotNil(t, c.got)
	assert.Equal(t, "203.0.113.10", c.got.IPAddress)
	assert.Equal(t, appID.String(), c.got.AppID)
	assert.Equal(t, "target@corp.com", c.got.Email,
		"contributors that score a user need the sign-in identifier")
}

func TestOnBeforeSignIn_PassesUsernameWhenEmailAbsent(t *testing.T) {
	c := &capturingContributor{}
	p := New(c)
	p.logger = log.NewNoopLogger()

	err := p.OnBeforeSignIn(context.Background(), &account.SignInRequest{
		AppID:    id.NewAppID(),
		Username: "targetuser",
	})
	require.NoError(t, err)
	require.NotNil(t, c.got)
	assert.Equal(t, "targetuser", c.got.Username)
}
```

Add whatever of `context`, `testing`, `account`, `id`, `log`, `require` and
`assert` that file does not already import.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./plugins/riskengine/ -run TestOnBeforeSignIn_Passes -v`
Expected: FAIL, `RiskRequest` has no `Email` or `Username` field.

- [ ] **Step 3: Fix riskengine**

In `plugins/riskengine/plugin.go`, extend `RiskRequest`:

```go
// RiskRequest contains the data needed for risk evaluation.
type RiskRequest struct {
	IPAddress string
	UserAgent string
	AppID     string
	EnvID     string

	// UserID is set once the principal is known. It is empty on the
	// pre-authentication path, where Email or Username identify the attempt
	// instead.
	UserID string

	// Email and Username carry the sign-in identifier so a contributor that
	// scores a user rather than an address has something to resolve.
	Email    string
	Username string
}
```

and populate it in `OnBeforeSignIn`:

```go
	riskReq := &RiskRequest{
		IPAddress: req.IPAddress,
		UserAgent: req.UserAgent,
		AppID:     req.AppID.String(),
		EnvID:     req.EnvID.String(),
		Email:     req.Email,
		Username:  req.Username,
	}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./plugins/riskengine/ -v`
Expected: PASS, including the pre-existing tests.

- [ ] **Step 5: Commit the riskengine fix on its own**

It stands alone and is worth reverting independently if it regresses anything.

```bash
git add plugins/riskengine/plugin.go plugins/riskengine/plugin_test.go
git commit -m "fix(riskengine): carry the sign-in identifier into RiskRequest"
```

- [ ] **Step 6: Write the failing contributor test**

Create `plugins/sharedsignals/risk_test.go`:

```go
package sharedsignals

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/riskengine"
	"github.com/xraph/authsome/store/memory"
	"github.com/xraph/authsome/user"
)

func newRiskFixture(t *testing.T) (*Plugin, id.AppID, id.EnvironmentID, *user.User) {
	t.Helper()
	ctx := context.Background()
	appID, envID := id.NewAppID(), id.NewEnvironmentID()

	authStore := memory.New()
	u := &user.User{
		ID: id.NewUserID(), AppID: appID, EnvID: envID,
		Email: "target@corp.com", EmailVerified: true,
	}
	require.NoError(t, authStore.CreateUserWithPrimaryEmail(ctx, u, &user.UserEmail{
		ID: id.NewUserEmailID(), UserID: u.ID, AppID: appID, EnvID: envID,
		Email: u.Email, Verified: true, Primary: true,
	}))

	p := New()
	p.store = NewMemoryStore()
	p.authStore = authStore
	return p, appID, envID, u
}

func TestEvaluateRisk_NoSignalsScoresZero(t *testing.T) {
	p, appID, _, u := newRiskFixture(t)
	got, err := p.EvaluateRisk(context.Background(), &riskengine.RiskRequest{
		AppID: appID.String(), Email: u.Email,
	})
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, 0, got.Score)
}

func TestEvaluateRisk_FreshSignalScoresFull(t *testing.T) {
	ctx := context.Background()
	p, appID, envID, u := newRiskFixture(t)
	now := time.Now()

	require.NoError(t, p.store.CreateSignal(ctx, &Signal{
		ID: id.NewSSFSignalID(), AppID: appID, EnvID: envID, UserID: u.ID,
		EventType: "session-revoked", Severity: 100, Reason: "compromised",
		EventAt: now, ExpiresAt: now.Add(p.config.SignalTTL), CreatedAt: now,
	}))

	got, err := p.EvaluateRisk(ctx, &riskengine.RiskRequest{
		AppID: appID.String(), Email: u.Email,
	})
	require.NoError(t, err)
	assert.Greater(t, got.Score, 90, "a signal received seconds ago has barely decayed")
	assert.Equal(t, "sharedsignals", got.Source)
	assert.Equal(t, 2.0, got.Weight)
}

// A signal near the end of its life should barely move the score, so an old
// event does not keep challenging a user forever.
func TestEvaluateRisk_SignalDecays(t *testing.T) {
	ctx := context.Background()
	p, appID, envID, u := newRiskFixture(t)
	now := time.Now()

	require.NoError(t, p.store.CreateSignal(ctx, &Signal{
		ID: id.NewSSFSignalID(), AppID: appID, EnvID: envID, UserID: u.ID,
		EventType: "session-revoked", Severity: 100,
		EventAt: now.Add(-23 * time.Hour), ExpiresAt: now.Add(time.Hour),
		CreatedAt: now.Add(-23 * time.Hour),
	}))

	got, err := p.EvaluateRisk(ctx, &riskengine.RiskRequest{
		AppID: appID.String(), Email: u.Email,
	})
	require.NoError(t, err)
	assert.Less(t, got.Score, 20, "a nearly expired signal must have decayed")
	assert.Greater(t, got.Score, 0)
}

func TestEvaluateRisk_TakesTheHighestSignal(t *testing.T) {
	ctx := context.Background()
	p, appID, envID, u := newRiskFixture(t)
	now := time.Now()

	for _, sev := range []int{20, 90, 40} {
		require.NoError(t, p.store.CreateSignal(ctx, &Signal{
			ID: id.NewSSFSignalID(), AppID: appID, EnvID: envID, UserID: u.ID,
			EventType: "e", Severity: sev,
			EventAt: now, ExpiresAt: now.Add(p.config.SignalTTL), CreatedAt: now,
		}))
	}

	got, err := p.EvaluateRisk(ctx, &riskengine.RiskRequest{
		AppID: appID.String(), Email: u.Email,
	})
	require.NoError(t, err)
	assert.Greater(t, got.Score, 80)
}

// A sign-in we cannot attribute to a user must score nothing rather than
// guessing.
func TestEvaluateRisk_UnknownUserScoresZero(t *testing.T) {
	p, appID, _, _ := newRiskFixture(t)
	got, err := p.EvaluateRisk(context.Background(), &riskengine.RiskRequest{
		AppID: appID.String(), Email: "stranger@corp.com",
	})
	require.NoError(t, err)
	assert.Equal(t, 0, got.Score)
}

func TestPlugin_IsRiskContributor(t *testing.T) {
	var _ riskengine.RiskContributor = New()
}
```

- [ ] **Step 7: Run test to verify it fails**

Run: `go test ./plugins/sharedsignals/ -run 'EvaluateRisk|RiskContributor' -v`
Expected: FAIL, `EvaluateRisk` is undefined.

- [ ] **Step 8: Write the contributor**

Create `plugins/sharedsignals/risk.go`:

```go
package sharedsignals

import (
	"context"
	"time"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/riskengine"
)

var _ riskengine.RiskContributor = (*Plugin)(nil)

// caepSignalWeight is how much more a CAEP event counts than an IP heuristic.
// An identity provider that watched an account get taken over is a far better
// source than a reputation list.
const caepSignalWeight = 2.0

// EvaluateRisk replays stored signals at sign-in time. This is how an event
// that arrived at 02:00, when nobody was signing in, reaches the 08:00 login
// that cares about it. A high score crosses riskengine's medium threshold and
// its decision becomes "challenge", which is how step-up is expressed without
// inventing a second mechanism.
func (p *Plugin) EvaluateRisk(ctx context.Context, req *riskengine.RiskRequest) (*riskengine.RiskSignal, error) {
	none := &riskengine.RiskSignal{
		Source: "sharedsignals", Score: 0, Weight: caepSignalWeight,
	}

	if p.store == nil || req == nil || req.AppID == "" {
		return none, nil
	}
	appID, err := id.ParseAppID(req.AppID)
	if err != nil {
		return none, nil //nolint:nilerr // an unparseable app is not our failure
	}

	userID, ok := p.resolveRiskUser(ctx, appID, req)
	if !ok {
		return none, nil
	}

	now := time.Now()
	signals, err := p.store.ListActiveSignals(ctx, appID, userID, now)
	if err != nil {
		return none, err
	}
	if len(signals) == 0 {
		return none, nil
	}

	best := 0
	reason := ""
	for _, s := range signals {
		score := decayedSeverity(s, now)
		if score > best {
			best = score
			reason = s.EventType
		}
	}

	return &riskengine.RiskSignal{
		Source: "sharedsignals",
		Score:  best,
		Weight: caepSignalWeight,
		Reason: "shared signals event: " + reason,
	}, nil
}

// resolveRiskUser turns whatever identifies the sign-in attempt into a user.
// UserID is preferred when the caller already knows it.
func (p *Plugin) resolveRiskUser(ctx context.Context, appID id.AppID,
	req *riskengine.RiskRequest) (id.UserID, bool) {
	if req.UserID != "" {
		if userID, err := id.ParseUserID(req.UserID); err == nil {
			return userID, true
		}
	}
	if p.authStore == nil {
		return id.Nil, false
	}
	if req.Email != "" {
		if u, err := p.authStore.GetUserByEmail(ctx, appID, req.Email); err == nil && u != nil {
			return u.ID, true
		}
	}
	if req.Username != "" {
		if u, err := p.authStore.GetUserByUsername(ctx, appID, req.Username); err == nil && u != nil {
			return u.ID, true
		}
	}
	return id.Nil, false
}

// decayedSeverity fades a signal linearly from its full severity at
// EventAt to zero at ExpiresAt.
func decayedSeverity(s *Signal, now time.Time) int {
	total := s.ExpiresAt.Sub(s.EventAt)
	if total <= 0 {
		return 0
	}
	remaining := s.ExpiresAt.Sub(now)
	if remaining <= 0 {
		return 0
	}
	if remaining > total {
		remaining = total
	}
	return int(float64(s.Severity) * (float64(remaining) / float64(total)))
}
```

- [ ] **Step 9: Run test to verify it passes**

Run: `go test ./plugins/sharedsignals/ -run 'EvaluateRisk|RiskContributor' -v`
Expected: PASS, six tests.

- [ ] **Step 10: Commit**

```bash
git add plugins/sharedsignals/risk.go plugins/sharedsignals/risk_test.go
git commit -m "feat(sharedsignals): replay stored CAEP signals into the risk engine"
```

---

### Task 16: Stream admin CRUD

**Files:**
- Create: `plugins/sharedsignals/admin.go`
- Modify: `plugins/sharedsignals/plugin.go` (delete the `registerAdminRoutes` stub from Task 10)
- Test: `plugins/sharedsignals/admin_test.go`

**Interfaces:**
- Consumes: `Store`, `InboundStream`, `NewPushSecret`, `jwksclient.ValidateURI`.
- Produces: `CreateStreamRequest`, `CreateStreamResponse`, `StreamView`, `ListStreamsResponse`, `UpdateStreamRequest`; `(*Plugin).registerAdminRoutes(router forge.Router) error`; `(*Plugin).CreateStream(ctx context.Context, appID id.AppID, envID id.EnvironmentID, req CreateStreamRequest) (*CreateStreamResponse, error)`.

The service method carries the logic and gets the tests; the routes are a thin binding over it.

- [ ] **Step 1: Write the failing test**

Create `plugins/sharedsignals/admin_test.go`:

```go
package sharedsignals

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/sharedsignals/caep"
)

func newAdminFixture(t *testing.T) (*Plugin, id.AppID, id.EnvironmentID) {
	t.Helper()
	p := New(Config{Audience: "https://authsome.example/ssf"})
	p.store = NewMemoryStore()
	return p, id.NewAppID(), id.NewEnvironmentID()
}

func TestCreateStream_ReturnsSecretsOnceAndStoresHashes(t *testing.T) {
	ctx := context.Background()
	p, appID, envID := newAdminFixture(t)

	got, err := p.CreateStream(ctx, appID, envID, CreateStreamRequest{
		Name: "okta-prod", Issuer: "https://org.okta.com",
		JWKSURI: "https://org.okta.com/oauth2/v1/keys",
	})
	require.NoError(t, err)

	assert.NotEmpty(t, got.PushURLPath, "the caller needs the plaintext once")
	assert.NotEmpty(t, got.PushToken)

	stored, err := p.store.GetInboundStream(ctx, got.Stream.ID)
	require.NoError(t, err)
	assert.Equal(t, HashSecret(got.PushURLPath), stored.PushPathHash)
	assert.Equal(t, HashSecret(got.PushToken), stored.PushTokenHash)
	assert.NotContains(t, stored.PushPathHash, got.PushURLPath,
		"the plaintext must never be persisted")
}

func TestCreateStream_AppliesDefaults(t *testing.T) {
	ctx := context.Background()
	p, appID, envID := newAdminFixture(t)

	got, err := p.CreateStream(ctx, appID, envID, CreateStreamRequest{
		Name: "okta", Issuer: "https://org.okta.com",
		JWKSURI: "https://org.okta.com/keys",
	})
	require.NoError(t, err)

	stored, err := p.store.GetInboundStream(ctx, got.Stream.ID)
	require.NoError(t, err)
	assert.Equal(t, StatusEnabled, stored.Status)
	assert.Equal(t, EnforcementEnforce, stored.EnforcementMode)
	assert.Equal(t, 100, stored.MaxActionsPerHour)
	assert.Equal(t, "https://authsome.example/ssf", stored.Audience)
	assert.Equal(t, []string{caep.FormatIssSub}, stored.AllowedSubjectFormats,
		"iss_sub only unless the operator opts into more")
}

func TestCreateStream_RejectsBadInput(t *testing.T) {
	ctx := context.Background()
	p, appID, envID := newAdminFixture(t)

	_, err := p.CreateStream(ctx, appID, envID, CreateStreamRequest{
		Issuer: "", JWKSURI: "https://org.okta.com/keys"})
	require.Error(t, err)

	_, err = p.CreateStream(ctx, appID, envID, CreateStreamRequest{
		Issuer: "https://org.okta.com", JWKSURI: ""})
	require.Error(t, err)
}

// An operator pasting a metadata-service URL is still an SSRF, so the same
// check the fetcher uses runs at registration too.
func TestCreateStream_RejectsUnsafeJWKSURI(t *testing.T) {
	ctx := context.Background()
	p, appID, envID := newAdminFixture(t)

	for _, uri := range []string{
		"http://org.okta.com/keys",
		"https://169.254.169.254/latest/meta-data/",
		"https://127.0.0.1/keys",
	} {
		_, err := p.CreateStream(ctx, appID, envID, CreateStreamRequest{
			Name: "x", Issuer: "https://org.okta.com", JWKSURI: uri,
		})
		require.Error(t, err, "jwks_uri %q must be refused", uri)
	}
}

// Allowing email as a subject format without naming a verified domain would
// let the transmitter name anyone at all.
func TestCreateStream_EmailFormatRequiresVerifiedDomains(t *testing.T) {
	ctx := context.Background()
	p, appID, envID := newAdminFixture(t)

	_, err := p.CreateStream(ctx, appID, envID, CreateStreamRequest{
		Name: "x", Issuer: "https://org.okta.com",
		JWKSURI:               "https://org.okta.com/keys",
		AllowedSubjectFormats: []string{caep.FormatEmail},
	})
	require.Error(t, err)

	_, err = p.CreateStream(ctx, appID, envID, CreateStreamRequest{
		Name: "x", Issuer: "https://org.okta.com",
		JWKSURI:               "https://org.okta.com/keys",
		AllowedSubjectFormats: []string{caep.FormatEmail},
		VerifiedDomains:       []string{"corp.com"},
	})
	require.NoError(t, err)
}

// A listing must never hand back a hash that could be checked offline
// against a guessed secret.
func TestStreamView_OmitsSecrets(t *testing.T) {
	ctx := context.Background()
	p, appID, envID := newAdminFixture(t)

	created, err := p.CreateStream(ctx, appID, envID, CreateStreamRequest{
		Name: "okta", Issuer: "https://org.okta.com",
		JWKSURI: "https://org.okta.com/keys",
	})
	require.NoError(t, err)

	stored, err := p.store.GetInboundStream(ctx, created.Stream.ID)
	require.NoError(t, err)

	view := toStreamView(stored)
	rec := httptest.NewRecorder()
	require.NoError(t, writeJSONForTest(rec, view))

	body := rec.Body.String()
	assert.NotContains(t, body, stored.PushPathHash)
	assert.NotContains(t, body, stored.PushTokenHash)
	assert.Contains(t, body, "https://org.okta.com")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./plugins/sharedsignals/ -run 'CreateStream|StreamView' -v`
Expected: FAIL, `CreateStream` is undefined.

- [ ] **Step 3: Write the admin surface**

Create `plugins/sharedsignals/admin.go`:

```go
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
	"github.com/xraph/authsome/plugins/sharedsignals/caep"
	"github.com/xraph/authsome/plugins/sharedsignals/jwksclient"
)

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

// UpdateStreamRequest changes the mutable parts of a stream. Secrets are not
// among them; rotate by creating a new stream.
type UpdateStreamRequest struct {
	Name                  *string           `json:"name,omitempty"`
	Status                *string           `json:"status,omitempty"`
	EnforcementMode       *string           `json:"enforcement_mode,omitempty"`
	MaxActionsPerHour     *int              `json:"max_actions_per_hour,omitempty"`
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

// CreateStream registers a stream and mints its two secrets.
func (p *Plugin) CreateStream(ctx context.Context, appID id.AppID,
	envID id.EnvironmentID, req CreateStreamRequest) (*CreateStreamResponse, error) {
	if req.Issuer == "" {
		return nil, errors.New("sharedsignals: issuer is required")
	}
	if req.JWKSURI == "" {
		return nil, errors.New("sharedsignals: jwks_uri is required")
	}
	// The same check the fetcher runs, applied before the URL is ever stored.
	if err := jwksclient.ValidateURI(req.JWKSURI); err != nil {
		return nil, fmt.Errorf("sharedsignals: %w", err)
	}

	formats := req.AllowedSubjectFormats
	if len(formats) == 0 {
		// iss_sub only by default. Every other format widens who the
		// transmitter is allowed to name.
		formats = []string{caep.FormatIssSub}
	}
	// Email and phone name a person without the IdP proving it issued them,
	// so they only make sense inside domains the operator has claimed.
	if (containsString(formats, caep.FormatEmail) ||
		containsString(formats, caep.FormatPhoneNumber)) &&
		len(req.VerifiedDomains) == 0 {
		return nil, errors.New(
			"sharedsignals: the email and phone_number formats require verified_domains")
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
		JWKSURI: req.JWKSURI,
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
	if req.AllowedEventTypes != nil {
		stream.AllowedEventTypes = req.AllowedEventTypes
	}
	if req.AllowedSubjectFormats != nil {
		if (containsString(req.AllowedSubjectFormats, caep.FormatEmail) ||
			containsString(req.AllowedSubjectFormats, caep.FormatPhoneNumber)) &&
			len(stream.VerifiedDomains) == 0 && len(req.VerifiedDomains) == 0 {
			return nil, errors.New(
				"sharedsignals: the email and phone_number formats require verified_domains")
		}
		stream.AllowedSubjectFormats = req.AllowedSubjectFormats
	}
	if req.VerifiedDomains != nil {
		stream.VerifiedDomains = req.VerifiedDomains
	}
	if req.ActionOverrides != nil {
		stream.ActionOverrides = req.ActionOverrides
	}

	if err := p.store.UpdateInboundStream(ctx, stream); err != nil {
		return nil, err
	}
	view := toStreamView(stream)
	return &view, nil
}

// registerAdminRoutes mounts the stream CRUD behind session auth.
func (p *Plugin) registerAdminRoutes(router forge.Router) error {
	g := router.Group("/v1/ssf/admin",
		forge.WithGroupTags("Shared Signals"),
		forge.WithGroupAuth("session"),
	)

	if err := g.POST("/streams", p.handleCreateStream,
		forge.WithSummary("Register an inbound Shared Signals stream"),
		forge.WithOperationID("createSSFStream"),
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
		forge.WithSummary("Update an inbound Shared Signals stream"),
		forge.WithOperationID("updateSSFStream"),
	); err != nil {
		return err
	}
	return g.DELETE("/streams/:id", p.handleDeleteStream,
		forge.WithSummary("Delete an inbound Shared Signals stream"),
		forge.WithOperationID("deleteSSFStream"),
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
	if err := json.Unmarshal(body, &req); err != nil {
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

func (p *Plugin) handleUpdateStream(ctx forge.Context) error {
	streamID, perr := id.ParseWithPrefix(ctx.Param("id"), id.PrefixSSFStream)
	if perr != nil {
		return forge.BadRequest("invalid stream id")
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
func (p *Plugin) requestScope(ctx forge.Context) (id.AppID, id.EnvironmentID, error) {
	envID, _ := middleware.EnvIDFrom(ctx.Context())

	if appID, ok := middleware.AppIDFrom(ctx.Context()); ok {
		return appID, envID, nil
	}
	appID, err := id.ParseAppID(p.engine.DefaultAppID())
	if err != nil {
		return id.Nil, id.Nil, forge.BadRequest("invalid app configuration")
	}
	return appID, envID, nil
}
```

Add `"github.com/xraph/authsome/middleware"` to the import block.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./plugins/sharedsignals/ -run 'CreateStream|StreamView' -v`
Expected: PASS, six tests.

- [ ] **Step 5: Delete the Task 10 stub and run the whole package**

Remove `func (p *Plugin) registerAdminRoutes(_ forge.Router) error { return nil }`
from `plugin.go`.

Run: `go test ./plugins/sharedsignals/... -race`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add plugins/sharedsignals/admin.go plugins/sharedsignals/admin_test.go plugins/sharedsignals/plugin.go
git commit -m "feat(sharedsignals): stream registration and admin CRUD"
```

---

### Task 17: End-to-end through a real engine

**Files:**
- Create: `plugins/sharedsignals/e2e_test.go`
- Test: `plugins/sharedsignals/e2e_test.go`

**Interfaces:**
- Consumes: everything. This task adds no production code.

Every test so far has poked one layer. This one stands up a real engine, registers the plugin, points a fake transmitter at it, pushes a `session-revoked` and asserts the session is gone from the store. It is the test that would have caught the whole feature being wired up wrong.

- [ ] **Step 1: Write the end-to-end test**

Create `plugins/sharedsignals/e2e_test.go`:

```go
package sharedsignals_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/secutil"
	"github.com/xraph/authsome/plugins/sharedsignals"
	"github.com/xraph/authsome/plugins/sharedsignals/caep"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/user"
)

// fakeTransmitter is an identity provider: a keypair, a JWKS endpoint and the
// ability to sign a SET the way Okta does.
type fakeTransmitter struct {
	key    *rsa.PrivateKey
	server *httptest.Server
	issuer string
}

func newFakeTransmitter(t *testing.T) *fakeTransmitter {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := base64.RawURLEncoding.EncodeToString(key.PublicKey.N.Bytes())
		e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.PublicKey.E)).Bytes())
		fmt.Fprintf(w, `{"keys":[{"kty":"RSA","use":"sig","alg":"RS256","kid":"k1","n":%q,"e":%q}]}`, n, e)
	}))
	t.Cleanup(srv.Close)

	return &fakeTransmitter{key: key, server: srv, issuer: "https://idp.test.example"}
}

// sessionRevokedSET signs a session-revoked event in Okta's shape, meaning
// the subject arrives under "subject" rather than "sub_id".
func (f *fakeTransmitter) sessionRevokedSET(t *testing.T, audience, subject string) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iss": f.issuer,
		"aud": audience,
		"jti": "jti-" + id.NewSSFEventID().String(),
		"iat": time.Now().Unix(),
		"events": map[string]any{
			caep.EventSessionRevoked: map[string]any{
				"subject": map[string]any{
					"format": "iss_sub", "iss": f.issuer, "sub": subject,
				},
				"reason_admin":    map[string]any{"en": "Account compromised"},
				"event_timestamp": time.Now().UnixMilli(),
			},
		},
	})
	tok.Header["typ"] = "secevent+jwt"
	tok.Header["kid"] = "k1"
	signed, err := tok.SignedString(f.key)
	require.NoError(t, err)
	return signed
}

// The whole point of the feature, in one test: an upstream compromise ends
// the sessions authsome issued.
func TestEndToEnd_UpstreamRevocationKillsLiveSessions(t *testing.T) {
	ctx := context.Background()
	idp := newFakeTransmitter(t)

	ssf := sharedsignals.New(sharedsignals.Config{
		Audience: "https://authsome.test/ssf",
	})
	eng := secutil.NewTestEngine(t, authsome.WithPlugin(ssf))

	appID, err := id.ParseAppID(eng.DefaultAppID())
	require.NoError(t, err)

	// A user with two live sessions.
	u := &user.User{
		ID: id.NewUserID(), AppID: appID,
		Email: "victim@corp.com", EmailVerified: true,
	}
	require.NoError(t, eng.Store().CreateUserWithPrimaryEmail(ctx, u, &user.UserEmail{
		ID: id.NewUserEmailID(), UserID: u.ID, AppID: appID,
		Email: u.Email, Verified: true, Primary: true,
	}))
	for i := 0; i < 2; i++ {
		require.NoError(t, eng.Store().CreateSession(ctx, &session.Session{
			ID: id.NewSessionID(), AppID: appID, UserID: u.ID,
			Token:     fmt.Sprintf("live-token-%d", i),
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}))
	}

	live, err := eng.ListSessions(ctx, u.ID)
	require.NoError(t, err)
	require.Len(t, live, 2, "the user starts with two live sessions")

	// Register the IdP as a stream and link its subject to our user, which is
	// what an SSO sign-in would have done.
	created, err := ssf.CreateStream(ctx, appID, id.Nil, sharedsignals.CreateStreamRequest{
		Name:    "test-idp",
		Issuer:  idp.issuer,
		JWKSURI: idp.server.URL,
	})
	require.NoError(t, err)
	require.NoError(t, ssf.LinkSubject(ctx, appID, id.Nil,
		idp.issuer, "idp-user-1", u.ID, sharedsignals.SourceSSO))

	// The IdP decides the account is compromised and pushes the event.
	body := idp.sessionRevokedSET(t, "https://authsome.test/ssf", "idp-user-1")
	req := httptest.NewRequest(http.MethodPost,
		"/v1/ssf/streams/"+created.PushURLPath+"/events", stringReader(body))
	req.Header.Set("Content-Type", "application/secevent+jwt")
	req.Header.Set("Authorization", "Bearer "+created.PushToken)
	rec := httptest.NewRecorder()

	ssf.ServePushForTest(rec, req, created.PushURLPath)
	require.Equal(t, http.StatusAccepted, rec.Code, "body: %s", rec.Body.String())

	// The sessions are gone from the store, not merely marked.
	after, err := eng.ListSessions(ctx, u.ID)
	require.NoError(t, err)
	assert.Empty(t, after, "an upstream revocation must end every live session")

	// And a durable signal is left behind for the next sign-in to score.
	riskReq := riskRequestFor(appID, u.Email)
	signal, err := ssf.EvaluateRisk(ctx, riskReq)
	require.NoError(t, err)
	assert.Greater(t, signal.Score, 90,
		"the revocation must also leave a high-confidence risk signal")
}

// A SET signed by a key the stream does not trust changes nothing.
func TestEndToEnd_ForgedSETLeavesSessionsAlone(t *testing.T) {
	ctx := context.Background()
	realIDP := newFakeTransmitter(t)
	attacker := newFakeTransmitter(t)
	attacker.issuer = realIDP.issuer // same issuer claim, different key

	ssf := sharedsignals.New(sharedsignals.Config{Audience: "https://authsome.test/ssf"})
	eng := secutil.NewTestEngine(t, authsome.WithPlugin(ssf))

	appID, err := id.ParseAppID(eng.DefaultAppID())
	require.NoError(t, err)

	u := &user.User{ID: id.NewUserID(), AppID: appID,
		Email: "victim@corp.com", EmailVerified: true}
	require.NoError(t, eng.Store().CreateUserWithPrimaryEmail(ctx, u, &user.UserEmail{
		ID: id.NewUserEmailID(), UserID: u.ID, AppID: appID,
		Email: u.Email, Verified: true, Primary: true,
	}))
	require.NoError(t, eng.Store().CreateSession(ctx, &session.Session{
		ID: id.NewSessionID(), AppID: appID, UserID: u.ID,
		Token: "live-token", ExpiresAt: time.Now().Add(24 * time.Hour),
	}))

	created, err := ssf.CreateStream(ctx, appID, id.Nil, sharedsignals.CreateStreamRequest{
		Name: "test-idp", Issuer: realIDP.issuer, JWKSURI: realIDP.server.URL,
	})
	require.NoError(t, err)
	require.NoError(t, ssf.LinkSubject(ctx, appID, id.Nil,
		realIDP.issuer, "idp-user-1", u.ID, sharedsignals.SourceSSO))

	// Signed by the attacker's key, but the stream only trusts the real IdP's
	// JWKS, so the signature check fails.
	body := attacker.sessionRevokedSET(t, "https://authsome.test/ssf", "idp-user-1")
	req := httptest.NewRequest(http.MethodPost,
		"/v1/ssf/streams/"+created.PushURLPath+"/events", stringReader(body))
	req.Header.Set("Content-Type", "application/secevent+jwt")
	req.Header.Set("Authorization", "Bearer "+created.PushToken)
	rec := httptest.NewRecorder()

	ssf.ServePushForTest(rec, req, created.PushURLPath)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	after, err := eng.ListSessions(ctx, u.ID)
	require.NoError(t, err)
	assert.Len(t, after, 1, "a forged SET must leave every session alone")
}
```

- [ ] **Step 2: Export the two test seams this needs**

The end-to-end test lives in package `sharedsignals_test`, so it needs two
exported helpers. In `plugins/sharedsignals/receiver.go`, rename the unexported
`servePushForTest` to an exported one and keep the internal tests calling it:

```go
// ServePushForTest drives the push pipeline without a forge context. It is
// exported for end-to-end tests that live outside this package; production
// traffic arrives through handlePush.
func (p *Plugin) ServePushForTest(w http.ResponseWriter, r *http.Request, pushPath string) {
	p.servePush(w, r, pushPath)
}
```

Update the call in `receiver_test.go` from `servePushForTest` to
`ServePushForTest`.

Add the two small helpers the e2e file references at the bottom of
`e2e_test.go`:

```go
func stringReader(s string) *strings.Reader { return strings.NewReader(s) }

func riskRequestFor(appID id.AppID, email string) *riskengine.RiskRequest {
	return &riskengine.RiskRequest{AppID: appID.String(), Email: email}
}
```

with `"strings"` and `"github.com/xraph/authsome/plugins/riskengine"` imported.

- [ ] **Step 3: Run the end-to-end test**

Run: `go test ./plugins/sharedsignals/ -run TestEndToEnd -v`
Expected: PASS, two tests.

If `TestEndToEnd_UpstreamRevocationKillsLiveSessions` reports 202 but leaves
the sessions alive, the plugin did not pick up a `SessionRevoker` in `OnInit`.
Check that `secutil.NewTestEngine` passes `*authsome.Engine` (not a wrapper)
into `OnInit`, and that the type assertion in `OnInit` succeeds.

- [ ] **Step 4: Run everything**

```bash
go test ./plugins/sharedsignals/... ./plugins/riskengine/... ./plugins/sso/... ./id/... ./plugin/... -race
```

Expected: PASS.

- [ ] **Step 5: Run the full suite and the linter**

```bash
make test
```

```bash
make lint
```

Expected: both clean. Do not move on with a failing lint; the `//nolint`
comments in this plan cover the intentional best-effort error drops, and
anything else the linter finds is a real defect.

- [ ] **Step 6: Commit**

```bash
git add plugins/sharedsignals/e2e_test.go plugins/sharedsignals/receiver.go plugins/sharedsignals/receiver_test.go
git commit -m "test(sharedsignals): end-to-end revocation and forged-token coverage"
```

---

## Wiring it up

The plugin is registered like any other, and it needs to be added wherever the
host builds its engine:

```go
import "github.com/xraph/authsome/plugins/sharedsignals"

ssf := sharedsignals.New(sharedsignals.Config{
    Audience: "https://auth.example.com/ssf",
})

eng, err := authsome.NewEngine(
    authsome.WithPlugin(ssf),
    // Feed CAEP signals into the composite score alongside the IP-based
    // contributors. Without this the signals are recorded but never scored.
    authsome.WithPlugin(riskengine.New(ipreputation.New(), ssf)),
)
```

Registration order matters in one direction only: `riskengine` holds the
contributor slice, so `ssf` has to exist before it is constructed. Both are
registered with the engine independently.

## Definition of done

- [ ] `make test` passes.
- [ ] `make lint` passes.
- [ ] `go test ./plugins/sharedsignals/... -race` passes.
- [ ] Every negative security test named in the spec has a test in this
      package: `alg: none`, HMAC confusion, wrong `iss`, wrong `aud`, stale
      `iat`, future `iat`, missing `jti`, over-long `jti`, replayed `jti`,
      unknown `kid`, wrong signing key, cross-stream replay, cross-app
      subject, email outside `verified_domains`, unverified email, disagreeing
      aliases, oversized body, oversized JWKS, and a circuit breaker trip.
- [ ] The store conformance suite runs against memory and sqlite.
- [ ] A real Okta-shaped SET, meaning `subject` rather than `sub_id`, is in
      the fixtures and passes.

## What M1 deliberately leaves out

M2 through M4 in the spec: the outbound stream client, the dashboard contract
and dashui page, and the transmitter emit path. Also poll-based receiving, the
`account`, `uri` and `did` subject formats, and transmitter key rotation.
