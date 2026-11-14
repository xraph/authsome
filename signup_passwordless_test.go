package authsome_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/appclientconfig"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/user"
)

// signUpHookPlugin records the sign-up hooks it sees and can refuse.
type signUpHookPlugin struct {
	refuse error
	before []string
	after  []string
}

func (p *signUpHookPlugin) Name() string { return "signup-hook-probe" }

func (p *signUpHookPlugin) OnBeforeSignUp(_ context.Context, req *account.SignUpRequest) error {
	p.before = append(p.before, req.Email)
	return p.refuse
}

func (p *signUpHookPlugin) OnAfterSignUp(_ context.Context, u *user.User, s *session.Session) error {
	p.after = append(p.after, u.Email)
	if s != nil {
		return errors.New("passwordless sign-up must not mint a session")
	}
	return nil
}

func TestSignUpPasswordless_CreatesUserThroughSignUpHooks(t *testing.T) {
	probe := &signUpHookPlugin{}
	eng, st := newTestEngine(t, authsome.WithPlugin(probe))
	ctx := context.Background()

	u, err := eng.SignUpPasswordless(ctx, &account.SignUpRequest{
		AppID:     testAppID(t),
		Email:     "  New.Person@Example.com ",
		Password:  "ignored-on-this-path",
		FirstName: "New",
	})
	require.NoError(t, err)
	assert.Equal(t, "new.person@example.com", u.Email, "the address is normalized like /v1/signup")
	assert.Empty(t, u.PasswordHash, "no password is set")
	assert.False(t, u.EmailVerified, "the inbox is proven by the link, later")
	assert.Equal(t, []string{"new.person@example.com"}, probe.before)
	assert.Equal(t, []string{"new.person@example.com"}, probe.after)

	stored, err := st.GetUser(ctx, u.ID)
	require.NoError(t, err)
	assert.Empty(t, stored.PasswordHash)

	sessions, err := st.ListUserSessions(ctx, u.ID)
	require.NoError(t, err)
	assert.Empty(t, sessions, "no session until the link or code comes back")

	_, err = st.GetActiveEmailVerification(ctx, u.ID)
	assert.Error(t, err, "no email-verification code: the magic link proves the inbox")
}

func TestSignUpPasswordless_HookRefusalIsSignUpRefused(t *testing.T) {
	probe := &signUpHookPlugin{refuse: errors.New("waitlist: join the waitlist first")}
	eng, _ := newTestEngine(t, authsome.WithPlugin(probe))

	_, err := eng.SignUpPasswordless(context.Background(), &account.SignUpRequest{
		AppID: testAppID(t), Email: "waiting@example.com",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, account.ErrSignUpRefused)
	assert.Contains(t, err.Error(), "waitlist")
}

func TestSignUpPasswordless_RefusedWhenSignUpDisabled(t *testing.T) {
	eng, st := newTestEngine(t)
	ctx := context.Background()
	disabled := false
	require.NoError(t, st.SetAppClientConfig(ctx, &appclientconfig.Config{AppID: testAppID(t), SignupEnabled: &disabled}))

	_, err := eng.SignUpPasswordless(ctx, &account.SignUpRequest{AppID: testAppID(t), Email: "nope@example.com"})
	assert.ErrorIs(t, err, account.ErrSignUpDisabled)

	// The password path reports the same sentinel.
	_, _, err = eng.SignUp(ctx, &account.SignUpRequest{AppID: testAppID(t), Email: "nope@example.com", Password: "SecureP@ss123"})
	assert.ErrorIs(t, err, account.ErrSignUpDisabled)
}

func TestSignUpPasswordless_DuplicateIsEmailTaken(t *testing.T) {
	eng, _ := newTestEngine(t)
	signUpTestUser(t, eng, "taken@example.com", "SecureP@ss123")

	_, err := eng.SignUpPasswordless(context.Background(), &account.SignUpRequest{AppID: testAppID(t), Email: "TAKEN@example.com"})
	assert.ErrorIs(t, err, account.ErrEmailTaken)
}

// An account with no password can't be signed in with an empty one, or any
// other, until forgot and reset password give it one.
func TestSignIn_PasswordlessAccountNeedsResetFirst(t *testing.T) {
	eng, _ := newTestEngine(t)
	ctx := context.Background()
	appID := testAppID(t)

	u, err := eng.SignUpPasswordless(ctx, &account.SignUpRequest{AppID: appID, Email: "nopass@example.com"})
	require.NoError(t, err)

	for _, pw := range []string{"", "anything-at-all"} {
		_, _, err = eng.SignIn(ctx, &account.SignInRequest{AppID: appID, Email: "nopass@example.com", Password: pw})
		assert.ErrorIs(t, err, account.ErrInvalidCredentials, "password %q", pw)
	}

	pr, err := eng.ForgotPassword(ctx, appID, "nopass@example.com")
	require.NoError(t, err)
	require.NotNil(t, pr, "a passwordless account can ask for a reset")
	require.NoError(t, eng.ResetPassword(ctx, pr.Token, "BrandNewP@ss123"))

	signedIn, sess, err := eng.SignIn(ctx, &account.SignInRequest{AppID: appID, Email: "nopass@example.com", Password: "BrandNewP@ss123"})
	require.NoError(t, err)
	assert.Equal(t, u.ID, signedIn.ID)
	assert.NotNil(t, sess)
}

func TestMarkEmailVerified_PersistsAndIsIdempotent(t *testing.T) {
	eng, st := newTestEngine(t)
	ctx := context.Background()

	u, err := eng.SignUpPasswordless(ctx, &account.SignUpRequest{AppID: testAppID(t), Email: "proved@example.com"})
	require.NoError(t, err)

	require.NoError(t, eng.MarkEmailVerified(ctx, u))
	stored, err := st.GetUser(ctx, u.ID)
	require.NoError(t, err)
	assert.True(t, stored.EmailVerified)

	require.NoError(t, eng.MarkEmailVerified(ctx, u), "a second call is a no-op")
	require.NoError(t, eng.MarkEmailVerified(ctx, nil))
}
