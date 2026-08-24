// App-scoped variants of generated list calls.
//
// Each one wraps an endpoint whose app_id, email, cursor or limit is a query
// parameter, which the generator emits as a params struct rather than as plain
// arguments. These signatures are the ones callers actually want, so they are
// maintained here by hand.
//
// They live outside client.go on purpose. That file is regenerated from the
// OpenAPI spec on every `make -C sdkgen generate`, and anything added to it is
// silently deleted the next time the generator runs. Anything hand-written
// belongs in a companion file like this one, or in the template if it needs a
// field on Client.

package authclient

import (
	"context"
	"fmt"
	"strings"
)

// AdminListUsersInApp lists users scoped to a specific App. Supplies
// the app_id, email, cursor, and limit query params understood by the
// /v1/admin/users handler. Empty fields are omitted from the URL.
func (c *Client) AdminListUsersInApp(ctx context.Context, appID, email, cursor string, limit int) (*AdminUserListResponse, error) {
	path := "/v1/admin/users"
	q := []string{}
	if appID != "" {
		q = append(q, "app_id="+appID)
	}
	if email != "" {
		q = append(q, "email="+email)
	}
	if cursor != "" {
		q = append(q, "cursor="+cursor)
	}
	if limit > 0 {
		q = append(q, fmt.Sprintf("limit=%d", limit))
	}
	if len(q) > 0 {
		path += "?" + strings.Join(q, "&")
	}
	var result AdminUserListResponse
	if err := c.do(ctx, "GET", path, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// AuthsomeListRolesInApp — List roles scoped to an explicit app_id.
// The bare AuthsomeListRoles uses whatever app the engine derives from
// the call's auth context (which falls back to the configured default
// app); this variant lets cross-app admin clients (e.g. TwinOS studio,
// which runs one query per workspace) target a specific app without
// switching credentials.
func (c *Client) AuthsomeListRolesInApp(ctx context.Context, appID string) (*RoleListResponse, error) {
	path := "/v1/roles"
	if appID != "" {
		path = path + "?app_id=" + appID
	}
	var result RoleListResponse
	if err := c.do(ctx, "GET", path, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// AuthsomeListUserRolesInApp — List user roles scoped to an explicit
// app_id. The bare AuthsomeListUserRoles uses the engine's platform
// app, which is wrong for cross-app admin tooling: TwinOS studio's
// API key authenticates against the platform app but role assignments
// live in per-workspace apps. Pass the workspace's AppID here so the
// engine's ListUserRolesInApp queries the right tenant.
func (c *Client) AuthsomeListUserRolesInApp(ctx context.Context, userId, appID string) (*UserRoleListResponse, error) {
	path := "/v1/users/{userId}/roles"
	path = strings.Replace(path, "{userId}", userId, 1)
	if appID != "" {
		path = path + "?app_id=" + appID
	}
	var result UserRoleListResponse
	if err := c.do(ctx, "GET", path, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
