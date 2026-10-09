package authprovider_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/authprovider"
	"github.com/xraph/authsome/id"
	authmw "github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/user"
)

func TestSessionProviderCredentialProvenance(t *testing.T) {
	sess := &session.Session{ID: id.NewSessionID(), AppID: id.NewAppID(), UserID: id.NewUserID()}
	provider := authprovider.NewSessionProvider(func(context.Context, string) (*session.Session, error) { return sess, nil }, func(context.Context, string) (*user.User, error) { return &user.User{ID: sess.UserID}, nil }, log.NewNoopLogger())
	for _, scheme := range []string{"bearer", "dpop", "cookie", "bridged-cookie"} {
		t.Run(scheme, func(t *testing.T) {
			req := bearerRequest(t, "token")
			ctx := req.Context()
			switch scheme {
			case "dpop":
				req.Header.Set("Authorization", "DPoP token")
			case "cookie":
				req.Header.Del("Authorization")
				req.AddCookie(&http.Cookie{Name: authprovider.DefaultSessionCookieName, Value: "token"})
			case "bridged-cookie":
				ctx = authmw.WithCookieBridgedToken(ctx, "token")
			}
			result, err := provider.Authenticate(ctx, req)
			require.NoError(t, err)
			want := scheme
			if scheme == "bridged-cookie" {
				want = "cookie"
			}
			require.Equal(t, want, result.Metadata["credential_scheme"])
		})
	}
}
