package postgres

import (
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/store"
)

// TestPgError_NoRowsReturnsErrNotFound pins the existing
// translation that the rest of the codebase reasons about.
func TestPgError_NoRowsReturnsErrNotFound(t *testing.T) {
	t.Parallel()
	got := pgError(sql.ErrNoRows)
	assert.True(t, errors.Is(got, store.ErrNotFound),
		"sql.ErrNoRows must map to store.ErrNotFound; got %v", got)
}

// TestPgError_DuplicateEmailMapsSentinel pins that a unique-
// constraint violation on the email index surfaces as
// account.ErrEmailTaken — the API maps it to 409 — instead of
// bubbling up the raw 23505 message that would leak the index
// name and the colliding value.
func TestPgError_DuplicateEmailMapsSentinel(t *testing.T) {
	t.Parallel()
	// Mirrors the surface message pgx wraps around the underlying
	// PgError. We match by substring so any wrapping the grove
	// layer adds doesn't break the translation.
	raw := errors.New(`ERROR: duplicate key value violates unique constraint "idx_authsome_users_email" (SQLSTATE 23505)`)
	got := pgError(raw)
	assert.True(t, errors.Is(got, account.ErrEmailTaken),
		"unique-violation on the email index must map to ErrEmailTaken; got %v", got)
}

// TestPgError_DuplicateUsernameMapsSentinel pins the same for
// the username index — fixes the production bug that surfaced
// the raw E11000-equivalent message instead of a clean 409.
func TestPgError_DuplicateUsernameMapsSentinel(t *testing.T) {
	t.Parallel()
	raw := fmt.Errorf("insert: %s",
		`ERROR: duplicate key value violates unique constraint "idx_authsome_users_username" (SQLSTATE 23505)`)
	got := pgError(raw)
	assert.True(t, errors.Is(got, account.ErrUsernameTaken),
		"unique-violation on the username index must map to ErrUsernameTaken; got %v", got)
}

// TestPgError_OtherUniqueMapsToConflict pins the catch-all: any
// unrecognized unique violation surfaces as store.ErrConflict
// (generic 409) rather than the raw error which would leak the
// index name.
func TestPgError_OtherUniqueMapsToConflict(t *testing.T) {
	t.Parallel()
	raw := errors.New(`ERROR: duplicate key value violates unique constraint "idx_some_other_thing" (SQLSTATE 23505)`)
	got := pgError(raw)
	assert.True(t, errors.Is(got, store.ErrConflict),
		"unrecognized unique violation must map to store.ErrConflict; got %v", got)
}

// TestPgError_UnrelatedErrorPassesThrough pins that non-
// duplicate errors are NOT translated — operators still see the
// root cause in logs.
func TestPgError_UnrelatedErrorPassesThrough(t *testing.T) {
	t.Parallel()
	raw := errors.New("connection refused")
	got := pgError(raw)
	assert.Equal(t, raw, got)
}
