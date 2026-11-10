// handlers_context.go: Phase C.17 — App + environment switcher.
//
// Three intents back the dashboard's switcher chrome:
//
//   - apps.context (query): returns the active app + environment plus
//     the lists of available choices. The shell renders the switcher
//     dropdowns from this single query and re-fetches after a
//     successful switch command.
//   - apps.switch (command): records the operator's choice of app by
//     writing an `authsome_app` cookie.
//   - environments.switch (command): same shape, writes
//     `authsome_env` with the environment ID.
//
// The cookies are a request, not a grant. The switch commands only
// check that the ID names a real app or environment. The decision is
// made on every later request by the dashboard's auth checker
// (authChecker.CheckAuth in the extension package), which reads the
// cookies back and projects them onto the principal's "app_id" and
// "env_id" claims only when the session is allowed to act there:
//
//   - authsome_app is honoured when it names the session's own app, or
//     when the session belongs to the platform app and holds
//     platform-admin or platform-owner. Anything else is ignored and
//     the claim stays on the session's app. An unverified cookie never
//     widens what an operator can see.
//   - authsome_env is honoured only when the environment belongs to
//     whichever app was resolved above.
//
// So a hand-written cookie gets the operator exactly what they had
// without it. Handlers read the result through AppIDFromPrincipal and
// envIDFromPrincipal, never from the cookies directly.
package contract

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/xraph/authsome/app"
	"github.com/xraph/authsome/environment"
	"github.com/xraph/authsome/id"

	dashauth "github.com/xraph/forge/extensions/dashboard/auth"
	"github.com/xraph/forge/extensions/dashboard/contract"
)

// Cookie names the switch commands write and the dashboard's auth
// checker reads back. Exported so both sides share one spelling.
const (
	AppSwitcherCookie = "authsome_app"
	EnvSwitcherCookie = "authsome_env"
)

// AppContextResponse is the apps.context query shape. Field names
// match the React shell's switcher hook bindings — renames are wire
// breaks.
type AppContextResponse struct {
	// CurrentApp is the active app the principal is scoped to. May be
	// nil for unauthenticated requests or when no platform app exists.
	CurrentApp *SwitcherApp `json:"currentApp,omitempty"`
	// CurrentEnv is the active environment (when set via cookie).
	CurrentEnv *SwitcherEnv `json:"currentEnv,omitempty"`
	// AvailableApps is the global list of apps the switcher renders.
	// Filtering by user permissions can land in a later iteration.
	AvailableApps []SwitcherApp `json:"availableApps"`
	// AvailableEnvs is the list of environments for the current app.
	AvailableEnvs []SwitcherEnv `json:"availableEnvs"`
}

type SwitcherApp struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Slug       string `json:"slug"`
	Logo       string `json:"logo,omitempty"`
	IsPlatform bool   `json:"isPlatform"`
}

type SwitcherEnv struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	Type      string `json:"type,omitempty"`
	IsDefault bool   `json:"isDefault"`
}

// AppSwitchInput / EnvSwitchInput are the wire shapes for the switch
// commands. The empty-string sentinel clears the cookie (resets to
// the default app/env).
type AppSwitchInput struct {
	AppID string `json:"appId,omitempty"`
}

type EnvSwitchInput struct {
	EnvID string `json:"envId,omitempty"`
}

type SwitchResponse struct {
	OK bool `json:"ok"`
}

// ────────────────────────────────────────────────────────────────────
// Handlers
// ────────────────────────────────────────────────────────────────────

