package sqlite

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/store"
)

func TestSqliteError_NoRowsReturnsErrNotFound(t *testing.T) {
	t.Parallel()
	got := sqliteError(sql.ErrNoRows)
	assert.True(t, errors.Is(got, store.ErrNotFound))
}

// SQLite reports unique violations as
//   "UNIQUE constraint failed: <table>.<column>[, <table>.<column>...]"
// We pin the column-substring routing so duplicate email/username
// surface as the right account-level sentinel.
func TestSqliteError_DuplicateEmailMapsSentinel(t *testing.T) {
	t.Parallel()
	raw := errors.New("UNIQUE constraint failed: authsome_users.app_id, authsome_users.email")
	got := sqliteError(raw)
	assert.True(t, errors.Is(got, account.ErrEmailTaken),
		"unique-violation on the email index must map to ErrEmailTaken; got %v", got)
}

func TestSqliteError_DuplicateUsernameMapsSentinel(t *testing.T) {
	t.Parallel()
	raw := errors.New("UNIQUE constraint failed: authsome_users.app_id, authsome_users.username")
	got := sqliteError(raw)
	assert.True(t, errors.Is(got, account.ErrUsernameTaken),
		"unique-violation on the username index must map to ErrUsernameTaken; got %v", got)
}

func TestSqliteError_OtherUniqueMapsToConflict(t *testing.T) {
	t.Parallel()
	raw := errors.New("UNIQUE constraint failed: authsome_orgs.slug")
	got := sqliteError(raw)
	assert.True(t, errors.Is(got, store.ErrConflict))
}

func TestSqliteError_UnrelatedErrorPassesThrough(t *testing.T) {
	t.Parallel()
	raw := errors.New("disk I/O error")
	got := sqliteError(raw)
	assert.Equal(t, raw, got)
}
