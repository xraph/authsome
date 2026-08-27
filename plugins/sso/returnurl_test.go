package sso

import "testing"

// TestIsAllowedReturnURL covers the return-URL allowlist, including the wildcard
// suffix used for stay-on-domain tenant SSO (https://*.kineta.ai).
func TestIsAllowedReturnURL(t *testing.T) {
	p := &Plugin{config: Config{AllowedReturnOrigins: []string{
		"https://app.kineta.ai",
		"https://*.kineta.ai",
	}}}

	cases := []struct {
		name string
		url  string
		want bool
	}{
		{"exact apex", "https://app.kineta.ai/sso/callback", true},
		{"wildcard subdomain", "https://acme.kineta.ai/sso/callback?ws=aorg_1", true},
		{"wildcard subdomain, other tenant", "https://dental.kineta.ai/sso/callback", true},
		{"localhost always allowed", "http://localhost:3300/sso/callback", true},
		{"127.0.0.1 always allowed", "http://127.0.0.1:3300/sso/callback", true},
		{"unrelated https origin", "https://evil.com/sso/callback", false},
		{"lookalike suffix is not a subdomain", "https://evil-kineta.ai/sso/callback", false},
		{"suffix-embedded elsewhere", "https://kineta.ai.evil.com/sso/callback", false},
		{"non-https (not localhost)", "http://acme.kineta.ai/sso/callback", false},
		{"garbage", "::::", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := p.isAllowedReturnURL(c.url); got != c.want {
				t.Fatalf("isAllowedReturnURL(%q) = %v, want %v", c.url, got, c.want)
			}
		})
	}
}

// TestIsAllowedReturnURL_NoWildcardConfigured confirms a plain allowlist keeps
// exact-match semantics (no accidental wildcarding).
func TestIsAllowedReturnURL_NoWildcardConfigured(t *testing.T) {
	p := &Plugin{config: Config{AllowedReturnOrigins: []string{"https://app.kineta.ai"}}}
	if p.isAllowedReturnURL("https://acme.kineta.ai/sso/callback") {
		t.Fatal("subdomain must not be allowed without a wildcard entry")
	}
	if !p.isAllowedReturnURL("https://app.kineta.ai/x") {
		t.Fatal("exact apex must be allowed")
	}
}
