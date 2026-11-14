package magiclink_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/appclientconfig"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/secutil"
	"github.com/xraph/authsome/plugins/magiclink"
	"github.com/xraph/authsome/plugins/mfa"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/settings"
	"github.com/xraph/authsome/user"
)

// signUpProbe is a sign-up hook plugin, standing in for the waitlist and
// friends: it records what it sees and can refuse.
type signUpProbe struct {
	refuse error
	before []string
	after  []string
}

func (p *signUpProbe) Name() string { return "signup-probe" }

func (p *signUpProbe) OnBeforeSignUp(_ context.Context, req *account.SignUpRequest) error {
	p.before = append(p.before, req.Email)
	return p.refuse
}

func (p *signUpProbe) OnAfterSignUp(_ context.Context, u *user.User, _ *session.Session) error {
	p.after = append(p.after, u.Email)
	return nil
}

type engineFixture struct {
	eng    *authsome.Engine
	p      *magiclink.Plugin
	mailer *codeMailer
	h      http.Handler
}

func newEngineFixture(t *testing.T, cfg magiclink.Config, extra ...authsome.Option) engineFixture {
	t.Helper()
	mailer := &codeMailer{}
	cfg.Mailer = mailer
	if cfg.TokenTTL == 0 {
		cfg.TokenTTL = 5 * time.Minute
	}
	p := magiclink.New(cfg)
	opts := append([]authsome.Option{authsome.WithPlugin(p)}, extra...)
	eng := secutil.NewTestEngine(t, opts...)
	secutil.RelaxAuthDefaults(t, eng)
	p.SetAppID(testAppIDStr)
	mux := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(mux))
	return engineFixture{eng: eng, p: p, mailer: mailer, h: mux.Handler()}
}

func (f engineFixture) userByEmail(t *testing.T, email string) (*user.User, error) {
	t.Helper()
	appID, err := id.ParseAppID(testAppIDStr)
	require.NoError(t, err)
	return f.eng.Store().GetUserByAnyEmail(context.Background(), appID, id.Nil, email)
}

func TestSignUp_CreatesUserThroughSignUpHooks(t *testing.T) {
	probe := &signUpProbe{}
	f := newEngineFixture(t, magiclink.Config{EmailCode: true, AllowSignUp: true}, authsome.WithPlugin(probe))

	rec := post(t, f.h, "/v1/magic-link/send", map[string]any{
		"email": "  Ada@Example.com ", "create_user": true, "name": "Ada  King Lovelace",
	})
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	resp := decode(t, rec)
	assert.InDelta(t, 6, resp["code_length"], 0)

	assert.Equal(t, []string{"ada@example.com"}, probe.before, "BeforeSignUp runs, on the normalized address")
	assert.Equal(t, []string{"ada@example.com"}, probe.after, "AfterSignUp runs")

	u, err := f.userByEmail(t, "ada@example.com")
	require.NoError(t, err)
	assert.Empty(t, u.PasswordHash)
	assert.Equal(t, "Ada", u.FirstName)
	assert.Equal(t, "King Lovelace", u.LastName)
	assert.False(t, u.EmailVerified)

	msg := f.mailer.last(t)
	assert.Equal(t, magiclink.PurposeSignUp, msg.Purpose)
	assert.Equal(t, "Ada King Lovelace", msg.Name)

	// Entering the code proves the inbox and signs the new account in.
	verify := post(t, f.h, "/v1/magic-link/verify-code", map[string]string{"email": "ada@example.com", "code": msg.Code})
	require.Equal(t, http.StatusOK, verify.Code, "body=%s", verify.Body.String())
	assert.NotEmpty(t, decode(t, verify)["session_token"])

	u, err = f.userByEmail(t, "ada@example.com")
	require.NoError(t, err)
	assert.True(t, u.EmailVerified)

	// No password was set, so none signs in.
	_, _, err = f.eng.SignIn(context.Background(), &account.SignInRequest{AppID: u.AppID, Email: u.Email, Password: ""})
	assert.ErrorIs(t, err, account.ErrInvalidCredentials)
}

