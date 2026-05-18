package components

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/xraph/authsome/app"
	"github.com/xraph/authsome/environment"
	"github.com/xraph/authsome/id"
)

func renderToString(t *testing.T, render func(ctx context.Context, w *bytes.Buffer) error) string {
	t.Helper()

	var buf bytes.Buffer
	if err := render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}

	return buf.String()
}

func TestContextScript_UsesPageBaseAttribute(t *testing.T) {
	cases := []struct {
		name        string
		pagesPrefix string
		wantAttr    string
		wantInJS    string
	}{
		{
			name:        "remote prefix",
			pagesPrefix: "/dashboard/remote/authsome/pages/",
			wantAttr:    `data-pages-prefix="/dashboard/remote/authsome/pages/"`,
			wantInJS:    "data-pages-prefix",
		},
		{
			name:        "local prefix",
			pagesPrefix: "/dashboard/ext/authsome/pages/",
			wantAttr:    `data-pages-prefix="/dashboard/ext/authsome/pages/"`,
			wantInJS:    "data-pages-prefix",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			html := renderToString(t, func(ctx context.Context, w *bytes.Buffer) error {
				return ContextScript("platform", "development", "/users,/sessions", tc.pagesPrefix).Render(ctx, w)
			})

			if !strings.Contains(html, tc.wantAttr) {
				t.Errorf("HTML missing attribute %q.\nHTML: %s", tc.wantAttr, html)
			}

			if !strings.Contains(html, tc.wantInJS) {
				t.Errorf("HTML missing JS reference to data-pages-prefix.\nHTML: %s", html)
			}
		})
	}
}

func TestAppSwitcher_HrefBuiltFromPageBase(t *testing.T) {
	currentApp := &app.App{ID: id.AppID{}, Name: "Platform", Slug: "platform"}
	otherApp := &app.App{ID: id.AppID{}, Name: "GameFramework", Slug: "game-framework"}

	cases := []struct {
		name     string
		data     AppSwitcherData
		wantHref string
	}{
		{
			name: "remote PageBase",
			data: AppSwitcherData{
				Current:        currentApp,
				All:            []*app.App{currentApp, otherApp},
				CurrentEnvSlug: "development",
				CurrentPage:    "/users",
				BasePath:       "/dashboard",
				PageBase:       "/dashboard/remote/authsome/pages",
			},
			wantHref: "/dashboard/remote/authsome/pages/game-framework/development/users",
		},
		{
			name: "local PageBase",
			data: AppSwitcherData{
				Current:        currentApp,
				All:            []*app.App{currentApp, otherApp},
				CurrentEnvSlug: "development",
				CurrentPage:    "/users",
				BasePath:       "/dashboard",
				PageBase:       "/dashboard/ext/authsome/pages",
			},
			wantHref: "/dashboard/ext/authsome/pages/game-framework/development/users",
		},
		{
			name: "empty PageBase falls back to BasePath + /ext/authsome/pages",
			data: AppSwitcherData{
				Current:        currentApp,
				All:            []*app.App{currentApp, otherApp},
				CurrentEnvSlug: "development",
				CurrentPage:    "/users",
				BasePath:       "/dashboard",
				PageBase:       "",
			},
			wantHref: "/dashboard/ext/authsome/pages/game-framework/development/users",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			html := renderToString(t, func(ctx context.Context, w *bytes.Buffer) error {
				return AppSwitcher(tc.data).Render(ctx, w)
			})

			if !strings.Contains(html, tc.wantHref) {
				t.Errorf("AppSwitcher HTML missing %q\nHTML excerpt: %s", tc.wantHref, hxGetSnippet(html))
			}
		})
	}
}

func TestAppSwitcherPagesPrefix(t *testing.T) {
	cases := []struct {
		name string
		data AppSwitcherData
		want string
	}{
		{"explicit page base wins", AppSwitcherData{BasePath: "/dashboard", PageBase: "/dashboard/remote/authsome/pages"}, "/dashboard/remote/authsome/pages"},
		{"empty page base falls back", AppSwitcherData{BasePath: "/dashboard", PageBase: ""}, "/dashboard/ext/authsome/pages"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.data.AppSwitcherPagesPrefix(); got != tc.want {
				t.Errorf("AppSwitcherPagesPrefix = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestTopbarEnvSwitcher_HrefBuiltFromPageBase(t *testing.T) {
	current := &environment.Environment{ID: id.EnvironmentID{}, Name: "Development", Slug: "development", Color: "#3b82f6", IsDefault: true}
	other := &environment.Environment{ID: id.EnvironmentID{}, Name: "Production", Slug: "production", Color: "#10b981"}

	cases := []struct {
		name        string
		basePath    string
		pagesPrefix string
		wantHref    string
	}{
		{
			name:        "remote pages prefix",
			basePath:    "/dashboard",
			pagesPrefix: "/dashboard/remote/authsome/pages",
			wantHref:    "/dashboard/remote/authsome/pages/platform/production/users",
		},
		{
			name:        "local pages prefix",
			basePath:    "/dashboard",
			pagesPrefix: "/dashboard/ext/authsome/pages",
			wantHref:    "/dashboard/ext/authsome/pages/platform/production/users",
		},
		{
			name:        "empty pages prefix falls back to basePath + /ext/authsome/pages",
			basePath:    "/dashboard",
			pagesPrefix: "",
			wantHref:    "/dashboard/ext/authsome/pages/platform/production/users",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			html := renderToString(t, func(ctx context.Context, w *bytes.Buffer) error {
				return TopbarEnvSwitcher(
					current,
					[]*environment.Environment{current, other},
					"platform",
					"/users",
					tc.basePath,
					tc.pagesPrefix,
				).Render(ctx, w)
			})

			if !strings.Contains(html, tc.wantHref) {
				t.Errorf("TopbarEnvSwitcher missing %q\nexcerpt: %s", tc.wantHref, hxGetSnippet(html))
			}
		})
	}
}

// hxGetSnippet pulls the first hx-get attribute out of an HTML blob so test
// failure messages point at the relevant URL rather than a giant render.
func hxGetSnippet(html string) string {
	idx := strings.Index(html, `hx-get="`)
	if idx < 0 {
		return "<no hx-get found>"
	}

	end := strings.Index(html[idx:], `"`)
	if end < 0 {
		return html[idx:]
	}

	rest := html[idx+end+1:]

	closeIdx := strings.Index(rest, `"`)
	if closeIdx < 0 {
		return html[idx:]
	}

	return html[idx : idx+end+1+closeIdx+1]
}
