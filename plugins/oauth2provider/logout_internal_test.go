package oauth2provider

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"

	"github.com/xraph/authsome/id"
)

func TestVerifyIDTokenHint(t *testing.T) {
	ctx := context.Background()
	p := New(Config{Issuer: "https://auth.example.com"})
	p.SetOAuth2Store(NewMemoryStore())
	p.keys = newTestKeyManager(t, p.config, p.oauth2Store, testEncryptor(t))
	require.NoError(t, p.oauth2Store.CreateClient(ctx, &OAuth2Client{ID: id.NewOAuth2ClientID(), AppID: id.NewAppID(), ClientID: "rp"}))
	active, err := p.keys.Active(ctx, "")
	require.NoError(t, err)

	sign := func(s *signer, over jwt.MapClaims) string {
		claims := jwt.MapClaims{"iss": "https://auth.example.com", "aud": "rp", "sub": "u-1", "exp": time.Now().Add(-time.Hour).Unix()}
		for k, v := range over {
			if v == nil {
				delete(claims, k)
				continue
			}
			claims[k] = v
		}
		tok, serr := signIDToken(s, claims)
		require.NoError(t, serr)
		return tok
	}

	sub, client, err := p.verifyIDTokenHint(ctx, sign(active, nil))
	require.NoError(t, err, "an expired hint still counts")
	assert.Equal(t, "u-1", sub)
	assert.Equal(t, "rp", client.ClientID)

	sub, _, err = p.verifyIDTokenHint(ctx, sign(active, jwt.MapClaims{"aud": []string{"other"}, "azp": "rp"}))
	require.Error(t, err, "azp names the client but the audience leaves it out")
	assert.Empty(t, sub)

	ec, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	for name, tok := range map[string]string{
		"too long":          strings.Repeat("a", maxIDTokenHintBytes+1),
		"not a jwt":         "nope",
		"no client":         sign(active, jwt.MapClaims{"aud": []string{"a", "b"}}),
		"unknown client":    sign(active, jwt.MapClaims{"aud": "nobody"}),
		"wrong issuer":      sign(active, jwt.MapClaims{"iss": "https://evil.example.com"}),
		"no subject":        sign(active, jwt.MapClaims{"sub": nil}),
		"unknown key":       sign(&signer{kid: "someone-else", alg: active.alg, key: active.key}, nil),
		"alg doesn't match": sign(&signer{kid: active.kid, alg: "ES256", key: ec}, nil),
	} {
		t.Run(name, func(t *testing.T) {
			_, _, verr := p.verifyIDTokenHint(ctx, tok)
			assert.Error(t, verr)
		})
	}

	p.keys = nil
	_, _, err = p.verifyIDTokenHint(ctx, "a.b.c")
	assert.ErrorIs(t, err, errNoSigningKey)
}

func TestWithState(t *testing.T) {
	got, err := withState("https://rp.example.com/bye?x=1", "")
	require.NoError(t, err)
	assert.Equal(t, "https://rp.example.com/bye?x=1", got)
	_, err = withState("%zz", "s")
	assert.Error(t, err)
}

func TestLogoutConfirm_BadRequests(t *testing.T) {
	ctx := context.Background()
	p := New(Config{Issuer: "https://auth.example.com"})
	p.SetOAuth2Store(NewMemoryStore())
	mux := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(mux))
	require.NoError(t, p.ceremonyStore().Set(ctx, logoutKey("corrupt"), []byte("{"), time.Minute))

	post := func(path, body string) int {
		req := httptest.NewRequestWithContext(ctx, http.MethodPost, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		return rec.Code
	}
	assert.Equal(t, http.StatusBadRequest, post("/v1/oauth/logout/confirm", "lid=%zz"), "a form that won't parse")
	assert.Equal(t, http.StatusBadRequest, post("/v1/oauth/logout/confirm", "csrf=x"), "no lid")
	assert.Equal(t, http.StatusBadRequest, post("/v1/oauth/logout/confirm", "lid=corrupt&csrf=x"), "a parked request that won't decode")
	assert.Equal(t, http.StatusBadRequest, post("/v1/oauth/logout", "state=%zz"), "logout form that won't parse")
}
