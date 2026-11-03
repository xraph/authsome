package oauth2provider_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/oauth2provider"
	"github.com/xraph/authsome/store"
	"github.com/xraph/authsome/user"
)

var (
	signingKeyOnce sync.Once
	signingKeyPEM  []byte
	errSigningKey  error
)

// testSigningKeyPEM is an RSA-2048 operator key for fixtures whose grants
// include openid and so mint an id_token. It is generated once per test
// binary because RSA key generation is slow.
func testSigningKeyPEM(t *testing.T) []byte {
	t.Helper()
	signingKeyOnce.Do(func() {
		k, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			errSigningKey = err
			return
		}
		der, err := x509.MarshalPKCS8PrivateKey(k)
		if err != nil {
			errSigningKey = err
			return
		}
		signingKeyPEM = pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
	})
	require.NoError(t, errSigningKey)
	return signingKeyPEM
}

// giveSigningKey hands a plugin that never runs OnInit a usable id_token key.
func giveSigningKey(t *testing.T, p *oauth2provider.Plugin) {
	t.Helper()
	require.NoError(t, p.UseSigningKeyForTest(testSigningKeyPEM(t)))
}

// fixtureUserID is the user authorize signs in as. Fixtures seed it into
// their core store so an openid grant can load the id_token's subject.
var fixtureUserID = func() id.UserID {
	u, err := id.ParseUserID("ausr_01jf0000000000000000000001")
	if err != nil {
		panic(err)
	}
	return u
}()

// seedFixtureUser stores the user authorize signs in as, in appID.
func seedFixtureUser(t *testing.T, core store.Store, appID id.AppID) {
	t.Helper()
	require.NoError(t, core.CreateUser(context.Background(), &user.User{
		ID: fixtureUserID, AppID: appID, Email: "fixture@example.com", FirstName: "Fixture",
	}))
}