// The link verifies a new account as well as the code does, even where the
// app requires a verified email to sign in.
func TestSignUp_LinkVerifiesNewAccount(t *testing.T) {
	f := newEngineFixture(t, magiclink.Config{AllowSignUp: true})
	appID, err := id.ParseAppID(testAppIDStr)
	require.NoError(t, err)
	require.NoError(t, f.eng.Settings().Set(context.Background(), "auth.require_email_verification",
		json.RawMessage(`true`), settings.ScopeApp, appID.String(), appID.String(), "", "test"))

	require.Equal(t, http.StatusOK, post(t, f.h, "/v1/magic-link/send", map[string]any{"email": "linker@example.com", "create_user": true}).Code)
	msg := f.mailer.last(t)
	assert.Empty(t, msg.Code, "no code unless EmailCode is on")

	rec := post(t, f.h, "/v1/magic-link/verify", map[string]string{"token": msg.Token})
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
}

// An address that already has an account gets an ordinary sign-in link and
// the same response a new one does.
func TestSignUp_ExistingAccountGetsSignInLink(t *testing.T) {
	f := newEngineFixture(t, magiclink.Config{EmailCode: true, AllowSignUp: true})
	appID, err := id.ParseAppID(testAppIDStr)
	require.NoError(t, err)
	existing, _, err := f.eng.SignUp(context.Background(), &account.SignUpRequest{AppID: appID, Email: "old@example.com", Password: "SecureP@ss123"})
	require.NoError(t, err)
	secutil.VerifyEmail(t, f.eng, existing.ID)

	fresh := post(t, f.h, "/v1/magic-link/send", map[string]any{"email": "fresh@example.com", "create_user": true})
	old := post(t, f.h, "/v1/magic-link/send", map[string]any{"email": "old@example.com", "create_user": true, "name": "Someone Else"})
	require.Equal(t, http.StatusOK, fresh.Code)
	require.Equal(t, http.StatusOK, old.Code)
	assert.JSONEq(t, fresh.Body.String(), old.Body.String())

	msg := f.mailer.last(t)
	assert.Equal(t, "old@example.com", msg.Email)
	assert.Equal(t, magiclink.PurposeSignIn, msg.Purpose)

	u, err := f.userByEmail(t, "old@example.com")
	require.NoError(t, err)
	assert.Equal(t, existing.ID, u.ID, "no second account")
	assert.NotEmpty(t, u.PasswordHash, "the existing password is untouched")
	assert.Empty(t, u.FirstName, "the name in the request is ignored for an existing account")
}

func TestSignUp_RefusedWhenDisabled(t *testing.T) {
	t.Run("plugin flag off", func(t *testing.T) {
		probe := &signUpProbe{}
		f := newEngineFixture(t, magiclink.Config{EmailCode: true}, authsome.WithPlugin(probe))

		rec := post(t, f.h, "/v1/magic-link/send", map[string]any{"email": "new@example.com", "create_user": true})
		assert.Equal(t, http.StatusForbidden, rec.Code, "body=%s", rec.Body.String())
		_, err := f.userByEmail(t, "new@example.com")
		require.Error(t, err, "no account is created")
		assert.Empty(t, probe.before)
		assert.Zero(t, f.mailer.count())

		// Without create_user it is an ordinary send, unaffected.
		assert.Equal(t, http.StatusOK, post(t, f.h, "/v1/magic-link/send", map[string]any{"email": "new@example.com"}).Code)
	})

	t.Run("app sign-up off", func(t *testing.T) {
		f := newEngineFixture(t, magiclink.Config{EmailCode: true, AllowSignUp: true})
		appID, err := id.ParseAppID(testAppIDStr)
		require.NoError(t, err)
		disabled := false
		require.NoError(t, f.eng.Store().SetAppClientConfig(context.Background(), &appclientconfig.Config{AppID: appID, SignupEnabled: &disabled}))

		rec := post(t, f.h, "/v1/magic-link/send", map[string]any{"email": "new@example.com", "create_user": true})
		assert.Equal(t, http.StatusForbidden, rec.Code, "body=%s", rec.Body.String())
		_, err = f.userByEmail(t, "new@example.com")
		assert.Error(t, err, "no account is created")
	})

	t.Run("setting turns it on", func(t *testing.T) {
		f := newEngineFixture(t, magiclink.Config{})
		require.NoError(t, f.eng.Settings().Set(context.Background(), magiclink.SettingAllowSignUp.Def.Key,
			json.RawMessage(`true`), settings.ScopeGlobal, "", "", "", "test"))

		rec := post(t, f.h, "/v1/magic-link/send", map[string]any{"email": "via-setting@example.com", "create_user": true})
		require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
		_, err := f.userByEmail(t, "via-setting@example.com")
		assert.NoError(t, err)
	})
}

