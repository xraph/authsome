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
