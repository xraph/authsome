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
