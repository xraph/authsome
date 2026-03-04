package dashboard

import (
	"context"
	"fmt"
	"time"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/environment"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/rbac"
	"github.com/xraph/authsome/user"
	"github.com/xraph/authsome/webhook"
)

// fetchStats returns the total number of users for the given app.
func fetchStats(ctx context.Context, engine *authsome.Engine, appID id.AppID) (totalUsers int, err error) {
	users, err := engine.AdminListUsers(ctx, &user.UserQuery{
		AppID: appID,
		Limit: 1,
	})
	if err != nil {
		return 0, fmt.Errorf("dashboard: fetch user stats: %w", err)
	}

	return users.Total, nil
}

// fetchUsers returns a paginated list of users for the given app.
func fetchUsers(ctx context.Context, engine *authsome.Engine, appID id.AppID, cursor string, limit int) (*user.UserList, error) {
	if limit <= 0 {
		limit = 25
	}

	list, err := engine.AdminListUsers(ctx, &user.UserQuery{
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

// formatTimeAgo returns a human-readable relative time string such as
// "2m ago", "3h ago", "5d ago", or "1y ago".
func formatTimeAgo(t time.Time) string {
	d := time.Since(t)

	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	case d < 365*24*time.Hour:
		return fmt.Sprintf("%dmo ago", int(d.Hours()/(24*30)))
	default:
		return fmt.Sprintf("%dy ago", int(d.Hours()/(24*365)))
	}
}

// fetchEnvironments returns all environments for the given app.
func fetchEnvironments(ctx context.Context, engine *authsome.Engine, appID id.AppID) ([]*environment.Environment, error) {
	envs, err := engine.ListEnvironments(ctx, appID)
	if err != nil {
		return nil, fmt.Errorf("dashboard: fetch environments: %w", err)
	}

	return envs, nil
}

// truncateString shortens s to max characters and appends "..." if truncated.
func truncateString(s string, max int) string {
	if len(s) <= max {
		return s
	}

	if max <= 3 {
		return s[:max]
	}

	return s[:max-3] + "..."
}
