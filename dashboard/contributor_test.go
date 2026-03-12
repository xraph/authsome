package dashboard

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestParseAppEnvRoute verifies URL parsing for the dashboard route dispatcher.
func TestParseAppEnvRoute(t *testing.T) {
	// Build a contributor with known page routes (core + one plugin route).
	c := &Contributor{
		pageRoutes: map[string]bool{
			"/":                     true,
			"/users":                true,
			"/users/detail":         true,
			"/sessions":             true,
			"/sessions/detail":      true,
			"/devices":              true,
			"/devices/detail":       true,
			"/roles":                true,
			"/roles/detail":         true,
			"/webhooks":             true,
			"/environments":         true,
			"/environments/detail":  true,
			"/signup-forms":         true,
			"/signup-forms/edit":    true,
			"/credentials":          true,
			"/plugins":              true,
			"/settings":             true,
			"/settings/editor":      true,
			"/social-providers":     true,
			"/organizations":        true,
			"/scim":                 true,
			"/notifications":        true,
			"/plans":                true,
			"/subscriptions":        true,
			"/invoices":             true,
		},
	}

	tests := []struct {
		name                          string
		route                         string
		wantApp, wantEnv, wantPageRoute string
	}{
		{
			name: "root",
			route: "/",
			wantApp: "", wantEnv: "", wantPageRoute: "/",
		},
		{
			name: "bare core route",
			route: "/users",
			wantApp: "", wantEnv: "", wantPageRoute: "/users",
		},
		{
			name: "bare core sub-route",
			route: "/users/detail",
			wantApp: "", wantEnv: "", wantPageRoute: "/users/detail",
		},
		{
			name: "app/env with core route",
			route: "/platform/development/users",
			wantApp: "platform", wantEnv: "development", wantPageRoute: "/users",
		},
		{
			name: "app/env with root",
			route: "/platform/development/",
			wantApp: "platform", wantEnv: "development", wantPageRoute: "/",
		},
		{
			name: "bare plugin route",
			route: "/social-providers",
			wantApp: "", wantEnv: "", wantPageRoute: "/social-providers",
		},
		{
			name: "bare plugin sub-route",
			route: "/social-providers/detail",
			wantApp: "", wantEnv: "", wantPageRoute: "/social-providers/detail",
		},
		{
			name: "bare scim route",
			route: "/scim",
			wantApp: "", wantEnv: "", wantPageRoute: "/scim",
		},
		{
			name: "bare plans route",
			route: "/plans",
			wantApp: "", wantEnv: "", wantPageRoute: "/plans",
		},
		{
			name: "app/env only",
			route: "/platform/development",
			wantApp: "platform", wantEnv: "development", wantPageRoute: "/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotApp, gotEnv, gotPage := c.parseAppEnvRoute(tt.route)
			if gotApp != tt.wantApp {
				t.Errorf("appSlug = %q, want %q", gotApp, tt.wantApp)
			}
			if gotEnv != tt.wantEnv {
				t.Errorf("envSlug = %q, want %q", gotEnv, tt.wantEnv)
			}
			if gotPage != tt.wantPageRoute {
				t.Errorf("pageRoute = %q, want %q", gotPage, tt.wantPageRoute)
			}
		})
	}
}

// TestTemplFiles_NoAbsoluteHxGetPaths scans all .templ files for hx-get attributes
// that use absolute paths (starting with "/" but not "./" or "../"). Absolute paths
// bypass the app/env context and break dashboard navigation.
func TestTemplFiles_NoAbsoluteHxGetPaths(t *testing.T) {
	root := findProjectRoot(t)

	// Match hx-get="/ or hx-get={ "/ patterns (absolute paths).
	// These are WRONG because they bypass the app/env URL context.
	absPathRe := regexp.MustCompile(`hx-get=\{?\s*"(/[a-z])`)

	// Files to skip (not dashboard navigation links).
	skipFiles := map[string]bool{
		"context_script.templ": true, // interceptor script, not a link
	}

	var violations []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		if !strings.HasSuffix(path, ".templ") {
			return nil
		}
		if skipFiles[filepath.Base(path)] {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		lines := strings.Split(string(data), "\n")
		for i, line := range lines {
			if absPathRe.MatchString(line) {
				rel, _ := filepath.Rel(root, path)
				violations = append(violations, rel+":"+itoa(i+1)+": "+strings.TrimSpace(line))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk error: %v", err)
	}

	if len(violations) > 0 {
		t.Errorf("found %d hx-get attributes with absolute paths (should use ./ or ../  relative paths):\n%s",
			len(violations), strings.Join(violations, "\n"))
	}
}

// TestTemplFiles_DetailLinksUseIdParam scans all .templ files for hx-get attributes
// targeting */detail? routes and verifies they use the standard "?id=" parameter name.
// All dashboard detail handlers expect QueryParams["id"], so using userId, user_id,
// device_id, or session_id causes 404 errors.
func TestTemplFiles_DetailLinksUseIdParam(t *testing.T) {
	root := findProjectRoot(t)

	// Match detail links with non-standard param names.
	badParamRe := regexp.MustCompile(`detail\?(userId|user_id|device_id|session_id)=`)

	var violations []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		if !strings.HasSuffix(path, ".templ") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		lines := strings.Split(string(data), "\n")
		for i, line := range lines {
			if badParamRe.MatchString(line) {
				rel, _ := filepath.Rel(root, path)
				violations = append(violations, rel+":"+itoa(i+1)+": "+strings.TrimSpace(line))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk error: %v", err)
	}

	if len(violations) > 0 {
		t.Errorf("found %d detail links using non-standard param names (should use ?id=):\n%s",
			len(violations), strings.Join(violations, "\n"))
	}
}

// findProjectRoot walks up from the current directory to find the project root
// (the directory containing go.mod).
func findProjectRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find project root (no go.mod)")
		}
		dir = parent
	}
}

// itoa converts int to string without importing strconv.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	if neg {
		s = "-" + s
	}
	return s
}
