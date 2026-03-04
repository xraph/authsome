package dashboard

import (
	"context"
	"io"

	"github.com/a-h/templ"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/dashboard/components"
	"github.com/xraph/authsome/environment"
)

// appSwitcherFromContext returns a templ.Component that renders the app switcher
// using app/env data from request context. This is set on the manifest at startup
// and renders dynamically at request time.
func appSwitcherFromContext(engine *authsome.Engine) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		appSlug := AppSlugFromContext(ctx)
		envSlug := EnvSlugFromContext(ctx)
		if appSlug == "" || envSlug == "" {
			return nil
		}

		appID, ok := AppIDFromContext(ctx)
		if !ok {
			return nil
		}

		currentApp, err := engine.GetApp(ctx, appID)
		if err != nil {
			return nil
		}

		allApps, err := engine.ListApps(ctx)
		if err != nil || len(allApps) == 0 {
			return nil
		}

		// Determine current page route (default to "/").
		currentPage := "/"

		return components.AppSwitcher(currentApp, allApps, envSlug, currentPage, "").Render(ctx, w)
	})
}

// envSwitcherFromContext returns a templ.Component that renders the topbar
// environment switcher using data from request context.
func envSwitcherFromContext(engine *authsome.Engine) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		appSlug := AppSlugFromContext(ctx)
		envSlug := EnvSlugFromContext(ctx)
		if appSlug == "" || envSlug == "" {
			return nil
		}

		appID, ok := AppIDFromContext(ctx)
		if !ok {
			return nil
		}

		envID, ok := EnvIDFromContext(ctx)
		if !ok {
			return nil
		}

		currentEnv, err := engine.GetEnvironment(ctx, envID)
		if err != nil {
			return nil
		}

		allEnvs, err := engine.ListEnvironments(ctx, appID)
		if err != nil {
			allEnvs = []*environment.Environment{currentEnv}
		}

		currentPage := "/"

		return components.TopbarEnvSwitcher(currentEnv, allEnvs, appSlug, currentPage, "").Render(ctx, w)
	})
}

