package auth_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"

	"github.com/xraph/authsome/dashboard/auth"
)

// mustRender renders a templ.Component to a string, failing the test on
// render error so callers can match against substrings without nil checks.
func mustRender(t *testing.T, c templ.Component) string {
	t.Helper()
	var buf bytes.Buffer
	if err := c.Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	return buf.String()
}

// assertHiddenCSRFField asserts that the rendered HTML contains a hidden
// csrf_token input bound to the supplied value.
func assertHiddenCSRFField(t *testing.T, html, token string) {
	t.Helper()
	if !strings.Contains(html, `name="csrf_token"`) {
		t.Errorf("rendered HTML missing csrf_token field; got:\n%s", html)
	}
	if !strings.Contains(html, `type="hidden"`) {
		t.Errorf("csrf_token field is not hidden; got:\n%s", html)
	}
	if !strings.Contains(html, token) {
		t.Errorf("rendered HTML missing CSRF token value %q; got:\n%s", token, html)
	}
}

func TestLoginPage_RendersHiddenCSRFField(t *testing.T) {
	const tok = "tok-login-123"
	html := mustRender(t, auth.LoginPage(auth.LoginPageLinks{CSRFToken: tok}))
	assertHiddenCSRFField(t, html, tok)
}

func TestLoginError_RendersHiddenCSRFField(t *testing.T) {
	const tok = "tok-login-err-456"
	html := mustRender(t, auth.LoginError("bad creds", auth.LoginPageLinks{CSRFToken: tok}))
	assertHiddenCSRFField(t, html, tok)
}

func TestRegisterPage_RendersHiddenCSRFField(t *testing.T) {
	const tok = "tok-register-789"
	html := mustRender(t, auth.RegisterPage(auth.RegisterPageLinks{CSRFToken: tok}))
	assertHiddenCSRFField(t, html, tok)
}

func TestRegisterError_RendersHiddenCSRFField(t *testing.T) {
	const tok = "tok-register-err-abc"
	html := mustRender(t, auth.RegisterError("nope", auth.RegisterPageLinks{CSRFToken: tok}))
	assertHiddenCSRFField(t, html, tok)
}

func TestForgotPasswordPage_RendersHiddenCSRFField(t *testing.T) {
	const tok = "tok-forgot-def"
	html := mustRender(t, auth.ForgotPasswordPage(auth.ForgotPasswordPageLinks{CSRFToken: tok}))
	assertHiddenCSRFField(t, html, tok)
}

func TestForgotPasswordError_RendersHiddenCSRFField(t *testing.T) {
	const tok = "tok-forgot-err-ghi"
	html := mustRender(t, auth.ForgotPasswordError("bad", auth.ForgotPasswordPageLinks{CSRFToken: tok}))
	assertHiddenCSRFField(t, html, tok)
}

func TestSetupPage_RendersHiddenCSRFField(t *testing.T) {
	const tok = "tok-setup-jkl"
	html := mustRender(t, auth.SetupPage(auth.SetupPageLinks{CSRFToken: tok}))
	assertHiddenCSRFField(t, html, tok)
}

func TestSetupError_RendersHiddenCSRFField(t *testing.T) {
	const tok = "tok-setup-err-mno"
	html := mustRender(t, auth.SetupError("err", auth.SetupPageLinks{CSRFToken: tok}))
	assertHiddenCSRFField(t, html, tok)
}

func TestDynamicRegisterPage_RendersHiddenCSRFField(t *testing.T) {
	const tok = "tok-dynamic-pqr"
	html := mustRender(t, auth.DynamicRegisterPage(auth.DynamicRegisterProps{CSRFToken: tok}))
	assertHiddenCSRFField(t, html, tok)
}
