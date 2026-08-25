package authprovider_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	log "github.com/xraph/go-utils/log"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/authprovider"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/principal"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/user"
)

func bearerRequest(t *testing.T, token string) *http.Request {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/x", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	return req
}

// A machine principal has no user row, so resolving one and treating failure
// as an authentication failure turned away every credential of this kind on
// any route behind forge.WithGroupAuth("session").
func TestSessionProvider_MachinePrincipalAuthenticates(t *testing.T) {
	for _, kind := range []principal.Kind{
		principal.KindService,
		principal.KindWorkload,
		principal.KindAgent,
	} {
		t.Run(string(kind), func(t *testing.T) {
			svcID := id.NewServiceAccountID()
			sess := &session.Session{
				ID:               id.NewSessionID(),
				AppID:            id.NewAppID(),
				PrincipalKind:    kind,
				ServiceAccountID: svcID,
				Roles:            []string{"deployer"},
			}

			p := authprovider.NewSessionProvider(
				func(token string) (*session.Session, error) {
					if token == "machine-token" {
						return sess, nil
					}
					return nil, errors.New("not found")
				},
				func(string) (*user.User, error) {
					t.Fatal("resolveUser must not be called for a machine principal")
					return nil, nil
				},
				log.NewNoopLogger(),
			)

			authCtx, err := p.Authenticate(context.Background(), bearerRequest(t, "machine-token"))
			require.NoError(t, err, "a %s session must authenticate", kind)
			assert.Equal(t, svcID.String(), authCtx.Subject)
			assert.Equal(t, []string{"deployer"}, authCtx.Roles)
			assert.Equal(t, string(kind), authCtx.Claims["principal_kind"])
			assert.Equal(t, svcID.String(), authCtx.Claims["principal_id"])

			data, ok := authCtx.Data.(*authprovider.SessionData)
			require.True(t, ok)
			assert.Nil(t, data.User, "there is no user behind a machine principal")
			require.NotNil(t, data.Session)
			assert.Equal(t, sess.ID, data.Session.ID)
		})
	}
}

// The regression guard. A human session must authenticate exactly as before,
// carrying the user and the profile claims handlers already read.
func TestSessionProvider_UserPathUnchanged(t *testing.T) {
	appID := id.NewAppID()
	userID := id.NewUserID()
	sess := &session.Session{
		ID:     id.NewSessionID(),
		AppID:  appID,
		UserID: userID,
		Roles:  []string{"admin"},
	}
	u := &user.User{ID: userID, AppID: appID, Email: "a@b.com", FirstName: "Ada"}

	p := authprovider.NewSessionProvider(
		func(string) (*session.Session, error) { return sess, nil },
		func(idStr string) (*user.User, error) {
			require.Equal(t, userID.String(), idStr)
			return u, nil
		},
		log.NewNoopLogger(),
	)

	authCtx, err := p.Authenticate(context.Background(), bearerRequest(t, "human-token"))
	require.NoError(t, err)
	assert.Equal(t, userID.String(), authCtx.Subject)
	assert.Equal(t, []string{"admin"}, authCtx.Roles)
	assert.Equal(t, "a@b.com", authCtx.Claims["email"])
	assert.Equal(t, "Ada", authCtx.Claims["first_name"])

	data, ok := authCtx.Data.(*authprovider.SessionData)
	require.True(t, ok)
	require.NotNil(t, data.User)
	assert.Equal(t, userID, data.User.ID)
}

// A user session whose user cannot be resolved is still an authentication
// failure. The fix must not widen into "any session at all will do".
func TestSessionProvider_UserSessionWithMissingUserStillFails(t *testing.T) {
	sess := &session.Session{
		ID:     id.NewSessionID(),
		AppID:  id.NewAppID(),
		UserID: id.NewUserID(),
	}

	p := authprovider.NewSessionProvider(
		func(string) (*session.Session, error) { return sess, nil },
		func(string) (*user.User, error) { return nil, errors.New("gone") },
		log.NewNoopLogger(),
	)

	_, err := p.Authenticate(context.Background(), bearerRequest(t, "orphan-token"))
	require.Error(t, err)
}
