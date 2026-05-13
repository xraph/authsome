package passkey

import (
	"context"
	"net/http"
	"net/http/httptest"
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
		ID:        id.NewUserID(),
		Email:     "alice@example.com",
		Username:  "alice",
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

// ──────────────────────────────────────────────────
// waForRequest — per-request origin handling
// ──────────────────────────────────────────────────

func newRequestWithOrigin(origin string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/v1/passkeys/register/finish", nil)
	if origin != "" {
		r.Header.Set("Origin", origin)
	}
	return r
}

func TestWaForRequest_LocalhostAcceptsAnyPort(t *testing.T) {
	p := New(Config{RPID: "localhost"})
	require.NotNil(t, p.wa)

	wa := p.waForRequest(newRequestWithOrigin("http://localhost:3000"))
	require.NotNil(t, wa)
	// Must be a different (per-origin) instance, not the base p.wa.
	assert.NotSame(t, p.wa, wa, "expected a per-origin webauthn instance for localhost dev mode")
}

func TestWaForRequest_LocalhostMismatchedHostFallsBack(t *testing.T) {
	p := New(Config{RPID: "localhost"})
	require.NotNil(t, p.wa)

	// Origin host (evil.com) does not match RPID (localhost) — must NOT be
	// added to RPOrigins. Falls back to the base config.
	wa := p.waForRequest(newRequestWithOrigin("http://evil.com:3000"))
	assert.Same(t, p.wa, wa, "evil origins must not produce a new webauthn instance")
}

func TestWaForRequest_NonLocalhostUnchanged(t *testing.T) {
	p := New(Config{RPID: "example.com", RPOrigins: []string{"https://example.com"}})
	require.NotNil(t, p.wa)

	// Even if the request comes from localhost, a production RPID stays strict.
	wa := p.waForRequest(newRequestWithOrigin("http://localhost:3000"))
	assert.Same(t, p.wa, wa)
}

func TestWaForRequest_MissingOriginHeaderFallsBack(t *testing.T) {
	p := New(Config{RPID: "localhost"})
	require.NotNil(t, p.wa)

	wa := p.waForRequest(newRequestWithOrigin(""))
	assert.Same(t, p.wa, wa, "missing Origin header should not change webauthn instance")
}

func TestWaForRequest_NilRequest(t *testing.T) {
	p := New(Config{RPID: "localhost"})
	require.NotNil(t, p.wa)
	assert.Same(t, p.wa, p.waForRequest(nil))
}

func TestWaForRequest_CachesPerOrigin(t *testing.T) {
	p := New(Config{RPID: "localhost"})

	wa1 := p.waForRequest(newRequestWithOrigin("http://localhost:3000"))
	wa2 := p.waForRequest(newRequestWithOrigin("http://localhost:3000"))
	assert.Same(t, wa1, wa2, "identical request origins must hit the cache")

	wa3 := p.waForRequest(newRequestWithOrigin("http://localhost:5173"))
	assert.NotSame(t, wa1, wa3, "different origins should produce different cached instances")
}

func TestWaForRequest_RefererFallback(t *testing.T) {
	p := New(Config{RPID: "localhost"})

	r := httptest.NewRequest(http.MethodPost, "/v1/passkeys/register/finish", nil)
	r.Header.Set("Referer", "http://localhost:4321/some/path?q=1")
	wa := p.waForRequest(r)
	assert.NotSame(t, p.wa, wa, "Referer should be used when Origin is absent")
}

func TestNew_NonLocalhostDefaultsToHTTPS(t *testing.T) {
	p := New(Config{RPID: "example.com"})
	require.Len(t, p.config.RPOrigins, 1)
	assert.Equal(t, "https://example.com", p.config.RPOrigins[0])
}

func TestNew_LocalhostLeavesOriginsEmpty(t *testing.T) {
	p := New(Config{RPID: "localhost"})
	assert.Empty(t, p.config.RPOrigins, "localhost RPID should not get a static origin default; per-request resolution handles it")
}

func TestAllowedOrigins_NoSettingsManager(t *testing.T) {
	p := New(Config{RPID: "example.com", RPOrigins: []string{"https://example.com", "https://example.com"}})
	got := p.allowedOrigins(context.Background())
	assert.Equal(t, []string{"https://example.com"}, got, "duplicates must be deduped")
}

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
	assert.NotEmpty(t, c.UserID.String())
	assert.NotEmpty(t, c.AppID.String())
	assert.Equal(t, []byte{0x01, 0x02, 0x03}, c.CredentialID)
	assert.Equal(t, []byte{0xAA, 0xBB}, c.PublicKey)
	assert.Equal(t, "none", c.AttestationType)
	assert.Equal(t, []string{"internal"}, c.Transport)
	assert.Equal(t, uint32(42), c.SignCount)
	assert.Equal(t, "My Passkey", c.DisplayName)
}
