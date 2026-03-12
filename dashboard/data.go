package dashboard

import (
	"context"
	"fmt"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/environment"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/rbac"
	"github.com/xraph/authsome/user"
	"github.com/xraph/authsome/webhook"
)

// fetchStats returns the total number of users for the given app.
func fetchStats(ctx context.Context, engine *authsome.Engine, appID id.AppID) (totalUsers int, err error) {
	users, err := engine.AdminListUsers(ctx, &user.Query{
		AppID: appID,
		Limit: 1,
	})
	if err != nil {
		return 0, fmt.Errorf("dashboard: fetch user stats: %w", err)
	}

	return users.Total, nil
}

// fetchUsers returns a paginated list of users for the given app.
func fetchUsers(ctx context.Context, engine *authsome.Engine, appID id.AppID, cursor string, limit int) (*user.List, error) {
	if limit <= 0 {
		limit = 25
	}

	list, err := engine.AdminListUsers(ctx, &user.Query{
		AppID:  appID,
		Cursor: cursor,
		Limit:  limit,
	})
	if err != nil {
		return nil, fmt.Errorf("dashboard: fetch users: %w", err)
	}

	return list, nil
}

// fetchRoles returns all roles defined for the given app.
func fetchRoles(ctx context.Context, engine *authsome.Engine, appID id.AppID) ([]*rbac.Role, error) {
	roles, err := engine.ListRoles(ctx, appID)
	if err != nil {
		return nil, fmt.Errorf("dashboard: fetch roles: %w", err)
	}

	return roles, nil
}

// fetchWebhooks returns all registered webhooks for the given app.
func fetchWebhooks(ctx context.Context, engine *authsome.Engine, appID id.AppID) ([]*webhook.Webhook, error) {
	hooks, err := engine.ListWebhooks(ctx, appID)
	if err != nil {
		return nil, fmt.Errorf("dashboard: fetch webhooks: %w", err)
	}

	return hooks, nil
}

// fetchEnvironments returns all environments for the given app.
func fetchEnvironments(ctx context.Context, engine *authsome.Engine, appID id.AppID) ([]*environment.Environment, error) {
	envs, err := engine.ListEnvironments(ctx, appID)
	if err != nil {
		return nil, fmt.Errorf("dashboard: fetch environments: %w", err)
	}

	return envs, nil
}
