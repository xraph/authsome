package wardenseed

import (
	"context"
	"fmt"

	"github.com/xraph/warden"
	"github.com/xraph/warden/dsl"

	"github.com/xraph/authsome/id"
)

// ApplyOptions configures ApplyForApp.
type ApplyOptions struct {
	// SharedOverride, when set, replaces the embedded shared source.
	// Used by extension config (cfg.WardenDir) to let operators ship
	// custom .warden files without rebuilding the binary. Applies to
	// every app (platform + non-platform).
	SharedOverride *Source

	// PlatformOverride, when set, replaces the embedded platform
	// source. Only consulted when ApplyForApp is called with
	// isPlatform=true.
	PlatformOverride *Source

	// DryRun forwards to dsl.ApplyOptions.DryRun. When true, no writes
	// hit the warden store; the result still reports what *would* have
	// changed.
	DryRun bool
}

// ApplyForApp materialises authsome's default warden DSL programs against
// the given engine for the given app. When isPlatform is true, the
// platform-only roles are merged into the same Program before applying so
// cross-file grant references (platform-* roles granting perms from the
// shared catalog) resolve in a single pass — important for DryRun, which
// can only validate against declarations + already-stored entities.
//
// The function is idempotent: calling it twice on a clean engine yields
// 0 created / 0 updated entries on the second pass.
func ApplyForApp(ctx context.Context, eng *warden.Engine, appID id.AppID, isPlatform bool, opts ApplyOptions) (*dsl.ApplyResult, error) {
	if eng == nil {
		return nil, fmt.Errorf("wardenseed: warden engine is nil")
	}
	if appID.IsNil() {
		return nil, fmt.Errorf("wardenseed: app id is nil")
	}

	sharedSrc := SharedSource()
	if opts.SharedOverride != nil {
		sharedSrc = *opts.SharedOverride
	}
	prog, err := Load(sharedSrc, LoadOptions{AppID: appID.String()})
	if err != nil {
		return nil, fmt.Errorf("wardenseed: load shared: %w", err)
	}

	if isPlatform {
		platformSrc := PlatformSource()
		if opts.PlatformOverride != nil {
			platformSrc = *opts.PlatformOverride
		}
		platformProg, perr := Load(platformSrc, LoadOptions{AppID: appID.String()})
		if perr != nil {
			return nil, fmt.Errorf("wardenseed: load platform: %w", perr)
		}
		mergePrograms(prog, platformProg)
	}

	res, err := dsl.Apply(ctx, eng, prog, dsl.ApplyOptions{
		TenantID: appID.String(),
		AppID:    appID.String(),
		DryRun:   opts.DryRun,
	})
	if err != nil {
		return nil, fmt.Errorf("wardenseed: apply: %w", err)
	}
	return res, nil
}

// mergePrograms folds src into dst. The two programs use different
// namespaces (shared = root + "app"; platform = "platform"), so name
// collisions across the merge are not expected. Header metadata
// (tenant, app, version) is identical because both sources are loaded
// with the same ${APP_ID} substitution.
func mergePrograms(dst, src *dsl.Program) {
	dst.ResourceTypes = append(dst.ResourceTypes, src.ResourceTypes...)
	dst.Permissions = append(dst.Permissions, src.Permissions...)
	dst.Roles = append(dst.Roles, src.Roles...)
	dst.Policies = append(dst.Policies, src.Policies...)
	dst.Relations = append(dst.Relations, src.Relations...)
	dst.Namespaces = append(dst.Namespaces, src.Namespaces...)
}
