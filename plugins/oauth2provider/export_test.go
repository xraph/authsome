package oauth2provider

import (
	"context"

	"github.com/xraph/authsome/bridge"
)

// UseSigningKeyForTest makes keyPEM the operator id_token key and builds the
// key manager the way OnInit does, for fixtures that never run OnInit.
func (p *Plugin) UseSigningKeyForTest(keyPEM []byte) error {
	p.config.IDTokenSigningKey = keyPEM
	return p.initSigningKeys(context.Background(), bridge.NoopEncryptor{})
}