func appsContextHandler(deps Deps) func(ctx context.Context, _ struct{}, p contract.Principal) (AppContextResponse, error) {
	return func(ctx context.Context, _ struct{}, p contract.Principal) (AppContextResponse, error) {
		eng := deps.Engine
		if eng == nil {
			return AppContextResponse{}, &contract.Error{Code: contract.CodeUnavailable, Message: "auth engine not configured"}
		}

		apps, err := eng.ListApps(ctx)
		if err != nil {
			return AppContextResponse{}, mapEngineError(err)
		}
		out := AppContextResponse{
			AvailableApps: make([]SwitcherApp, 0, len(apps)),
		}
		for _, a := range apps {
			out.AvailableApps = append(out.AvailableApps, projectSwitcherApp(a))
		}

		// Resolve the active app via the principal's app_id claim, which
		// the dashboard's auth checker sets from the session, or from the
		// authsome_app cookie when the session may act in that app.
		appID := AppIDFromPrincipal(p, eng)
		if !appID.IsNil() {
			if a, gerr := eng.GetApp(ctx, appID); gerr == nil && a != nil {
				cur := projectSwitcherApp(a)
				out.CurrentApp = &cur
			}
		}

		// Environments are scoped to the current app; the switcher's
		// env dropdown rebuilds whenever the app changes.
		if !appID.IsNil() {
			envs, eerr := eng.ListEnvironments(ctx, appID)
			if eerr == nil {
				out.AvailableEnvs = make([]SwitcherEnv, 0, len(envs))
				for _, e := range envs {
					out.AvailableEnvs = append(out.AvailableEnvs, projectSwitcherEnv(e))
				}
				// Resolve the active env from the principal claim with
				// fallback to the app's default environment.
				if envID := envIDFromPrincipal(p); !envID.IsNil() {
					for _, e := range envs {
						if e.ID == envID {
							cur := projectSwitcherEnv(e)
							out.CurrentEnv = &cur
							break
						}
					}
				}
				if out.CurrentEnv == nil {
					for _, e := range envs {
						if e.IsDefault {
							cur := projectSwitcherEnv(e)
							out.CurrentEnv = &cur
							break
						}
					}
				}
			}
		}

		return out, nil
	}
}

func appsSwitchHandler(deps Deps) func(ctx context.Context, in AppSwitchInput, _ contract.Principal) (SwitchResponse, error) {
	return func(ctx context.Context, in AppSwitchInput, _ contract.Principal) (SwitchResponse, error) {
		if deps.Engine == nil {
			return SwitchResponse{}, &contract.Error{Code: contract.CodeUnavailable, Message: "auth engine not configured"}
		}
		httpReq := dashauth.RequestFromContext(ctx)
		httpRes := dashauth.ResponseWriterFromContext(ctx)
		if httpRes == nil || httpReq == nil {
			return SwitchResponse{}, &contract.Error{Code: contract.CodeInternal, Message: "no http context (forge >= dashauth.WithHTTP required)"}
		}

		raw := strings.TrimSpace(in.AppID)
		if raw == "" {
			clearSwitcherCookie(httpRes, httpReq, AppSwitcherCookie, secureForRequest(httpReq, deps))
			return SwitchResponse{OK: true}, nil
		}
		// Validate the ID parses and references an existing app so
		// callers don't poison their own cookie with junk.
		appID, err := id.ParseAppID(raw)
		if err != nil {
			return SwitchResponse{}, &contract.Error{Code: contract.CodeBadRequest, Message: "invalid app id: " + err.Error()}
		}
		if _, err := deps.Engine.GetApp(ctx, appID); err != nil {
			return SwitchResponse{}, mapEngineError(err)
		}
		writeSwitcherCookie(httpRes, httpReq, AppSwitcherCookie, appID.String(), secureForRequest(httpReq, deps))
		// Switching apps invalidates any env selection — the new app
		// has a different env list. Clear the env cookie alongside.
		clearSwitcherCookie(httpRes, httpReq, EnvSwitcherCookie, secureForRequest(httpReq, deps))
		return SwitchResponse{OK: true}, nil
	}
}

