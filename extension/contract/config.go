package contract

import (
	"context"
	"net/http"
	"strings"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/plugins/social"
	"github.com/xraph/authsome/settings"

	dashauth "github.com/xraph/forge/extensions/dashboard/auth"
	"github.com/xraph/forge/extensions/dashboard/contract"
)

// AuthConfigResponse is what the React shell's AuthLoginForm reads to
// build the login UI. The shape mirrors shadcn's login-04 block: brand
// lockup, signup link, terms/privacy footer, password block (toggled by
// PasswordEnabled), and a list of social provider buttons. The shell
// renders the buttons in source order.
type AuthConfigResponse struct {
	Brand           string                 `json:"brand,omitempty"`
	BrandLogoURL    string                 `json:"brandLogoURL,omitempty"`
	PasswordEnabled bool                   `json:"passwordEnabled"`
	SignupURL       string                 `json:"signupURL,omitempty"`
	SignupLabel     string                 `json:"signupLabel,omitempty"`
	TermsURL        string                 `json:"termsURL,omitempty"`
	PrivacyURL      string                 `json:"privacyURL,omitempty"`
	SocialProviders []SocialProviderConfig `json:"socialProviders,omitempty"`
}

// SocialProviderConfig is a single button on the login form. AuthStartURL
// is the absolute URL the shell POSTs to begin the OAuth flow; the upstream
// endpoint replies with `{auth_url}` and the shell navigates the browser
// there. Authsome's social plugin already exposes a compatible endpoint
// at /v1/social/<provider>.
type SocialProviderConfig struct {
	ID           string `json:"id"`
	Label        string `json:"label"`
	AuthStartURL string `json:"authStartURL"`
}

// configHandler returns the auth.config query handler that powers the
// dashboard's login screen. It walks the social plugin's provider settings
// so enabling Apple/Google in authsome automatically lights them up on
// the form — no React change required.
func configHandler(deps Deps) func(ctx context.Context, _ struct{}, _ contract.Principal) (AuthConfigResponse, error) {
	return func(ctx context.Context, _ struct{}, _ contract.Principal) (AuthConfigResponse, error) {
		eng := deps.Engine
		if eng == nil {
			return AuthConfigResponse{}, &contract.Error{Code: contract.CodeUnavailable, Message: "auth engine not configured"}
		}

		out := AuthConfigResponse{
			PasswordEnabled: true,
			Brand:           deps.Brand,
			BrandLogoURL:    deps.BrandLogoURL,
			SignupURL:       strings.TrimSpace(deps.SignupURL),
			SignupLabel:     deps.SignupLabel,
			TermsURL:        strings.TrimSpace(deps.TermsURL),
			PrivacyURL:      strings.TrimSpace(deps.PrivacyURL),
		}

		// Default the signup link to the contract /signup graph route (always
		// registered in manifest.yaml) so the dashboard login form advertises
		// it out of the box. Operators override via the signup_url config —
		// point it at an external page, or unset it to keep this default.
		if out.SignupURL == "" {
			out.SignupURL = "/signup"
		}

		// Brand fallback: pull the platform app's display name + logo when
		// not explicitly set in Deps. Operators that want a custom label
		// override via Deps.Brand.
		if out.Brand == "" || out.BrandLogoURL == "" {
			if appID := defaultAppID(eng); !appID.IsNil() {
				if app, err := eng.Store().GetApp(ctx, appID); err == nil && app != nil {
					if out.Brand == "" && app.Name != "" {
						out.Brand = app.Name
					}
					if out.BrandLogoURL == "" && app.Logo != "" {
						out.BrandLogoURL = app.Logo
					}
				}
			}
		}
		if out.Brand == "" {
			out.Brand = "Forge Dashboard"
		}

		// Social providers: cascade-resolve at the platform app's scope so
		// app-level overrides land. Filter by Enabled so disabled rows
		// don't surface as buttons.
		providers := resolveSocialProviders(ctx, eng)
		req := dashauth.RequestFromContext(ctx)
		out.SocialProviders = projectSocialProviders(providers, req, deps.SocialBasePath)
		return out, nil
	}
}

// resolveSocialProviders reads the social.providers setting at the platform
// app's scope. Best-effort: returns nil when settings aren't wired (test
// envs) rather than failing the whole login screen.
func resolveSocialProviders(ctx context.Context, eng *authsome.Engine) []social.ProviderSetting {
	mgr := eng.Settings()
	if mgr == nil {
		return nil
	}
	opts := settings.ResolveOpts{}
	if appID := defaultAppID(eng); !appID.IsNil() {
		opts.AppID = appID.String()
	}
	providers, err := settings.Get(ctx, mgr, social.SettingSocialProviders, opts)
	if err != nil {
		return nil
	}
	return providers
}

// projectSocialProviders converts ProviderSetting records into the wire
// shape the React shell consumes. Disabled providers are filtered out.
// AuthStartURL is built absolute when we have a live request (so the
// shell can navigate the browser to the right host even when the
// dashboard is hosted on a different origin from authsome).
func projectSocialProviders(in []social.ProviderSetting, r *http.Request, socialBase string) []SocialProviderConfig {
	out := make([]SocialProviderConfig, 0, len(in))
	for _, p := range in {
		if !p.Enabled {
			continue
		}
		out = append(out, SocialProviderConfig{
			ID:           p.Name,
			Label:        socialLabel(p.Name),
			AuthStartURL: buildAuthStartURL(r, socialBase, p.Name),
		})
	}
	return out
}

// buildAuthStartURL composes the absolute URL the shell POSTs to start
// the OAuth flow. socialBase overrides the path prefix when authsome is
// mounted at a non-default location; defaults to /v1/social. Scheme/Host
// come from the live request when available, falling back to a relative
// path (which works when the shell shares an origin with authsome).
func buildAuthStartURL(r *http.Request, socialBase, providerName string) string {
	prefix := strings.TrimRight(socialBase, "/")
	if prefix == "" {
		prefix = "/v1/social"
	}
	path := prefix + "/" + providerName
	if r == nil {
		return path
	}
	scheme := "https"
	if r.TLS == nil && r.Header.Get("X-Forwarded-Proto") != "https" {
		scheme = "http"
	}
	host := r.Host
	if host == "" {
		return path
	}
	return scheme + "://" + host + path
}

// socialLabel turns a provider name like "google" into the button label
// "Continue with Google". Unknown providers get a Title-Cased name.
func socialLabel(name string) string {
	known := map[string]string{
		"google":    "Continue with Google",
		"apple":     "Continue with Apple",
		"github":    "Continue with GitHub",
		"microsoft": "Continue with Microsoft",
		"facebook":  "Continue with Facebook",
		"discord":   "Continue with Discord",
		"slack":     "Continue with Slack",
		"twitter":   "Continue with Twitter",
		"linkedin":  "Continue with LinkedIn",
		"gitlab":    "Continue with GitLab",
		"bitbucket": "Continue with Bitbucket",
	}
	if label, ok := known[strings.ToLower(name)]; ok {
		return label
	}
	if name == "" {
		return "Continue"
	}
	return "Continue with " + strings.ToUpper(name[:1]) + name[1:]
}
