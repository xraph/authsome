package email

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/user"
)

// ──────────────────────────────────────────────────
// Mock mailer
// ──────────────────────────────────────────────────

type mockMailer struct {
	mu    sync.Mutex
	calls []*bridge.EmailMessage
}

func (m *mockMailer) SendEmail(_ context.Context, msg *bridge.EmailMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, msg)
	return nil
}

func (m *mockMailer) lastMessage() *bridge.EmailMessage {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.calls) == 0 {
		return nil
	}
	return m.calls[len(m.calls)-1]
}

func (m *mockMailer) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.calls)
}

// ──────────────────────────────────────────────────
// Plugin basics
// ──────────────────────────────────────────────────

func TestPlugin_Name(t *testing.T) {
	p := New(Config{})
	assert.Equal(t, "email", p.Name())
}

func TestPlugin_Defaults(t *testing.T) {
	p := New(Config{})
	assert.Equal(t, "AuthSome", p.config.AppName)
	assert.Equal(t, "noreply@authsome.local", p.config.From)
}

// ──────────────────────────────────────────────────
// AfterSignUp
// ──────────────────────────────────────────────────

func TestOnAfterSignUp_SendsWelcomeEmail(t *testing.T) {
	mailer := &mockMailer{}
	p := New(Config{
		From:    "test@example.com",
		AppName: "TestApp",
	})
	p.SetMailer(mailer)

	u := &user.User{
		ID:    id.NewUserID(),
		Email: "alice@example.com",
		FirstName: "Alice",
	}

	err := p.OnAfterSignUp(context.Background(), u, &session.Session{})
	require.NoError(t, err)
	assert.Equal(t, 1, mailer.count())

	msg := mailer.lastMessage()
	assert.Equal(t, []string{"alice@example.com"}, msg.To)
	assert.Equal(t, "test@example.com", msg.From)
	assert.Contains(t, msg.Subject, "Welcome to TestApp")
	assert.Contains(t, msg.HTML, "Alice")
	assert.Contains(t, msg.Text, "Alice")
}

func TestOnAfterSignUp_NoMailer_Noop(t *testing.T) {
	p := New(Config{})
	// No mailer set

	err := p.OnAfterSignUp(context.Background(), &user.User{}, &session.Session{})
	require.NoError(t, err)
}

func TestOnAfterSignUp_FallsBackToEmail(t *testing.T) {
	mailer := &mockMailer{}
	p := New(Config{AppName: "App"})
	p.SetMailer(mailer)

	u := &user.User{
		ID:    id.NewUserID(),
		Email: "bob@example.com",
		FirstName: "", // empty name
	}

	err := p.OnAfterSignUp(context.Background(), u, &session.Session{})
	require.NoError(t, err)

	msg := mailer.lastMessage()
	assert.Contains(t, msg.HTML, "bob@example.com")
}

// ──────────────────────────────────────────────────
// AfterUserCreate
// ──────────────────────────────────────────────────

func TestOnAfterUserCreate_SendsVerificationEmail(t *testing.T) {
	mailer := &mockMailer{}
	p := New(Config{
		AppName: "TestApp",
		BaseURL: "https://example.com",
	})
	p.SetMailer(mailer)

	u := &user.User{
		ID:            id.NewUserID(),
		Email:         "carol@example.com",
		FirstName:     "Carol",
		EmailVerified: false,
	}

	err := p.OnAfterUserCreate(context.Background(), u)
	require.NoError(t, err)
	assert.Equal(t, 1, mailer.count())

	msg := mailer.lastMessage()
	assert.Contains(t, msg.Subject, "Verify")
	assert.Contains(t, msg.HTML, "https://example.com/verify-email")
}

func TestOnAfterUserCreate_SkipsVerifiedEmail(t *testing.T) {
	mailer := &mockMailer{}
	p := New(Config{})
	p.SetMailer(mailer)

	u := &user.User{
		ID:            id.NewUserID(),
		Email:         "verified@example.com",
		EmailVerified: true,
	}

	err := p.OnAfterUserCreate(context.Background(), u)
	require.NoError(t, err)
	assert.Equal(t, 0, mailer.count())
}

// ──────────────────────────────────────────────────
// Templates
// ──────────────────────────────────────────────────

func TestWelcomeEmail(t *testing.T) {
	subject, html, text := WelcomeEmail("Alice", "TestApp")
	assert.Equal(t, "Welcome to TestApp", subject)
	assert.Contains(t, html, "Alice")
	assert.Contains(t, html, "TestApp")
	assert.Contains(t, text, "Alice")
	assert.Contains(t, text, "TestApp")
}

func TestVerificationEmail(t *testing.T) {
	subject, html, text := VerificationEmail("Bob", "TestApp", "https://example.com/verify")
	assert.Contains(t, subject, "Verify")
	assert.Contains(t, html, "https://example.com/verify")
	assert.Contains(t, html, "Bob")
	assert.Contains(t, text, "https://example.com/verify")
}

func TestPasswordResetEmail(t *testing.T) {
	subject, html, text := PasswordResetEmail("Carol", "TestApp", "https://example.com/reset")
	assert.Contains(t, subject, "Reset")
	assert.Contains(t, html, "https://example.com/reset")
	assert.Contains(t, html, "Carol")
	assert.Contains(t, text, "https://example.com/reset")
}

func TestInvitationEmail(t *testing.T) {
	subject, html, text := InvitationEmail("Dan", "Acme Corp", "TestApp", "https://example.com/accept")
	assert.Contains(t, subject, "Acme Corp")
	assert.Contains(t, html, "Dan")
	assert.Contains(t, html, "Acme Corp")
	assert.Contains(t, html, "https://example.com/accept")
	assert.Contains(t, text, "https://example.com/accept")
}

// ──────────────────────────────────────────────────
// SendPasswordReset / SendInvitation
// ──────────────────────────────────────────────────

func TestSendPasswordReset(t *testing.T) {
	mailer := &mockMailer{}
	p := New(Config{AppName: "App"})
	p.SetMailer(mailer)

	err := p.SendPasswordReset(context.Background(), "test@example.com", "Test User", "https://reset.url")
	require.NoError(t, err)

	msg := mailer.lastMessage()
	assert.Equal(t, []string{"test@example.com"}, msg.To)
	assert.Contains(t, msg.Subject, "Reset")
}

func TestSendPasswordReset_NoMailer(t *testing.T) {
	p := New(Config{})

	err := p.SendPasswordReset(context.Background(), "test@example.com", "User", "https://reset.url")
	assert.ErrorIs(t, err, bridge.ErrMailerNotAvailable)
}

func TestSendInvitation(t *testing.T) {
	mailer := &mockMailer{}
	p := New(Config{AppName: "App"})
	p.SetMailer(mailer)

	err := p.SendInvitation(context.Background(), "invite@example.com", "Dan", "Acme", "https://accept.url")
	require.NoError(t, err)

	msg := mailer.lastMessage()
	assert.Equal(t, []string{"invite@example.com"}, msg.To)
	assert.Contains(t, msg.Subject, "Acme")
}
