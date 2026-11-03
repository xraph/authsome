package account_test

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/account"
)

// roundTrip answers every request without touching the network.
type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func answer(status int, body string) roundTrip {
	return func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	}
}

func checker(rt http.RoundTripper) *account.BreachChecker {
	return account.NewBreachCheckerWithClient(&http.Client{Transport: rt})
}

func TestBreachCheckerFindsABreachedPassword(t *testing.T) {
	sum := sha1.Sum([]byte("hunter2"))
	hash := strings.ToUpper(hex.EncodeToString(sum[:]))
	c := checker(answer(http.StatusOK, "0000000000000000000000000000000000A:1\r\n"+hash[5:]+":42\r\n"))

	breached, err := c.IsBreachedContext(context.Background(), "hunter2")
	require.NoError(t, err)
	assert.True(t, breached)
}

// A lookup that can't run reports false either way, but only the context
// variant says it didn't run. IsBreached keeps its old contract.
func TestBreachCheckerSaysWhenItCouldNotRun(t *testing.T) {
	for name, rt := range map[string]roundTrip{
		"transport error": func(*http.Request) (*http.Response, error) { return nil, errors.New("dial tcp: refused") },
		"non-200":         answer(http.StatusServiceUnavailable, ""),
	} {
		t.Run(name, func(t *testing.T) {
			breached, err := checker(rt).IsBreachedContext(context.Background(), "hunter2")
			assert.False(t, breached)
			require.ErrorIs(t, err, account.ErrBreachCheckUnavailable)
			assert.NotContains(t, err.Error(), "refused", "the transport's error, which quotes the URL, isn't wrapped")

			breached, err = checker(rt).IsBreached("hunter2")
			assert.False(t, breached)
			assert.NoError(t, err, "IsBreached still fails open with a nil error")
		})
	}
}
