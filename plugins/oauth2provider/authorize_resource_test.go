package oauth2provider_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/plugins/oauth2provider"
)

const (
	resAPI   = "https://api.example.com"
	resFiles = "https://files.example.com"
	resOther = "https://other.example.com"
)

// grantResources gives the confidential fixture client an allowlist. The
// fixture registers no resources, which is the deny-by-default state every
// existing client is in.
func grantResources(t *testing.T, st oauth2provider.Store, resources ...string) {
	t.Helper()
	c, err := st.GetClient(context.Background(), confidentialID)
	require.NoError(t, err)
	c.Resources = resources
	require.NoError(t, st.CreateClient(context.Background(), c))
}

func TestAuthorizeResource(t *testing.T) {
	t.Run("a registered resource lands on the code", func(t *testing.T) {
		_, st, mux := newFixture(t)
		grantResources(t, st, resAPI)

		q := baseAuthorizeQuery(confidentialID)
		q.Add("resource", resAPI)

		code := codeFrom(t, authorize(t, mux, q))

		stored, err := st.GetAuthCode(context.Background(), code)
		require.NoError(t, err)
		assert.Equal(t, []string{resAPI}, stored.Resources)
	})

	t.Run("two resources both land on the code", func(t *testing.T) {
		_, st, mux := newFixture(t)
		grantResources(t, st, resAPI, resFiles)

		q := baseAuthorizeQuery(confidentialID)
		q.Add("resource", resAPI)
		q.Add("resource", resFiles)

		code := codeFrom(t, authorize(t, mux, q))

		stored, err := st.GetAuthCode(context.Background(), code)
		require.NoError(t, err)
		assert.Equal(t, []string{resAPI, resFiles}, stored.Resources,
			"a repeated resource parameter lost a value, which means it went through the struct binder")
	})

	t.Run("an unregistered resource is refused", func(t *testing.T) {
		_, st, mux := newFixture(t)
		grantResources(t, st, resAPI)

		q := baseAuthorizeQuery(confidentialID)
		q.Add("resource", resOther)

		rec := authorize(t, mux, q)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "invalid_target")
		assert.Contains(t, rec.Body.String(), "not registered for this client",
			"an unregistered resource must be refused for that specific reason, not merely refused")
	})

	t.Run("an empty allowlist refuses any resource", func(t *testing.T) {
		_, _, mux := newFixture(t)

		q := baseAuthorizeQuery(confidentialID)
		q.Add("resource", resAPI)

		rec := authorize(t, mux, q)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "invalid_target")
		assert.Contains(t, rec.Body.String(), "not registered for this client",
			"an empty allowlist still rejects via the allowlist-membership check, same as an unregistered resource")
	})

	// The regression guard. Every client that exists today sends no resource
	// parameter and must be completely unaffected.
	t.Run("no resource parameter still authorizes", func(t *testing.T) {
		_, st, mux := newFixture(t)

		code := codeFrom(t, authorize(t, mux, baseAuthorizeQuery(confidentialID)))

		stored, err := st.GetAuthCode(context.Background(), code)
		require.NoError(t, err)
		assert.Empty(t, stored.Resources)
	})
}
