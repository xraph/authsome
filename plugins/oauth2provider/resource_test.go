package oauth2provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveResources(t *testing.T) {
	allowed := &OAuth2Client{
		Resources: []string{"https://api.example.com", "https://files.example.com"},
	}
	noAllowlist := &OAuth2Client{}

	tests := []struct {
		name        string
		client      *OAuth2Client
		requested   []string
		want        []string
		wantErr     bool
		wantErrDesc string
	}{
		{
			name:      "no resource requested yields no audience",
			client:    allowed,
			requested: nil,
			want:      nil,
		},
		{
			name:      "a registered resource is granted",
			client:    allowed,
			requested: []string{"https://api.example.com"},
			want:      []string{"https://api.example.com"},
		},
		{
			name:      "two registered resources are both granted",
			client:    allowed,
			requested: []string{"https://api.example.com", "https://files.example.com"},
			want:      []string{"https://api.example.com", "https://files.example.com"},
		},
		{
			name:      "duplicates collapse and order is preserved",
			client:    allowed,
			requested: []string{"https://files.example.com", "https://api.example.com", "https://files.example.com"},
			want:      []string{"https://files.example.com", "https://api.example.com"},
		},
		{
			name:        "an unregistered resource is rejected",
			client:      allowed,
			requested:   []string{"https://evil.example.com"},
			wantErr:     true,
			wantErrDesc: `resource "https://evil.example.com" is not registered for this client`,
		},
		{
			name:        "an empty allowlist rejects any request",
			client:      noAllowlist,
			requested:   []string{"https://api.example.com"},
			wantErr:     true,
			wantErrDesc: `resource "https://api.example.com" is not registered for this client`,
		},
		{
			name:      "an empty allowlist still allows an empty request",
			client:    noAllowlist,
			requested: nil,
			want:      nil,
		},
		{
			// The allowlist does not contain "/api" either, so a broken
			// implementation that dropped the IsAbs check would still reject
			// this value, just for the wrong reason (membership, not syntax).
			// Pinning the description on the specific "is not an absolute
			// URI" wording is what proves the syntax rule itself fired.
			name:        "a relative URI is rejected",
			client:      allowed,
			requested:   []string{"/api"},
			wantErr:     true,
			wantErrDesc: `resource "/api" is not an absolute URI`,
		},
		{
			// Same reasoning as the relative URI case above: the raw value
			// is not in the allowlist, so membership alone would reject it
			// too. Pinning the description proves the fragment rule fired,
			// not the membership check.
			name:        "a URI carrying a fragment is rejected",
			client:      allowed,
			requested:   []string{"https://api.example.com#section"},
			wantErr:     true,
			wantErrDesc: `resource "https://api.example.com#section" must not include a fragment`,
		},
		{
			name:        "an empty string is rejected",
			client:      allowed,
			requested:   []string{""},
			wantErr:     true,
			wantErrDesc: "resource must not be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveResources(tt.client, tt.requested)
			if tt.wantErr {
				require.Error(t, err)
				// OAuth2HTTPError.Error() returns only the description
				// (plugin.go:412); the registered error code lives on
				// ResponseBody(). Asserting on Error() would pass for any
				// rejection reason at all, including the wrong one.
				var httpErr *OAuth2HTTPError
				require.ErrorAs(t, err, &httpErr)
				body, ok := httpErr.ResponseBody().(*OAuth2Error)
				require.True(t, ok)
				assert.Equal(t, "invalid_target", body.Error)
				// The code alone does not say which branch rejected the
				// value: every rejection in resolveResources uses the same
				// invalid_target code. Pinning the description is what
				// proves the specific rule named by the subtest actually
				// fired, rather than a later check (such as allowlist
				// membership) catching the same input for a different
				// reason.
				assert.Equal(t, tt.wantErrDesc, body.ErrorDescription)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
