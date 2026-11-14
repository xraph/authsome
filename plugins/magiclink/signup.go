package magiclink

import (
	"context"
	"strings"

	"github.com/xraph/forge"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/user"
)

// captchaTokenHeader is the header the captcha middleware reads first.
// #nosec G101 -- not a credential: a header name.
const captchaTokenHeader = "X-Captcha-Token"

// signUpEnabled reports whether /send may create an account for the app:
// passwordless sign-up is on (Config.AllowSignUp or the
// magiclink.allow_signup setting), the app has sign-up enabled, and the
// plugin runs on an engine that can create the account.
func (p *Plugin) signUpEnabled(ctx context.Context, appID id.AppID) bool {
	if eng, ok := p.engine.(*authsome.Engine); !ok || eng == nil {
		return false
	}
	if !p.resolveBool(ctx, appID, SettingAllowSignUp, p.config.AllowSignUp) {
		return false
	}
	if cfg, err := p.store.GetAppClientConfig(ctx, appID); err == nil && cfg != nil &&
		cfg.SignupEnabled != nil && !*cfg.SignupEnabled {
		return false
	}
	return true
}

// checkSignUpCaptcha holds a sign-up to the captcha /v1/signup requires,
// when the app requires one. It answers the way the captcha middleware
// does, and runs before the address is looked up, so it says nothing about
// whether the address has an account.
func (p *Plugin) checkSignUpCaptcha(ctx forge.Context, appID id.AppID, bodyToken string) (bool, error) {
	if p.settingsMgr == nil {
		return false, nil
	}
	r := ctx.Request()
	// The handler has already read the body, so a token sent in it is put
	// where the verifier looks.
	if r.Header.Get(captchaTokenHeader) == "" && bodyToken != "" {
		r = r.Clone(ctx.Context())
		r.Header.Set(captchaTokenHeader, bodyToken)
	}
	opts := middleware.CaptchaOptions{Settings: p.settingsMgr, Action: "signup"}
	if p.engine != nil {
		opts.Logger = p.engine.Logger()
	}
	res := middleware.VerifyCaptchaForRequest(ctx.Context(), opts, r, appID, "signup")
	if res.Allowed {
		return false, nil
	}
	message := "captcha challenge required for this endpoint"
	switch res.RejectCode {
	case "captcha_invalid":
		message = "captcha verification failed"
	case "captcha_unavailable":
		message = "captcha verification is temporarily unavailable"
	case "captcha_misconfigured":
		message = "captcha provider is not configured correctly"
	}
	return true, ctx.JSON(res.RejectStatus, map[string]any{
		"error": message,
		"code":  res.RejectStatus,
		"type":  res.RejectCode,
	})
}

// signUp creates a passwordless account through the engine's sign-up path,
// so the sign-up hooks, the app's sign-up switch, email normalization and
// uniqueness all apply as they do for /v1/signup.
func (p *Plugin) signUp(ctx forge.Context, appID id.AppID, email, name string) (*user.User, error) {
	eng, ok := p.engine.(*authsome.Engine)
	if !ok || eng == nil {
		return nil, account.ErrSignUpDisabled
	}
	first, last := splitName(name)
	r := ctx.Request()
	return eng.SignUpPasswordless(ctx.Context(), &account.SignUpRequest{
		AppID:     appID,
		Email:     email,
		FirstName: first,
		LastName:  last,
		IPAddress: middleware.ClientIP(r),
		UserAgent: r.UserAgent(),
	})
}

// splitName splits a full name at its first space: the first word is the
// first name and the rest the last name, which User.Name joins back.
func splitName(name string) (first, last string) {
	name = strings.Join(strings.Fields(name), " ")
	first, last, _ = strings.Cut(name, " ")
	return first, last
}
