package passkey

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/user"
)

// ──────────────────────────────────────────────────
// Plugin basics
// ──────────────────────────────────────────────────

func TestPlugin_Name(t *testing.T) {
	p := New(Config{})
	assert.Equal(t, "passkey", p.Name())
}

func TestPlugin_Defaults(t *testing.T) {
	p := New(Config{})
	assert.Equal(t, "AuthSome", p.config.RPDisplayName)
	assert.Equal(t, "localhost", p.config.RPID)
}

func TestPlugin_OnInit(t *testing.T) {
	p := New(Config{
		RPDisplayName: "Test",
		RPID:          "localhost",
		RPOrigins:     []string{"https://localhost"},
	})
	err := p.OnInit(context.Background(), nil)
	require.NoError(t, err)
	assert.NotNil(t, p.wa)
}

func TestPlugin_SetStore(t *testing.T) {
	p := New(Config{})
	ms := NewMemoryStore()
	p.SetStore(ms)
	assert.NotNil(t, p.store)
}

// ──────────────────────────────────────────────────
// Memory store tests
// ──────────────────────────────────────────────────

func TestMemoryStore_CreateAndGet(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()

	cred := &Credential{
		ID:           id.NewPasskeyID(),
		UserID:       id.NewUserID(),
		AppID:        id.NewAppID(),
		CredentialID: []byte{0x01, 0x02, 0x03},
		PublicKey:    []byte{0xAA, 0xBB},
		DisplayName:  "Test Key",
	}

	err := s.CreateCredential(ctx, cred)
	require.NoError(t, err)

	got, err := s.GetCredential(ctx, []byte{0x01, 0x02, 0x03})
	require.NoError(t, err)
	assert.Equal(t, "Test Key", got.DisplayName)
	assert.NotZero(t, got.CreatedAt)
}

func TestMemoryStore_GetNotFound(t *testing.T) {
	s := NewMemoryStore()
	_, err := s.GetCredential(context.Background(), []byte{0xFF})
	assert.ErrorIs(t, err, ErrCredentialNotFound)
}

func TestMemoryStore_ListUserCredentials(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	userID := id.NewUserID()

	for i := range 3 {
		_ = s.CreateCredential(ctx, &Credential{
			ID:           id.NewPasskeyID(),
			UserID:       userID,
			CredentialID: []byte{byte(i)},
			DisplayName:  "Key",
		})
	}

	// Another user's credential
	_ = s.CreateCredential(ctx, &Credential{
		ID:           id.NewPasskeyID(),
		UserID:       id.NewUserID(),
		CredentialID: []byte{0xFF},
	})

	creds, err := s.ListUserCredentials(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, creds, 3)
}

func TestMemoryStore_Delete(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()

	credID := []byte{0x01, 0x02}
	_ = s.CreateCredential(ctx, &Credential{
		ID:           id.NewPasskeyID(),
		UserID:       id.NewUserID(),
		CredentialID: credID,
	})

	err := s.DeleteCredential(ctx, credID)
	require.NoError(t, err)

	_, err = s.GetCredential(ctx, credID)
	assert.ErrorIs(t, err, ErrCredentialNotFound)
}

func TestMemoryStore_DeleteNotFound(t *testing.T) {
	s := NewMemoryStore()
	err := s.DeleteCredential(context.Background(), []byte{0xFF})
	assert.ErrorIs(t, err, ErrCredentialNotFound)
}

func TestMemoryStore_UpdateSignCount(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()

	credID := []byte{0x01}
	_ = s.CreateCredential(ctx, &Credential{
		ID:           id.NewPasskeyID(),
		UserID:       id.NewUserID(),
		CredentialID: credID,
		SignCount:    0,
	})

	err := s.UpdateSignCount(ctx, credID, 5)
	require.NoError(t, err)

	cred, _ := s.GetCredential(ctx, credID)
	assert.Equal(t, uint32(5), cred.SignCount)
}

func TestMemoryStore_UpdateSignCountNotFound(t *testing.T) {
	s := NewMemoryStore()
	err := s.UpdateSignCount(context.Background(), []byte{0xFF}, 1)
	assert.ErrorIs(t, err, ErrCredentialNotFound)
}

// ──────────────────────────────────────────────────
// WebAuthn user adapter tests
// ──────────────────────────────────────────────────

func TestWebAuthnUser_Interface(t *testing.T) {
	u := &user.User{
		ID:       id.NewUserID(),
		Email:    "alice@example.com",
		Username: "alice",
		FirstName: "Alice Smith",
	}

	wau := &webAuthnUser{user: u}

	assert.Equal(t, []byte(u.ID.String()), wau.WebAuthnID())
	assert.Equal(t, "alice", wau.WebAuthnName())
	assert.Equal(t, "Alice Smith", wau.WebAuthnDisplayName())
	assert.Empty(t, wau.WebAuthnIcon())
	assert.Empty(t, wau.WebAuthnCredentials())
}

func TestWebAuthnUser_FallbackToEmail(t *testing.T) {
	u := &user.User{
		ID:    id.NewUserID(),
		Email: "bob@example.com",
	}

	wau := &webAuthnUser{user: u}

	assert.Equal(t, "bob@example.com", wau.WebAuthnName())
	assert.Equal(t, "bob@example.com", wau.WebAuthnDisplayName())
}

// ──────────────────────────────────────────────────
// Credential conversion
// ──────────────────────────────────────────────────

func TestCredential_FieldsPopulated(t *testing.T) {
	c := &Credential{
		ID:              id.NewPasskeyID(),
		UserID:          id.NewUserID(),
		AppID:           id.NewAppID(),
		CredentialID:    []byte{0x01, 0x02, 0x03},
		PublicKey:       []byte{0xAA, 0xBB},
		AttestationType: "none",
		Transport:       []string{"internal"},
		SignCount:       42,
		DisplayName:     "My Passkey",
	}

	assert.NotEmpty(t, c.ID.String())
	assert.Equal(t, "none", c.AttestationType)
	assert.Equal(t, uint32(42), c.SignCount)
	assert.Equal(t, "My Passkey", c.DisplayName)
}
