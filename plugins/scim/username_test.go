package scim

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/forge"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/secutil"
	"github.com/xraph/authsome/user"
)

// A SCIM userName change is a primary email change: it goes through the
// email records, so the new address is a verified record the user owns, an
// address another account holds is refused, and the old address remains one
// of the user's records rather than vanishing.

func TestHandlePatchUser_UserNameGoesThroughEmailRecords(t *testing.T) {
	p := New()
	eng := secutil.NewTestEngine(t, authsome.WithPlugin(p))
	router := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(router))
	h := router.Handler()
	appID, err := id.ParseAppID(scimLifecycleTestAppID)
	require.NoError(t, err)
	ctx := context.Background()

	cfg := &SCIMConfig{ID: id.NewSCIMConfigID(), Name: "okta", AppID: appID, Enabled: true}
	require.NoError(t, p.service.CreateConfig(ctx, cfg))
	token, _, err := p.service.GenerateToken(ctx, cfg.ID, "t", nil)
	require.NoError(t, err)

	mkUser := func(email string) *user.User {
		u := &user.User{ID: id.NewUserID(), AppID: appID, Email: email, EmailVerified: true, CreatedAt: time.Now(), UpdatedAt: time.Now()}
		require.NoError(t, eng.Store().CreateUserWithPrimaryEmail(ctx, u, user.NewPrimaryEmail(u, "scim")))
		return u
	}
	u := mkUser("alice@corp.example")
	other := mkUser("taken@corp.example")

	patch := func(target *user.User, userName string) *httptest.ResponseRecorder {
		body, merr := json.Marshal(PatchOp{Schemas: []string{SchemaPatchOp}, Operations: []Operation{{Op: "replace", Path: "userName", Value: userName}}})
		require.NoError(t, merr)
		req := httptest.NewRequestWithContext(ctx, http.MethodPatch, p.config.BasePath+"/Users/"+target.ID.String(), bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/scim+json")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}

	rec := patch(u, "Alice.Renamed@corp.example")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	got, err := eng.Store().GetUser(ctx, u.ID)
	require.NoError(t, err)
	assert.Equal(t, "alice.renamed@corp.example", got.Email)
	emails, err := eng.Store().GetUserEmails(ctx, u.ID)
	require.NoError(t, err)
	var primary *user.UserEmail
	addresses := map[string]bool{}
	for _, e := range emails {
		addresses[e.Email] = true
		if e.IsPrimary {
			primary = e
		}
	}
	require.NotNil(t, primary)
	assert.Equal(t, "alice.renamed@corp.example", primary.Email)
	assert.True(t, primary.Verified, "the provisioning system vouches for the address")
	assert.Equal(t, "scim", primary.Source)
	assert.True(t, addresses["alice@corp.example"], "the previous address stays on the account")

	rec = patch(u, other.Email)
	assert.Equal(t, http.StatusNotFound, rec.Code, "an address another account owns is out of scope: %s", rec.Body.String())
	again, err := eng.Store().GetUser(ctx, u.ID)
	require.NoError(t, err)
	assert.Equal(t, "alice.renamed@corp.example", again.Email, "a refused change leaves the address alone")

	rec = patch(u, "alice.renamed@corp.example")
	assert.Equal(t, http.StatusOK, rec.Code, "the current address is a no-op")
}
