// handlers_overview.go: Phase C.12 — Overview dashboard.
//
// overview.stats returns the four counters the dashboard.grid widgets
// surface (users, active sessions, devices, plugins) in a single
// payload so the React shell makes one request rather than four.
// overview.recentSignups returns the top-N most recently created users
// for the dashboard.recentlist widget.
package contract

import (
	"context"
	"time"

	"github.com/xraph/authsome/user"

	"github.com/xraph/forge/extensions/dashboard/contract"
)

// ────────────────────────────────────────────────────────────────────
// Wire shapes
// ────────────────────────────────────────────────────────────────────

// OverviewStats is the overview.stats response shape. Field names match
// the React shell's dashboard.stat valueField bindings exactly; renames
// are wire breaks.
type OverviewStats struct {
	Users    int `json:"users"`
	Sessions int `json:"sessions"`
	Devices  int `json:"devices"`
	Plugins  int `json:"plugins"`
}

// RecentSignupsInput is the optional input for overview.recentSignups.
// Limit caps the returned slice; defaults to 10 when unset / non-positive.
type RecentSignupsInput struct {
	Limit int `json:"limit,omitempty"`
}

// RecentSignupsResponse is the overview.recentSignups reply. The
// dashboard.recentlist widget's extractor picks the first array-valued
// field, which is `users` — the same UserSummary shape the /users page
// uses, kept consistent so the recent-signups row click can deep-link
// to the user-detail page without a separate fetch.
type RecentSignupsResponse struct {
	Users []UserSummary `json:"users"`
}

// ────────────────────────────────────────────────────────────────────
// Handlers
// ────────────────────────────────────────────────────────────────────

func overviewStatsHandler(deps Deps) func(ctx context.Context, _ struct{}, _ contract.Principal) (OverviewStats, error) {
	return func(ctx context.Context, _ struct{}, _ contract.Principal) (OverviewStats, error) {
		eng := deps.Engine
		if eng == nil {
			return OverviewStats{}, &contract.Error{Code: contract.CodeUnavailable, Message: "auth engine not configured"}
		}

		out := OverviewStats{}

		// Users — Limit:1 keeps the payload tiny; Total is computed
		// server-side regardless of Limit (the engine fills it from the
		// underlying count query).
		userList, err := eng.AdminListUsers(ctx, &user.Query{
			AppID: defaultAppID(eng),
			Limit: 1,
		})
		if err != nil {
			return OverviewStats{}, mapEngineError(err)
		}
		out.Users = userList.Total

		// Active sessions — bound to a large cap so a snapshot stays
		// honest on busy deployments. The engine doesn't expose a
		// dedicated count today; len() of the listing is the cheapest
		// approximation that doesn't require a new admin endpoint.
		sessions, err := eng.ListAllSessions(ctx, 10000)
		if err != nil {
			return OverviewStats{}, mapEngineError(err)
		}
		out.Sessions = len(sessions)

		// Devices — same limit/approximation as sessions.
		devices, err := eng.ListAllDevices(ctx, 10000)
		if err != nil {
			return OverviewStats{}, mapEngineError(err)
		}
		out.Devices = len(devices)

		// Plugins — registry-level count, no I/O.
		if reg := eng.Plugins(); reg != nil {
			out.Plugins = len(reg.Plugins())
		}

		return out, nil
	}
}

func overviewRecentSignupsHandler(deps Deps) func(ctx context.Context, in RecentSignupsInput, _ contract.Principal) (RecentSignupsResponse, error) {
	return func(ctx context.Context, in RecentSignupsInput, _ contract.Principal) (RecentSignupsResponse, error) {
		eng := deps.Engine
		if eng == nil {
			return RecentSignupsResponse{}, &contract.Error{Code: contract.CodeUnavailable, Message: "auth engine not configured"}
		}
		limit := in.Limit
		if limit <= 0 {
			limit = 10
		}
		list, err := eng.AdminListUsers(ctx, &user.Query{
			AppID: defaultAppID(eng),
			Limit: limit,
		})
		if err != nil {
			return RecentSignupsResponse{}, mapEngineError(err)
		}
		out := RecentSignupsResponse{Users: make([]UserSummary, 0, len(list.Users))}
		for _, u := range list.Users {
			out.Users = append(out.Users, projectUserSummary(u))
		}
		return out, nil
	}
}

// Reserve a reference to time so future fields (last24h count, etc.)
// don't require an import-order shuffle. The compiler optimizes this
// out — kept here intentionally.
var _ = time.Now
