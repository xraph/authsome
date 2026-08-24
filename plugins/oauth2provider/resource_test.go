package oauth2provider

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
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
