package account_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/appsessionconfig"
)

func TestAppConfigOverridesTokenExchangeTTL(t *testing.T) {
	base := account.SessionConfig{TokenExchangeTTL: 5 * time.Minute}
	secs := 120
	(&appsessionconfig.Config{TokenExchangeTTLSeconds: &secs}).ApplyTo(&base)
	assert.Equal(t, 2*time.Minute, base.TokenExchangeTTL)
}

func TestNilAppConfigLeavesTokenExchangeTTL(t *testing.T) {
	base := account.SessionConfig{TokenExchangeTTL: 5 * time.Minute}
	(&appsessionconfig.Config{}).ApplyTo(&base)
	assert.Equal(t, 5*time.Minute, base.TokenExchangeTTL)
}
