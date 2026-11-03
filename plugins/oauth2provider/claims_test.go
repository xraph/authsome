package oauth2provider

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/user"
)

func only(scopes ...string) func(string) bool {
	return func(s string) bool {
		for _, have := range scopes {
			if have == s {
				return true
			}
		}
		return false
	}
}

func TestStandardClaims_FilterByScope(t *testing.T) {
	u := &user.User{FirstName: "Ada", LastName: "Lovelace", Email: "ada@example.com", Phone: "+15550100"}
	c := standardClaims(u, only("email"))
	assert.Equal(t, "ada@example.com", c.Email)
	assert.Empty(t, c.Name)
	assert.Empty(t, c.PhoneNumber)
	assert.Nil(t, c.PhoneNumberVerified)
}

func TestStandardClaims_VerifiedFalseIsPresent(t *testing.T) {
	u := &user.User{Email: "ada@example.com", EmailVerified: false}
	out, err := json.Marshal(standardClaims(u, only("email")))
	require.NoError(t, err)
	assert.Contains(t, string(out), `"email_verified":false`)
}

func TestStandardClaims_ProfileMapping(t *testing.T) {
	u := &user.User{
		FirstName: "Ada", LastName: "Lovelace", Image: "https://x/a.png", Username: "ada",
		UpdatedAt: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
	}
	c := standardClaims(u, only("profile"))
	assert.Equal(t, "Ada Lovelace", c.Name)
	assert.Equal(t, "Ada", c.GivenName)
	assert.Equal(t, "Lovelace", c.FamilyName)
	assert.Equal(t, "https://x/a.png", c.Picture)
	assert.Equal(t, "ada", c.PreferredUsername)
	assert.Equal(t, u.UpdatedAt.Unix(), c.UpdatedAt)
}

func TestSupportedClaims_ListsEveryStandardClaim(t *testing.T) {
	got := supportedClaims()
	for _, want := range []string{"sub", "iss", "aud", "exp", "iat", "auth_time", "nonce", "azp", "at_hash",
		"name", "given_name", "family_name", "picture", "preferred_username", "updated_at",
		"email", "email_verified", "phone_number", "phone_number_verified"} {
		assert.Contains(t, got, want)
	}
}
