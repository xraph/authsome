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
	cases := []string{"/", "/dashboard", "/path?x=1"}
	for _, in := range cases {
		got := sanitizeRedirectURL(in, "")
		if got != in {
			t.Errorf("relative %q: got %q want %q", in, got, in)
		}
	}
}

func TestSanitizeRedirectURL_BackslashBypass(t *testing.T) {
	t.Parallel()
	cases := []string{
		`\\evil.example`,
		`\\evil.example/path`,
		`/\evil.example`,
		`/\\evil.example`,
		`\evil`,
	}
	for _, raw := range cases {
		if got := sanitizeRedirectURL(raw, ""); got != "" {
			t.Errorf("sanitizeRedirectURL(%q, \"\") = %q, want \"\" (browsers normalise \\\\ to //)", raw, got)
		}
		if got := sanitizeRedirectURL(raw, "https://app.example"); got != "" {
			t.Errorf("sanitizeRedirectURL(%q, %q) = %q, want \"\"", raw, "https://app.example", got)
		}
	}
}

func TestSanitizeRedirectURL_RelativeRequiresLeadingSlash(t *testing.T) {
	t.Parallel()
	// Inputs without a host AND without a leading '/' could be interpreted
	// against the current page in surprising ways. Reject for safety.
	if got := sanitizeRedirectURL("relative/path", ""); got != "" {
		t.Errorf("expected empty for non-rooted relative path; got %q", got)
	}
	// Empty path is allowed (caller falls back).
	if got := sanitizeRedirectURL("", ""); got != "" {
		t.Errorf("empty input should map to empty; got %q", got)
	}
	// Leading-slash relative still passes.
	if got := sanitizeRedirectURL("/dashboard", ""); got != "/dashboard" {
		t.Errorf("/dashboard should pass; got %q", got)
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
		{`\\evil.example`, ""},
		{`/\evil.example`, "https://app.example"},
		{"https://app.example#@evil.example", "https://app.example"},
	}
	for _, s := range seeds {
		f.Add(s.raw, s.origin)
	}
	f.Fuzz(func(t *testing.T, raw, origin string) {
		out := sanitizeRedirectURL(raw, origin)
		if out == "" {
			return
		}
		if strings.ContainsRune(out, '\\') {
			t.Fatalf("output contains backslash: %q (input %q origin %q)", out, raw, origin)
		}
		if !strings.HasPrefix(out, "/") && !strings.HasPrefix(out, "http://") && !strings.HasPrefix(out, "https://") {
			t.Fatalf("output not safe-prefixed: %q (input %q origin %q)", out, raw, origin)
		}
	})
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
		if !strings.HasPrefix(lo, "http://") && !strings.HasPrefix(lo, "https://") {
			t.Fatalf("non-absolute http(s) output: raw=%q got=%q", raw, got)
		}
	})
}
