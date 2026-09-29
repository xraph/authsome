package scim

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/apikey"
	"github.com/xraph/authsome/hook"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/session"
)

// An IdP deactivation is a ban by another name: the user's sessions and
// keys end with it, and the trail records it as a critical entry.

func (f *scopeFixture) credentials(t *testing.T, uid id.UserID) id.APIKeyID {
	t.Helper()
	ctx := context.Background()
	now := time.Now()
	require.NoError(t, f.eng.Store().CreateSession(ctx, &session.Session{
		ID: id.NewSessionID(), AppID: f.appID, UserID: uid, Token: "tok-" + uid.String(),
		CreatedAt: now, ExpiresAt: now.Add(time.Hour),
	}))
	_, hash, prefix, err := apikey.GenerateKey()
	require.NoError(t, err)
	keyID := id.NewAPIKeyID()
	require.NoError(t, f.eng.APIKeyStore().CreateAPIKey(ctx, &apikey.APIKey{
		ID: keyID, AppID: f.appID, UserID: uid, Name: "k", KeyHash: hash, KeyPrefix: prefix,
		CreatedAt: now, UpdatedAt: now,
	}))
	return keyID
}

func (f *scopeFixture) assertRevoked(t *testing.T, uid id.UserID, keyID id.APIKeyID) {
	t.Helper()
	sessions, err := f.eng.Store().ListUserSessions(context.Background(), uid)
	require.NoError(t, err)
	assert.Empty(t, sessions, "sessions must end with the deactivation")
	k, err := f.eng.APIKeyStore().GetAPIKey(context.Background(), keyID)
	require.NoError(t, err)
	assert.True(t, k.Revoked, "keys must end with the deactivation")
}

func TestSCIM_Delete_RevokesSessionsKeysAndRecords(t *testing.T) {
	f := newScopeFixture(t)
	tok := f.token(t, id.Nil)
	u := f.user(t, f.appID, "deactivate@example.com")
	keyID := f.credentials(t, u.ID)

	var recorded []*hook.Event
	f.eng.Hooks().On("deactivate-test", func(_ context.Context, ev *hook.Event) error {
		if ev.Action == hook.ActionAdminBanUser {
			recorded = append(recorded, ev)
		}
		return nil
	})

	rec := f.do(t, tok, http.MethodDelete, "/Users/"+u.ID.String(), nil)
	require.Equal(t, http.StatusNoContent, rec.Code, "body=%s", rec.Body.String())
	f.assertRevoked(t, u.ID, keyID)
	require.Len(t, recorded, 1, "one critical trail entry per deactivation")
	assert.Equal(t, "scim", recorded[0].Metadata["source"])
	assert.Equal(t, hook.SeverityWarning, recorded[0].Severity)
}

func TestSCIM_PatchInactive_RevokesSessionsAndKeys(t *testing.T) {
	f := newScopeFixture(t)
	tok := f.token(t, id.Nil)
	u := f.user(t, f.appID, "patch-inactive@example.com")
	keyID := f.credentials(t, u.ID)

	rec := f.do(t, tok, http.MethodPatch, "/Users/"+u.ID.String(), deactivate())
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	f.assertRevoked(t, u.ID, keyID)
}

func TestSCIM_PutInactive_RevokesSessionsAndKeys(t *testing.T) {
	f := newScopeFixture(t)
	tok := f.token(t, id.Nil)
	u := f.user(t, f.appID, "put-inactive@example.com")
	keyID := f.credentials(t, u.ID)

	rec := f.do(t, tok, http.MethodPut, "/Users/"+u.ID.String(), UserResource{
		Schemas: []string{SchemaUser}, UserName: u.Email,
		Emails: []Email{{Value: u.Email, Primary: true}}, Active: false,
	})
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	f.assertRevoked(t, u.ID, keyID)
}
