package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xraph/forge/extensions/auth"

	authmw "github.com/xraph/authsome/middleware"
)

func TestExplicitFrameCredentialExtraction(t *testing.T) {
	for _, tt := range []struct {
		name, header, markerScheme, markerToken, wantScheme, wantToken string
	}{
		{"ordinary bridge", "Bearer cookie-token", "", "", "cookie", "cookie-token"},
		{"explicit equal token", "Bearer cookie-token", "Bearer", "cookie-token", "bearer", "cookie-token"},
		{"explicit DPoP", "DPoP cookie-token", "DPoP", "cookie-token", "dpop", "cookie-token"},
		{"stale token", "Bearer cookie-token", "Bearer", "old-token", "cookie", "cookie-token"},
		{"token case mismatch", "Bearer cookie-token", "Bearer", "Cookie-token", "cookie", "cookie-token"},
		{"scheme mismatch", "Bearer cookie-token", "DPoP", "cookie-token", "cookie", "cookie-token"},
		{"absent header", "", "Bearer", "cookie-token", "cookie", "cookie-token"},
		{"malformed header", "Bearer", "Bearer", "cookie-token", "cookie", "cookie-token"},
		{"unsupported header", "Basic cookie-token", "Bearer", "cookie-token", "cookie", "cookie-token"},
		{"empty header token", "Bearer ", "Bearer", "cookie-token", "bearer", ""},
		{"unrelated explicit token", "Bearer other-token", "Bearer", "cookie-token", "bearer", "other-token"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
			req.Header.Set("Authorization", tt.header)
			req.AddCookie(&http.Cookie{Name: "authsome_session_token", Value: "cookie-token"})
			ctx := authmw.WithCookieBridgedToken(req.Context(), "cookie-token")
			if tt.markerScheme != "" {
				var err error
				ctx, err = auth.WithExplicitFrameCredential(ctx, tt.markerScheme, tt.markerToken)
				require.NoError(t, err)
			}
			scheme, token := authmw.ExtractCredentialFromContext(ctx, req, "")
			require.Equal(t, tt.wantScheme, scheme)
			require.Equal(t, tt.wantToken, token)
		})
	}
}
