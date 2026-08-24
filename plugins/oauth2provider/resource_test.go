package oauth2provider

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResourceParams(t *testing.T) {
	tests := []struct {
		name string
		req  func() *http.Request
		want []string
	}{
		{
			name: "single query value",
			req: func() *http.Request {
				return httptest.NewRequest(http.MethodGet,
					"/authorize?resource=https%3A%2F%2Fapi.example.com", nil)
			},
			want: []string{"https://api.example.com"},
		},
		{
			name: "repeated query values are all kept",
			req: func() *http.Request {
				return httptest.NewRequest(http.MethodGet,
					"/authorize?resource=https%3A%2F%2Fa.example.com&resource=https%3A%2F%2Fb.example.com", nil)
			},
			want: []string{"https://a.example.com", "https://b.example.com"},
		},
		{
			name: "absent parameter yields nothing",
			req: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, "/authorize?client_id=abc", nil)
			},
			want: nil,
		},
		{
			name: "repeated form values on a POST",
			req: func() *http.Request {
				body := "grant_type=authorization_code&resource=https%3A%2F%2Fa.example.com&resource=https%3A%2F%2Fb.example.com"
				r := httptest.NewRequest(http.MethodPost, "/token", strings.NewReader(body))
				r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				return r
			},
			want: []string{"https://a.example.com", "https://b.example.com"},
		},
		{
			name: "query and form values are both collected on a POST",
			req: func() *http.Request {
				body := "resource=https%3A%2F%2Fb.example.com"
				r := httptest.NewRequest(http.MethodPost,
					"/token?resource=https%3A%2F%2Fa.example.com", strings.NewReader(body))
				r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				return r
			},
			want: []string{"https://a.example.com", "https://b.example.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, resourceParams(tt.req()))
		})
	}
}

func TestResolveResources(t *testing.T) {
	allowed := &OAuth2Client{
		Resources: []string{"https://api.example.com", "https://files.example.com"},
	}
	noAllowlist := &OAuth2Client{}

	tests := []struct {
		name      string
		client    *OAuth2Client
		requested []string
		want      []string
		wantErr   bool
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
			name:      "an unregistered resource is rejected",
			client:    allowed,
			requested: []string{"https://evil.example.com"},
			wantErr:   true,
		},
		{
			name:      "an empty allowlist rejects any request",
			client:    noAllowlist,
			requested: []string{"https://api.example.com"},
			wantErr:   true,
		},
		{
			name:      "an empty allowlist still allows an empty request",
			client:    noAllowlist,
			requested: nil,
			want:      nil,
		},
		{
			name:      "a relative URI is rejected",
			client:    allowed,
			requested: []string{"/api"},
			wantErr:   true,
		},
		{
			name:      "a URI carrying a fragment is rejected",
			client:    allowed,
			requested: []string{"https://api.example.com#section"},
			wantErr:   true,
		},
		{
			name:      "an empty string is rejected",
			client:    allowed,
			requested: []string{""},
			wantErr:   true,
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
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
