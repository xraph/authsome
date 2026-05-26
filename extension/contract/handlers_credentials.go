// handlers_credentials.go: Phase C.13 — Credentials dashboard.
//
// The credentials page surfaces the SDK-facing identifiers for the
// platform app: publishable key (safe client-side), app metadata
// (name/slug/ID), and the active environment. There's no rotate-secret
// command yet — App rows don't carry a client secret on the
// authsome.App struct itself; secret-key issuance lives on the apikeys
// surface. When the engine grows a publishable-key rotation method,
// add credentials.rotatePublishableKey here and bind to a confirm
// dialog in the manifest.
package contract

import (
	"context"

	"github.com/xraph/authsome/environment"

	"github.com/xraph/forge/extensions/dashboard/contract"
)

// CredentialsDetail is the credentials.detail response. Field names are
// the React shell's resource.detail bindings — renames are wire breaks.
type CredentialsDetail struct {
	AppID          string `json:"appId"`
	AppName        string `json:"appName"`
	AppSlug        string `json:"appSlug"`
	PublishableKey string `json:"publishableKey,omitempty"`
	EnvID          string `json:"envId,omitempty"`
	EnvName        string `json:"envName,omitempty"`
	EnvSlug        string `json:"envSlug,omitempty"`
	IsPlatform     bool   `json:"isPlatform"`
}

func credentialsDetailHandler(deps Deps) func(ctx context.Context, _ struct{}, _ contract.Principal) (CredentialsDetail, error) {
	return func(ctx context.Context, _ struct{}, _ contract.Principal) (CredentialsDetail, error) {
		eng := deps.Engine
		if eng == nil {
			return CredentialsDetail{}, &contract.Error{Code: contract.CodeUnavailable, Message: "auth engine not configured"}
		}
		appID := defaultAppID(eng)
		a, err := eng.GetApp(ctx, appID)
		if err != nil {
			return CredentialsDetail{}, mapEngineError(err)
		}
		out := CredentialsDetail{
			AppID:          a.ID.String(),
			AppName:        a.Name,
			AppSlug:        a.Slug,
			PublishableKey: a.PublishableKey,
			IsPlatform:     a.IsPlatform,
		}
		// Default environment is informational; failure is non-fatal so
		// freshly seeded deployments without an env still render the page.
		envs, err := eng.ListEnvironments(ctx, appID)
		if err == nil {
			if env := pickDefaultEnv(envs); env != nil {
				out.EnvID = env.ID.String()
				out.EnvName = env.Name
				out.EnvSlug = env.Slug
			}
		}
		return out, nil
	}
}

func pickDefaultEnv(envs []*environment.Environment) *environment.Environment {
	for _, e := range envs {
		if e.IsDefault {
			return e
		}
	}
	if len(envs) > 0 {
		return envs[0]
	}
	return nil
}
