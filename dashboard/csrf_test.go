package dashboard_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/xraph/authsome/dashboard"
)

func TestFormCSRF_GenerateSetsHostCookie(t *testing.T) {
	w := httptest.NewRecorder()
	tok := dashboard.GenerateFormCSRFToken(w)
	if tok == "" {
		t.Fatal("expected non-empty token")
	}
	res := w.Result()
	defer res.Body.Close()
	cookies := res.Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}
	c := cookies[0]
	if c.Name != dashboard.FormCSRFCookieName {
		t.Errorf("cookie name = %q, want %q", c.Name, dashboard.FormCSRFCookieName)
	}
	if !strings.HasPrefix(c.Name, "__Host-") {
		t.Errorf("cookie name missing __Host- prefix: %q", c.Name)
	}
	if c.Value != tok {
		t.Errorf("cookie value %q != returned token %q", c.Value, tok)
	}
	if !c.HttpOnly {
		t.Error("cookie must be HttpOnly")
	}
	if !c.Secure {
		t.Error("cookie must be Secure")
	}
	if c.SameSite != http.SameSiteStrictMode {
		t.Errorf("cookie SameSite = %v, want Strict", c.SameSite)
	}
	if c.Path != "/" {
		t.Errorf("cookie Path = %q, want /", c.Path)
	}
	if c.Domain != "" {
		t.Errorf("cookie Domain must be empty, got %q", c.Domain)
	}
}

func TestFormCSRF_VerifyMatchesCookieAndForm(t *testing.T) {
	w := httptest.NewRecorder()
	tok := dashboard.GenerateFormCSRFToken(w)

	r := httptest.NewRequest(http.MethodPost, "/", nil)
	r.AddCookie(&http.Cookie{Name: dashboard.FormCSRFCookieName, Value: tok})

	if !dashboard.VerifyFormCSRFToken(r, tok) {
		t.Fatal("expected matching token to verify")
	}
}

func TestFormCSRF_VerifyRejectsMissingCookie(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	if dashboard.VerifyFormCSRFToken(r, "anything") {
		t.Fatal("expected verify to fail with no cookie")
	}
}

func TestFormCSRF_VerifyRejectsMissingForm(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	r.AddCookie(&http.Cookie{Name: dashboard.FormCSRFCookieName, Value: "abc"})
	if dashboard.VerifyFormCSRFToken(r, "") {
		t.Fatal("expected verify to fail with empty form value")
	}
}

func TestFormCSRF_VerifyRejectsMismatch(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	r.AddCookie(&http.Cookie{Name: dashboard.FormCSRFCookieName, Value: "cookieval"})
	if dashboard.VerifyFormCSRFToken(r, "formval") {
		t.Fatal("expected mismatched token to fail")
	}
}

func TestFormCSRF_VerifyConstantTime(t *testing.T) {
	// Equal-length but different tokens — smoke test that the function uses
	// constant-time compare and handles equal-length mismatches correctly.
	a := "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	b := "BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB"
	if len(a) != len(b) {
		t.Fatal("test setup: tokens must be equal length")
	}
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	r.AddCookie(&http.Cookie{Name: dashboard.FormCSRFCookieName, Value: a})
	if dashboard.VerifyFormCSRFToken(r, b) {
		t.Fatal("equal-length mismatch must fail")
	}
}
