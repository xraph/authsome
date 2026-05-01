package social

import (
	"strings"
	"testing"
)

func TestSanitizeRedirectURL_NoOriginRejectsAbsolute(t *testing.T) {
	t.Parallel()
	got := sanitizeRedirectURL("https://attacker.example/path", "")
	if got != "" {
		t.Fatalf("expected empty (rejected); got %q", got)
	}
}

func TestSanitizeRedirectURL_NoOriginAllowsRelative(t *testing.T) {
	t.Parallel()
	cases := []string{"/", "/dashboard", "/path?x=1", "relative/path"}
	for _, in := range cases {
		got := sanitizeRedirectURL(in, "")
		if got != in {
			t.Errorf("relative %q: got %q want %q", in, got, in)
		}
	}
}

func TestSanitizeRedirectURL_OriginMatch(t *testing.T) {
	t.Parallel()
	got := sanitizeRedirectURL("https://app.example/dash", "https://app.example")
	if got != "https://app.example/dash" {
		t.Fatalf("origin match: got %q", got)
	}
}

func TestSanitizeRedirectURL_OriginMismatch(t *testing.T) {
	t.Parallel()
	got := sanitizeRedirectURL("https://attacker.example/", "https://app.example")
	if got != "" {
		t.Fatalf("origin mismatch: expected empty, got %q", got)
	}
}

func TestSanitizeRedirectURL_SchemeInjection(t *testing.T) {
	t.Parallel()
	bad := []string{
		"javascript:alert(1)",
		"data:text/html,<script>alert(1)</script>",
		"file:///etc/passwd",
	}
	for _, u := range bad {
		if got := sanitizeRedirectURL(u, "https://app.example"); got != "" {
			t.Errorf("%q: expected empty, got %q", u, got)
		}
		if got := sanitizeRedirectURL(u, ""); got != "" {
			t.Errorf("%q (no origin): expected empty, got %q", u, got)
		}
	}
}

func TestSanitizeRedirectURL_CredentialsRejected(t *testing.T) {
	t.Parallel()
	got := sanitizeRedirectURL("https://user:pass@app.example/x", "https://app.example")
	if got != "" {
		t.Fatalf("creds: expected empty, got %q", got)
	}
}

func TestSanitizeRedirectURL_CaseInsensitiveHost(t *testing.T) {
	t.Parallel()
	got := sanitizeRedirectURL("https://APP.example/x", "https://app.example")
	if got == "" {
		t.Fatal("case-insensitive host: expected non-empty")
	}
}

func TestSanitizeFrontendURL_RequiresAbsoluteHTTP(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"https passes through", "https://app.example", "https://app.example"},
		{"http passes through", "http://app.example", "http://app.example"},
		{"empty stays empty", "", ""},
		{"relative path rejected", "/relative/path", ""},
		{"javascript rejected", "javascript:alert(1)", ""},
		{"credentials rejected", "https://user:p@a.b/", ""},
		{"ftp rejected", "ftp://ftp.app.example/", ""},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := sanitizeFrontendURL(tc.in)
			if got != tc.want {
				t.Fatalf("in=%q: got %q want %q", tc.in, got, tc.want)
			}
		})
	}
}

func FuzzSanitizeRedirectURL(f *testing.F) {
	seeds := []struct {
		raw, origin string
	}{
		{"/dash", ""},
		{"https://app.example/x", "https://app.example"},
		{"javascript:alert(1)", "https://app.example"},
		{"https://attacker.example", ""},
		{"https://USER:p@app.example/x", "https://app.example"},
	}
	for _, s := range seeds {
		f.Add(s.raw, s.origin)
	}
	f.Fuzz(func(t *testing.T, raw, origin string) {
		got := sanitizeRedirectURL(raw, origin)
		if got == "" {
			return
		}
		// Output must be prefix-safe: relative path or http(s).
		if !(strings.HasPrefix(got, "/") ||
			strings.HasPrefix(strings.ToLower(got), "http://") ||
			strings.HasPrefix(strings.ToLower(got), "https://") ||
			// Relative paths without leading / are passed through (e.g. "foo/bar").
			// They lack a scheme so they cannot be dangerous as long as no scheme prefix appears.
			!containsScheme(got)) {
			t.Fatalf("unsafe output prefix: raw=%q origin=%q got=%q", raw, origin, got)
		}
	})
}

// containsScheme returns true if s looks like it has an absolute URL scheme.
func containsScheme(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == ':' {
			return i > 0
		}
		if c == '/' || c == '?' || c == '#' {
			return false
		}
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
			(i > 0 && ((c >= '0' && c <= '9') || c == '+' || c == '-' || c == '.'))) {
			return false
		}
	}
	return false
}

func FuzzSanitizeFrontendURL(f *testing.F) {
	seeds := []string{
		"https://app.example",
		"http://app.example/x",
		"javascript:alert(1)",
		"/relative",
		"",
		"ftp://x.y/",
		"https://u:p@a.b/",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		got := sanitizeFrontendURL(raw)
		if got == "" {
			return
		}
		lo := strings.ToLower(got)
		if !(strings.HasPrefix(lo, "http://") || strings.HasPrefix(lo, "https://")) {
			t.Fatalf("non-absolute http(s) output: raw=%q got=%q", raw, got)
		}
	})
}
