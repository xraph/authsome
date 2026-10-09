package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xraph/forge"

	"github.com/xraph/authsome/middleware"
)

func TestCSRF(t *testing.T) {
	cfg := middleware.CSRFConfig{Enabled: true, AllowedOrigins: []string{"https://app.example.com"}}
	cases := []struct {
		name    string
		method  string
		scheme  string
		headers map[string]string
		want    int
	}{
		{"cookie post, no metadata", "POST", "cookie", nil, http.StatusForbidden},
		{"cookie post, cross-site", "POST", "cookie", map[string]string{"Sec-Fetch-Site": "cross-site", "Origin": "https://evil.example"}, http.StatusForbidden},
		{"cookie post, same-origin", "POST", "cookie", map[string]string{"Sec-Fetch-Site": "same-origin"}, http.StatusOK},
		{"cookie post, user-initiated", "POST", "cookie", map[string]string{"Sec-Fetch-Site": "none"}, http.StatusOK},
		{"cookie post, allowlisted origin", "POST", "cookie", map[string]string{"Sec-Fetch-Site": "cross-site", "Origin": "https://App.Example.com"}, http.StatusOK},
		{"cookie post, origin is this host", "POST", "cookie", map[string]string{"Origin": "http://api.test"}, http.StatusOK},
		{"cookie delete, null origin", "DELETE", "cookie", map[string]string{"Origin": "null"}, http.StatusForbidden},
		{"cookie get is safe", "GET", "cookie", nil, http.StatusOK},
		{"bearer post is untouched", "POST", "bearer", nil, http.StatusOK},
		{"anonymous post is untouched", "POST", "", nil, http.StatusOK},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			router := forge.NewRouter()
			router.Use(func(next forge.Handler) forge.Handler {
				return func(ctx forge.Context) error {
					if tc.scheme != "" {
						ctx.WithContext(middleware.WithCredentialScheme(ctx.Context(), tc.scheme))
					}
					return next(ctx)
				}
			})
			router.Use(middleware.CSRF(cfg))
			handler := func(ctx forge.Context) error { return ctx.NoContent(http.StatusOK) }
			router.POST("/act", handler)
			router.DELETE("/act", handler)
			router.GET("/act", handler)

			req := httptest.NewRequestWithContext(context.Background(), tc.method, "http://api.test/act", nil)
			for k, v := range tc.headers {
				req.Header.Set(k, v)
			}
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			assert.Equal(t, tc.want, rec.Code, rec.Body.String())
		})
	}

	off := forge.NewRouter()
	off.Use(func(next forge.Handler) forge.Handler {
		return func(ctx forge.Context) error {
			ctx.WithContext(middleware.WithCredentialScheme(ctx.Context(), "cookie"))
			return next(ctx)
		}
	})
	off.Use(middleware.CSRF(middleware.CSRFConfig{Enabled: false}))
	off.POST("/act", func(ctx forge.Context) error { return ctx.NoContent(http.StatusOK) })
	req := httptest.NewRequestWithContext(context.Background(), "POST", "http://api.test/act", nil)
	rec := httptest.NewRecorder()
	off.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code, "disabled means no check")
}
