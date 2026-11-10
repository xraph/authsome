package oauth2provider_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/oauth2provider"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/store"
	"github.com/xraph/authsome/user"
)

// planBlindStore is the plugin's view of the core store with the refresh
// token lookup failing, so the refresh grant can't plan its id_token ahead
// of the rotation. failUser also fails every user lookup.
type planBlindStore struct {
	store.Store
	failUser bool
}

func (planBlindStore) GetSessionByRefreshToken(context.Context, string) (*session.Session, error) {
	return nil, store.ErrNotFound
}

func (s planBlindStore) GetUser(ctx context.Context, userID id.UserID) (*user.User, error) {
	if s.failUser {
		return nil, errors.New("user store is down")
	}
	return s.Store.GetUser(ctx, userID)
}

// When the id_token could not be planned before the rotation, the refresh
// grant still mints one for an openid session after it.
func TestRefreshGrant_UnplannedIDTokenIsMintedAfterRotation(t *testing.T) {
	f := newOIDCFixture(t, oauth2provider.Config{}, true)
	f.p.SetStore(planBlindStore{Store: f.eng.Store()})
	sess := seedProviderSession(t, f, f.appID, publicID, "openid")

	body := decodeTokenBody(t, postToken(t, f.mux, map[string]string{
		"grant_type": "refresh_token", "refresh_token": sess.RefreshToken, "client_id": publicID,
	}))
	raw, ok := body["id_token"].(string)
	require.True(t, ok, "the refresh carries an id_token: %v", body)
	_, claims := verifyIDToken(t, f.mux, raw)
	assert.Equal(t, f.user.ID.String(), claims["sub"])
	assert.NotContains(t, claims, "nonce")
	assert.NotContains(t, claims, "auth_time")

	// The rotation keeps the session, so the revocation the failure case
	// checks for is not just the rotation itself.
	_, err := f.eng.Store().GetSession(context.Background(), sess.ID)
	require.NoError(t, err)
}

// If that late mint fails, the rotated session is revoked rather than handed
// out without the id_token the client asked for.
func TestRefreshGrant_UnplannedIDTokenFailureRevokesSession(t *testing.T) {
	f := newOIDCFixture(t, oauth2provider.Config{}, true)
	f.p.SetStore(planBlindStore{Store: f.eng.Store(), failUser: true})
	sess := seedProviderSession(t, f, f.appID, publicID, "openid")

	rec := postToken(t, f.mux, map[string]string{
		"grant_type": "refresh_token", "refresh_token": sess.RefreshToken, "client_id": publicID,
	})
	require.Equal(t, http.StatusInternalServerError, rec.Code, "body: %s", rec.Body.String())
	assert.NotContains(t, rec.Body.String(), "access_token")
	assert.NotContains(t, rec.Body.String(), "user store is down")

	_, err := f.eng.Store().GetSession(context.Background(), sess.ID)
	assert.ErrorIs(t, err, store.ErrNotFound, "the rotated session was revoked")
	_, err = f.eng.Store().GetSessionByRefreshToken(context.Background(), sess.RefreshToken)
	assert.Error(t, err, "the old refresh token was spent by the rotation")
}
