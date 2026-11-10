package subscription

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/organization"
	"github.com/xraph/authsome/settings"
	"github.com/xraph/authsome/store"
)

// resolveTenant names the tenant a request is about. An explicit tenant_id
// wins. Without one, organization mode uses the session's active
// organization and user mode uses the caller. The answer still has to pass
// requireTenant: naming a tenant is not the same as being allowed to read it.
func (p *Plugin) resolveTenant(ctx forge.Context, explicit string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	if p.tenantMode(ctx.Context()) == "user" {
		userID, ok := middleware.UserIDFrom(ctx.Context())
		if !ok {
			return "", forge.Unauthorized("authentication required")
		}
		return userID.String(), nil
	}
	orgID, ok := middleware.OrgIDFrom(ctx.Context())
	if !ok || orgID.IsNil() {
		return "", forge.BadRequest("no active organization: switch to one or pass tenant_id")
	}
	return orgID.String(), nil
}

// canManageTenant reports whether userID may change tenantID's subscription
// through the self-service routes. An operator holding manage/billing always
// may. In user mode only the user is their own tenant. In organization mode
// the caller must be a member whose role is listed in
// subscription.billing_roles as resolved for that organization.
func (p *Plugin) canManageTenant(ctx context.Context, userID id.UserID, tenantID string) (bool, error) {
	if p.holdsBillingManage(ctx, userID) {
		return true, nil
	}
	if p.tenantMode(ctx) == "user" {
		return userID.String() == tenantID, nil
	}

	orgID, err := id.ParseOrgID(tenantID)
	if err != nil || p.authStore == nil {
		return false, nil //nolint:nilerr // a tenant that isn't an org ID has no members
	}
	member, err := p.authStore.GetMemberByUserAndOrg(ctx, userID, orgID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("look up membership: %w", err)
	}
	return slices.Contains(p.billingRoles(ctx, orgID), normalizeRole(string(member.Role))), nil
}

// defaultBillingRoles mirrors SettingBillingRoles' default for a plugin
// running without a settings manager.
var defaultBillingRoles = []string{string(organization.RoleOwner)}

// billingRoles resolves subscription.billing_roles for orgID, normalized.
// settings.Get falls back to the default on a store error.
func (p *Plugin) billingRoles(ctx context.Context, orgID id.OrgID) []string {
	roles := defaultBillingRoles
	if p.settings != nil {
		roles, _ = settings.Get(ctx, p.settings, SettingBillingRoles, settings.ResolveOpts{ //nolint:errcheck // Get returns the default on error
			AppID: p.contextAppID(ctx),
			OrgID: orgID.String(),
		})
	}
	out := make([]string, 0, len(roles))
	for _, r := range roles {
		out = append(out, normalizeRole(r))
	}
	return out
}

// requireSelfService gates the self-service writes. The caller must be able
// to read the tenant (a 404 otherwise, like every tenant read), self-service
// must be on for the app, and the caller must hold the billing permission for
// that tenant.
func (p *Plugin) requireSelfService(ctx forge.Context, tenantID string) error {
	if err := p.requireTenant(ctx, tenantID); err != nil {
		return err
	}
	userID, ok := middleware.UserIDFrom(ctx.Context())
	if !ok {
		return forge.Unauthorized("authentication required")
	}

	if !p.holdsBillingManage(ctx.Context(), userID) && !p.selfServiceOn(ctx.Context()) {
		return forge.Forbidden("self-service billing is turned off")
	}

	allowed, err := p.canManageTenant(ctx.Context(), userID, tenantID)
	if err != nil {
		return middleware.InternalError(ctx, err)
	}
	if !allowed {
		return forge.Forbidden("your role can't manage billing for this organization")
	}
	return nil
}

// selfServiceOn resolves subscription.self_service_upgrade for the request's
// app. settings.Get falls back to the default (on) on a store error.
func (p *Plugin) selfServiceOn(ctx context.Context) bool {
	if p.settings == nil {
		return true
	}
	on, _ := settings.Get(ctx, p.settings, SettingSelfServiceUpgrade, settings.ResolveOpts{AppID: p.contextAppID(ctx)}) //nolint:errcheck // Get returns the default on error
	return on
}

// contextAppID is the request's app scope, or the plugin's default app.
func (p *Plugin) contextAppID(ctx context.Context) string {
	if a, ok := middleware.AppIDFrom(ctx); ok && !a.IsNil() {
		return a.String()
	}
	return p.defaultAppID
}

// normalizeRole lowercases and trims a role so "Owner" and " owner " match.
func normalizeRole(r string) string {
	return strings.ToLower(strings.TrimSpace(r))
}
