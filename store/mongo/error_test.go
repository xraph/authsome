package mongo

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/store"
)

// dupKeyErr fabricates a write error that mongo.IsDuplicateKeyError
// recognizes. The driver's typed WriteException carries the magic
// code 11000 — we synthesize that minimal shape so the test doesn't
// need a live mongo to exercise mapWriteErr.
func dupKeyErr(msg string) error {
	return mongo.WriteException{
		WriteErrors: mongo.WriteErrors{
			{Code: 11000, Message: msg},
		},
	}
}

func TestMapWriteErr_NilPassesThrough(t *testing.T) {
	t.Parallel()
	assert.Nil(t, mapWriteErr(nil))
}

// TestMapWriteErr_DuplicateUsernameMapsSentinel pins the fix for
// the production bug: a duplicate-key violation on the username
// index now surfaces as account.ErrUsernameTaken (which the API
// maps to 409 + a friendly message) instead of leaking the raw
// E11000 + index name + colliding key value.
func TestMapWriteErr_DuplicateUsernameMapsSentinel(t *testing.T) {
	t.Parallel()
	raw := dupKeyErr(
		`E11000 duplicate key error collection: twinos-platform.authsome_users index: app_id_1_username_1 dup key: { app_id: "aapp_x", username: "" }`)
	got := mapWriteErr(raw)
	assert.True(t, errors.Is(got, account.ErrUsernameTaken),
		"username dup-key must map to ErrUsernameTaken; got %v", got)
}

func TestMapWriteErr_DuplicateEmailMapsSentinel(t *testing.T) {
	t.Parallel()
	raw := dupKeyErr(
		`E11000 duplicate key error collection: db.authsome_users index: app_id_1_email_1 dup key: { app_id: "aapp_x", email: "u@example.com" }`)
	got := mapWriteErr(raw)
	assert.True(t, errors.Is(got, account.ErrEmailTaken),
		"email dup-key must map to ErrEmailTaken; got %v", got)
}

func TestMapWriteErr_OtherDuplicateMapsToConflict(t *testing.T) {
	t.Parallel()
	raw := dupKeyErr(
		`E11000 duplicate key error collection: db.authsome_organizations index: app_id_1_slug_1 dup key: {}`)
	got := mapWriteErr(raw)
	assert.True(t, errors.Is(got, store.ErrConflict),
		"unrecognized dup-key must map to store.ErrConflict (generic 409 in API); got %v", got)
}

func TestMapWriteErr_NonDuplicatePassesThrough(t *testing.T) {
	t.Parallel()
	raw := errors.New("network: connection reset by peer")
	got := mapWriteErr(raw)
	assert.Equal(t, raw, got)
}

// ──────────────────────────────────────────────────
// Index-conflict detection (eager-path tolerance)
// ──────────────────────────────────────────────────

func TestMongoIsIndexConflict_NilFalse(t *testing.T) {
	t.Parallel()
	assert.False(t, mongoIsIndexConflict(nil))
}

func TestMongoIsIndexConflict_RecognizesCode86(t *testing.T) {
	t.Parallel()
	// IndexKeySpecsConflict — mongo returns this when an existing
	// index has the same name but different spec. This is the exact
	// shape the production failure surfaced (sparse → partialFilter).
	got := mongoIsIndexConflict(mongo.CommandError{
		Code:    86,
		Name:    "IndexKeySpecsConflict",
		Message: "An existing index has the same name as the requested index.",
	})
	assert.True(t, got, "code 86 must be recognized as an index conflict")
}

func TestMongoIsIndexConflict_RecognizesCode85(t *testing.T) {
	t.Parallel()
	// IndexOptionsConflict — sibling code for option-only differences.
	got := mongoIsIndexConflict(mongo.CommandError{
		Code:    85,
		Name:    "IndexOptionsConflict",
		Message: "Index already exists with different options",
	})
	assert.True(t, got, "code 85 must be recognized as an index conflict")
}

func TestMongoIsIndexConflict_StringFallback(t *testing.T) {
	t.Parallel()
	// Some driver versions / wrapping layers surface the conflict as
	// a plain error string rather than a typed CommandError. The
	// fallback substring match catches those too.
	for _, msg := range []string{
		"create indexes: IndexKeySpecsConflict: name in use",
		"wrapped: (IndexOptionsConflict) options differ",
	} {
		assert.True(t, mongoIsIndexConflict(errors.New(msg)),
			"string fallback must recognize %q", msg)
	}
}

func TestMongoIsIndexConflict_OtherErrorsFalse(t *testing.T) {
	t.Parallel()
	cases := []error{
		errors.New("connection refused"),
		mongo.CommandError{Code: 11000, Name: "DuplicateKey", Message: "E11000 dup key"},
		mongo.CommandError{Code: 27, Name: "IndexNotFound", Message: "no index"},
	}
	for _, e := range cases {
		assert.False(t, mongoIsIndexConflict(e),
			"%v must NOT be recognized as an index conflict", e)
	}
}

// ──────────────────────────────────────────────────
// Migration tracking corruption (upstream library bug)
// ──────────────────────────────────────────────────

// TestMongoIsMigrationDecodeCorruption_RecognizesUpstreamBug pins
// detection of the exact error string the broken-shape
// grove_migrations collection produces. The substring match is
// load-bearing: the wrapping layers add prefixes ("authsome/mongo:
// migration failed:", "migrate: list applied:") so we have to
// match on the inner cause.
func TestMongoIsMigrationDecodeCorruption_RecognizesUpstreamBug(t *testing.T) {
	t.Parallel()
	raw := errors.New("authsome/mongo: migration failed: migrate: list applied: mongomigrate: decode applied: error decoding key _id: cannot decode objectID into an integer type")
	assert.True(t, mongoIsMigrationDecodeCorruption(raw),
		"the production failure message must be recognized as the upstream decode-corruption bug")
}

func TestMongoIsMigrationDecodeCorruption_NilFalse(t *testing.T) {
	t.Parallel()
	assert.False(t, mongoIsMigrationDecodeCorruption(nil))
}

func TestMongoIsMigrationDecodeCorruption_OtherErrorsFalse(t *testing.T) {
	t.Parallel()
	cases := []error{
		errors.New("connection refused"),
		errors.New("decode applied: cursor closed"),                           // partial match — must still fail
		errors.New("error decoding key _id: cannot decode int into objectID"), // opposite direction — also a failure mode but not the one we self-heal
	}
	for _, e := range cases {
		assert.False(t, mongoIsMigrationDecodeCorruption(e),
			"%v must NOT trigger self-heal", e)
	}
}
