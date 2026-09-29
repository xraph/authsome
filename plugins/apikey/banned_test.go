package apikey_test

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/apikey"
	"github.com/xraph/authsome/id"
	apikeyPlugin "github.com/xraph/authsome/plugins/apikey"
	"github.com/xraph/authsome/principal"
	"github.com/xraph/authsome/user"
)

// A key stays a key after its owner is banned or its service account is
// disabled; the strategy has to check the owner, not only the key row.

type bannedUserEngine struct{ *mockEngine }

func (bannedUserEngine) ResolveUser(userID string) (*user.User, error) {
	uid, err := id.ParseUserID(userID)
	if err != nil {
		return nil, err
	}
	return &user.User{ID: uid, Email: "banned@example.com", Banned: true}, nil
}

func mintKey(t *testing.T, store apikey.Store, appID id.AppID, userID id.UserID, saID id.ServiceAccountID) string {
	t.Helper()
	raw, hash, prefix, err := apikey.GenerateKey()
	require.NoError(t, err)
	now := time.Now()
	require.NoError(t, store.CreateAPIKey(context.Background(), &apikey.APIKey{
		ID: id.NewAPIKeyID(), AppID: appID, UserID: userID, ServiceAccountID: saID,
		Name: "k", KeyHash: hash, KeyPrefix: prefix, CreatedAt: now, UpdatedAt: now,
	}))
	return raw
}

func authWith(t *testing.T, p *apikeyPlugin.Plugin, raw string, appID id.AppID) error {
	t.Helper()
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/api/data", nil)
	req.Header.Set("Authorization", "Bearer "+raw)
	req.Header.Set("X-App-ID", appID.String())
	_, err := p.Strategy().Authenticate(context.Background(), req)
	return err
}

func TestStrategy_BannedUserKeyIsRefused(t *testing.T) {
	p, store := newTestPlugin()
	eng := bannedUserEngine{&mockEngine{logger: log.NewNoopLogger(), store: store}}
	require.NoError(t, p.OnInit(context.Background(), eng))

	appID := id.NewAppID()
	raw := mintKey(t, store, appID, id.NewUserID(), id.Nil)
	err := authWith(t, p, raw, appID)
	assert.ErrorContains(t, err, "banned")
}

func TestStrategy_DisabledServiceAccountKeyIsRefused(t *testing.T) {
	p, store := newTestPlugin()
	saID := id.NewServiceAccountID()
	eng := &mockEngine{logger: log.NewNoopLogger(), store: store, principals: map[string]*principal.Principal{
		saID.String(): {Ref: principal.Ref{Kind: principal.KindService, ID: saID.String()}, Disabled: true},
	}}
	require.NoError(t, p.OnInit(context.Background(), eng))

	appID := id.NewAppID()
	raw := mintKey(t, store, appID, id.Nil, saID)
	err := authWith(t, p, raw, appID)
	assert.ErrorContains(t, err, "not active")
}

func TestStrategy_ActiveServiceAccountKeyStillWorks(t *testing.T) {
	p, store := newTestPlugin()
	saID := id.NewServiceAccountID()
	eng := &mockEngine{logger: log.NewNoopLogger(), store: store, principals: map[string]*principal.Principal{
		saID.String(): {Ref: principal.Ref{Kind: principal.KindService, ID: saID.String()}},
	}}
	require.NoError(t, p.OnInit(context.Background(), eng))

	appID := id.NewAppID()
	raw := mintKey(t, store, appID, id.Nil, saID)
	assert.NoError(t, authWith(t, p, raw, appID))
}
