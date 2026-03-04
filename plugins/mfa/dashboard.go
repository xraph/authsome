package mfa

import (
	"context"

	"github.com/a-h/templ"

	"github.com/xraph/authsome/dashboard"
	"github.com/xraph/authsome/id"
	mfadash "github.com/xraph/authsome/plugins/mfa/dashui"
)

// Compile-time interface checks.
var (
	_ dashboard.DashboardPlugin       = (*Plugin)(nil)
	_ dashboard.UserDetailContributor = (*Plugin)(nil)
)

// DashboardWidgets returns MFA-related widgets.
func (p *Plugin) DashboardWidgets(_ context.Context) []dashboard.PluginWidget {
	return []dashboard.PluginWidget{
		{
			ID:         "mfa-coverage",
			Title:      "MFA Coverage",
			Size:       "sm",
			RefreshSec: 60,
			Render: func(ctx context.Context) templ.Component {
				return mfadash.CoverageWidget()
			},
		},
	}
}

// DashboardSettingsPanel returns the MFA settings panel.
func (p *Plugin) DashboardSettingsPanel(_ context.Context) templ.Component {
	return mfadash.SettingsPanel(p.config.Issuer)
}

// DashboardPages returns extra page routes for MFA.
func (p *Plugin) DashboardPages() []dashboard.PluginPage {
	return []dashboard.PluginPage{
		{
			Route: "/mfa",
			Label: "MFA",
			Icon:  "shield-check",
			Render: func(ctx context.Context) templ.Component {
				return mfadash.EnrollmentsPage()
			},
		},
	}
}

// DashboardUserDetailSection returns the user-specific MFA section.
func (p *Plugin) DashboardUserDetailSection(ctx context.Context, userID id.UserID) templ.Component {
	if p.store == nil {
		return nil
	}
	enrollments, err := p.store.ListEnrollments(ctx, userID)
	if err != nil {
		enrollments = nil
	}

	// Convert to view models.
	views := make([]mfadash.EnrollmentView, len(enrollments))
	for i, e := range enrollments {
		views[i] = mfadash.EnrollmentView{
			Method:    e.Method,
			Verified:  e.Verified,
			CreatedAt: e.CreatedAt,
		}
	}
	return mfadash.UserSection(views)
}