func environmentsSwitchHandler(deps Deps) func(ctx context.Context, in EnvSwitchInput, p contract.Principal) (SwitchResponse, error) {
	return func(ctx context.Context, in EnvSwitchInput, p contract.Principal) (SwitchResponse, error) {
		if deps.Engine == nil {
			return SwitchResponse{}, &contract.Error{Code: contract.CodeUnavailable, Message: "auth engine not configured"}
		}
		httpReq := dashauth.RequestFromContext(ctx)
		httpRes := dashauth.ResponseWriterFromContext(ctx)
		if httpRes == nil || httpReq == nil {
			return SwitchResponse{}, &contract.Error{Code: contract.CodeInternal, Message: "no http context (forge >= dashauth.WithHTTP required)"}
		}

		raw := strings.TrimSpace(in.EnvID)
		if raw == "" {
			clearSwitcherCookie(httpRes, httpReq, EnvSwitcherCookie, secureForRequest(httpReq, deps))
			return SwitchResponse{OK: true}, nil
		}
		envID, err := id.ParseEnvironmentID(raw)
		if err != nil {
			return SwitchResponse{}, &contract.Error{Code: contract.CodeBadRequest, Message: "invalid env id: " + err.Error()}
		}
		// Validate the env exists and belongs to the principal's current
		// app — guards against a stale envID from a previous app switch.
		env, err := deps.Engine.GetEnvironment(ctx, envID)
		if err != nil {
			return SwitchResponse{}, mapEngineError(err)
		}
		if appID := AppIDFromPrincipal(p, deps.Engine); !appID.IsNil() && env.AppID != appID {
			return SwitchResponse{}, &contract.Error{Code: contract.CodeBadRequest, Message: "environment does not belong to the current app"}
		}
		writeSwitcherCookie(httpRes, httpReq, EnvSwitcherCookie, envID.String(), secureForRequest(httpReq, deps))
		return SwitchResponse{OK: true}, nil
	}
}

// ────────────────────────────────────────────────────────────────────
// Projections + helpers
// ────────────────────────────────────────────────────────────────────

func projectSwitcherApp(a *app.App) SwitcherApp {
	if a == nil {
		return SwitcherApp{}
	}
	return SwitcherApp{
		ID:         a.ID.String(),
		Name:       a.Name,
		Slug:       a.Slug,
		Logo:       a.Logo,
		IsPlatform: a.IsPlatform,
	}
}

func projectSwitcherEnv(e *environment.Environment) SwitcherEnv {
	if e == nil {
		return SwitcherEnv{}
	}
	return SwitcherEnv{
		ID:        e.ID.String(),
		Name:      e.Name,
		Slug:      e.Slug,
		Type:      string(e.Type),
		IsDefault: e.IsDefault,
	}
}

// envIDFromPrincipal reads the env_id claim. The dashboard's auth
// checker sets it from the authsome_env cookie when that environment
// belongs to the resolved app, and otherwise from the session's own
// environment. Returns the zero EnvironmentID when the claim is missing
// or unparseable.
func envIDFromPrincipal(p contract.Principal) id.EnvironmentID {
	raw, ok := p.Claims["env_id"].(string)
	if !ok || raw == "" {
		return id.EnvironmentID{}
	}
	parsed, err := id.ParseEnvironmentID(raw)
	if err != nil {
		return id.EnvironmentID{}
	}
	return parsed
}

// writeSwitcherCookie sets a switcher cookie matching the
// session-cookie attribute style: Path=/, SameSite=Lax, HttpOnly,
// Secure when the request is TLS. Lifetime is a year — switchers
// persist across browser sessions.
func writeSwitcherCookie(w http.ResponseWriter, r *http.Request, name, value string, secure bool) {
	http.SetCookie(w, &http.Cookie{ // #nosec G124 -- attributes set below
		Name:     name,
		Value:    value,
		Path:     "/",
		Expires:  time.Now().Add(365 * 24 * time.Hour),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secure,
	})
	_ = r // request kept on the signature for symmetry with setSessionCookie
}

func clearSwitcherCookie(w http.ResponseWriter, r *http.Request, name string, secure bool) {
	http.SetCookie(w, &http.Cookie{ // #nosec G124 -- attributes set below
		Name:     name,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secure,
	})
	_ = r
}
