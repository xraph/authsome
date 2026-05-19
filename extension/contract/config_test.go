package contract

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/xraph/authsome/plugins/social"

	dashcontract "github.com/xraph/forge/extensions/dashboard/contract"
)

func TestConfigHandler_NilEngineUnavailable(t *testing.T) {
	ctx, _, _ := withHTTPCtx(t)
	h := configHandler(Deps{Engine: nil})
	_, err := h(ctx, struct{}{}, dashcontract.Principal{})
	var ce *dashcontract.Error
	if !errors.As(err, &ce) || ce.Code != dashcontract.CodeUnavailable {
		t.Errorf("expected CodeUnavailable, got %v", err)
	}
}

func TestSocialLabel_Known(t *testing.T) {
	cases := map[string]string{
		"google":    "Continue with Google",
		"apple":     "Continue with Apple",
		"github":    "Continue with GitHub",
		"microsoft": "Continue with Microsoft",
	}
	for in, want := range cases {
		if got := socialLabel(in); got != want {
			t.Errorf("socialLabel(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSocialLabel_Unknown(t *testing.T) {
	if got := socialLabel("custom-sso"); got != "Continue with Custom-sso" {
		t.Errorf("unknown provider label = %q", got)
	}
	if got := socialLabel(""); got != "Continue" {
		t.Errorf("empty provider label = %q", got)
	}
}

func TestProjectSocialProviders_FiltersDisabled(t *testing.T) {
	in := []social.ProviderSetting{
		{Name: "google", Enabled: true},
		{Name: "apple", Enabled: false},
		{Name: "github", Enabled: true},
	}
	out := projectSocialProviders(in, nil, "")
	if len(out) != 2 {
		t.Fatalf("expected 2 enabled providers, got %d", len(out))
	}
	if out[0].ID != "google" || out[1].ID != "github" {
		t.Errorf("unexpected projection: %+v", out)
	}
}

func TestProjectSocialProviders_LabelsAndPaths(t *testing.T) {
	in := []social.ProviderSetting{{Name: "google", Enabled: true}}
	out := projectSocialProviders(in, nil, "")
	if out[0].Label != "Continue with Google" {
		t.Errorf("label = %q", out[0].Label)
	}
	if out[0].AuthStartURL != "/v1/social/google" {
		t.Errorf("default path = %q", out[0].AuthStartURL)
	}
}

func TestProjectSocialProviders_AbsoluteURLFromRequest(t *testing.T) {
	r := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "https://dashboard.example.com/some/path", nil)
	r.Host = "dashboard.example.com"
	in := []social.ProviderSetting{{Name: "google", Enabled: true}}
	out := projectSocialProviders(in, r, "/v1/social")
	want := "https://dashboard.example.com/v1/social/google"
	if out[0].AuthStartURL != want {
		t.Errorf("auth start URL = %q, want %q", out[0].AuthStartURL, want)
	}
}

func TestProjectSocialProviders_HonoursSocialBaseOverride(t *testing.T) {
	r := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "https://example.com/", nil)
	r.Host = "example.com"
	in := []social.ProviderSetting{{Name: "github", Enabled: true}}
	out := projectSocialProviders(in, r, "/auth/social")
	if out[0].AuthStartURL != "https://example.com/auth/social/github" {
		t.Errorf("override base ignored: %q", out[0].AuthStartURL)
	}
}