// A sign-up hook that refuses (the waitlist, say) refuses whether or not
// the address has an account, so the refusal reveals nothing.
func TestSignUp_HookRefusal(t *testing.T) {
	probe := &signUpProbe{refuse: errors.New("waitlist: signup requires waitlist approval")}
	f := newEngineFixture(t, magiclink.Config{AllowSignUp: true}, authsome.WithPlugin(probe))

	rec := post(t, f.h, "/v1/magic-link/send", map[string]any{"email": "queued@example.com", "create_user": true})
	assert.Equal(t, http.StatusForbidden, rec.Code, "body=%s", rec.Body.String())
	_, err := f.userByEmail(t, "queued@example.com")
	require.Error(t, err)
	assert.Zero(t, f.mailer.count())
}

// The captcha /v1/signup asks for is asked for here too, and only for a
// sign-up.
func TestSignUp_CaptchaRequired(t *testing.T) {
	f := newEngineFixture(t, magiclink.Config{AllowSignUp: true})
	require.NoError(t, f.eng.Settings().Set(context.Background(), "auth.captcha_required",
		json.RawMessage(`true`), settings.ScopeGlobal, "", "", "", "test"))

	rec := post(t, f.h, "/v1/magic-link/send", map[string]any{"email": "bot@example.com", "create_user": true})
	assert.Equal(t, http.StatusForbidden, rec.Code, "body=%s", rec.Body.String())
	assert.Equal(t, "captcha_required", decode(t, rec)["type"])
	_, err := f.userByEmail(t, "bot@example.com")
	require.Error(t, err)

	assert.Equal(t, http.StatusOK, post(t, f.h, "/v1/magic-link/send", map[string]any{"email": "bot@example.com"}).Code,
		"a sign-in send isn't held for the captcha")
}

// The code goes through IssueSession like the link, so a user with a second
// factor gets the MFA challenge, not a session.
func TestVerifyCode_MFAGateFires(t *testing.T) {
	mfaStore := mfa.NewMemoryStore()
	m := mfa.New()
	m.SetStore(mfaStore)
	f := newEngineFixture(t, magiclink.Config{EmailCode: true}, authsome.WithPlugin(m))

	appID, err := id.ParseAppID(testAppIDStr)
	require.NoError(t, err)
	u, _, err := f.eng.SignUp(context.Background(), &account.SignUpRequest{AppID: appID, Email: "two-step@example.com", Password: "SecureP@ss123"})
	require.NoError(t, err)
	key, err := mfa.GenerateTOTPKey(mfa.TOTPConfig{Issuer: "Test", AccountName: u.Email})
	require.NoError(t, err)
	require.NoError(t, mfaStore.CreateEnrollment(context.Background(), &mfa.Enrollment{
		ID: id.NewMFAID(), UserID: u.ID, Method: "totp", Secret: key.Secret(), Verified: true,
	}))

	require.Equal(t, http.StatusOK, post(t, f.h, "/v1/magic-link/send", map[string]string{"email": u.Email}).Code)
	rec := post(t, f.h, "/v1/magic-link/verify-code", map[string]string{"email": u.Email, "code": f.mailer.last(t).Code})
	require.Equal(t, http.StatusForbidden, rec.Code, "body=%s", rec.Body.String())
	resp := decode(t, rec)
	assert.Equal(t, "mfa_required", resp["type"])
	assert.NotEmpty(t, resp["mfa_ticket"])
	assert.NotContains(t, resp, "session_token")
}
