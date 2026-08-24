package oauth2provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAuthMethodForPublicMatchesTheFlag(t *testing.T) {
	assert.Equal(t, "none", authMethodForPublic(true))
	assert.Equal(t, "client_secret_basic", authMethodForPublic(false))
}
