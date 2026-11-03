package subscription

import (
	"context"
	"errors"
	"fmt"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/plugin"
	"github.com/xraph/authsome/settings"
	"github.com/xraph/authsome/store"
)

// requireTenant lets the caller through only when it may read tenantID's
// billing. In organization mode that means being a member of the org whose ID
// is tenantID; in user mode it means being that user. Either way, holding
// manage/billing reads any tenant, which is how operators see across them.
//
// Every refusal is a 404, so a caller outside the tenant can't tell a tenant
// that exists from one that doesn't. An empty tenantID belongs to no one, so
// only a billing manager gets past it.
func (p *Plugin) requireTenant(ctx forge.Context, tenantID string) error {
	ok, err := p.tenantAllowed(ctx, tenantID)
	if err != nil {
		return err
	}
	if !ok {
		return forge.NotFound("not found")
	}
	return nil
}

// tenantAllowed answers requireTenant without choosing the refusal, for
// handlers that must refuse with the same 404 a missing record gets.
func (p *Plugin) tenantAllowed(ctx forge.Context, tenantID string) (bool, error) {
	userID, ok := middleware.UserIDFrom(ctx.Context())
	if !ok {
		return false, forge.Unauthorized("authentication required")
	}

	if tenantID != "" {
		ok, err := p.ownsTenant(ctx.Context(), userID, tenantID)
		if err != nil {
			return false, middleware.InternalError(ctx, fmt.Errorf("check tenant access: %w", err))
		}
		if ok {
			return true, nil
		}
	}

	return p.holdsBillingManage(ctx.Context(), userID), nil
}

// ownsTenant reports whether userID is the tenant (user mode) or a member of
// it (organization mode).
func (p *Plugin) ownsTenant(ctx context.Context, userID id.UserID, tenantID string) (bool, error) {
	if p.tenantMode(ctx) == "user" {
		return userID.String() == tenantID, nil
	}

	orgID, err := id.ParseOrgID(tenantID)
	if err != nil || p.authStore == nil {
		return false, nil //nolint:nilerr // a tenant that isn't an org ID has no members
	}
	if _, err := p.authStore.GetMemberByUserAndOrg(ctx, userID, orgID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// tenantMode resolves subscription.tenant_mode for the request's app, falling
// back to the registered default when the plugin has no settings manager.
func (p *Plugin) tenantMode(ctx context.Context) string {
	if p.settings == nil {
		return "organization"
	}
	appID := p.defaultAppID
	if a, ok := middleware.AppIDFrom(ctx); ok {
		appID = a.String()
	}
	mode, _ := settings.Get(ctx, p.settings, SettingTenantMode, settings.ResolveOpts{AppID: appID}) //nolint:errcheck // falls back to the registered default
	return mode
}

// holdsBillingManage reports whether the caller holds manage/billing, the
// same permission writeGuard demands. It fails closed: an engine without RBAC,
// a delegated caller the engine can't evaluate, or a check that errors all
// count as not holding it.
//
// A delegated session is checked with its actor chain, as RequirePermission
// does, so an agent doesn't read across tenants on its user's authority.
func (p *Plugin) holdsBillingManage(ctx context.Context, userID id.UserID) bool {
	checker, ok := p.engine.(plugin.PermissionChecker)
	if !ok {
		return false
	}

	var (
		allowed bool
		err     error
	)
	if actors := middleware.AuthzActorsFrom(ctx); len(actors) > 0 {
		chain, ok := p.engine.(middleware.ChainPermissionChecker)
		if !ok {
			return false
		}
		subject, ok := middleware.CallerSubjectFrom(ctx)
		if !ok {
			return false
		}
		allowed, err = chain.Can(ctx, subject, actors, "manage", "billing")
	} else {
		allowed, err = checker.HasPermission(ctx, userID, "manage", "billing")
	}
	if err != nil {
		if p.logger != nil {
			p.logger.Warn("subscription: billing permission check failed",
				log.String("user_id", userID.String()), log.Error(err))
		}
		return false
	}
	return allowed
}

// entitlementContext carries the tenant and app the way ledger.Entitled reads
// them: as plain string values under the raw "tenant_id" and "app_id" keys.
// Without them it answers "missing tenant or app context" for everyone.
func entitlementContext(ctx context.Context, tenantID, appID string) context.Context {
	ctx = context.WithValue(ctx, "tenant_id", tenantID) //nolint:staticcheck,revive // ledger.Entitled reads this raw string key
	return context.WithValue(ctx, "app_id", appID)      //nolint:staticcheck,revive // ledger.Entitled reads this raw string key
}
