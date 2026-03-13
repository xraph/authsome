package strategy_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	log "github.com/xraph/go-utils/log"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/strategy"
	"github.com/xraph/authsome/user"
)

// mockStrategy is a test double for the strategy.Strategy interface.
type mockStrategy struct {
	name   string
	result *strategy.Result
	err    error
}

func (m *mockStrategy) Name() string { return m.name }
func (m *mockStrategy) Authenticate(_ context.Context, _ *http.Request) (*strategy.Result, error) {
	return m.result, m.err
}

func TestRegistry_Register_PriorityOrder(t *testing.T) {
	reg := strategy.NewRegistry(log.NewNoopLogger())
	reg.Register(&mockStrategy{name: "low"}, 10)
	reg.Register(&mockStrategy{name: "high"}, 1)
	reg.Register(&mockStrategy{name: "mid"}, 5)

	names := reg.Strategies()
	assert.Equal(t, []string{"high", "mid", "low"}, names)
}

func TestRegistry_Authenticate_FirstApplicable(t *testing.T) {
	reg := strategy.NewRegistry(log.NewNoopLogger())

	// Not applicable
	reg.Register(&mockStrategy{
		name: "skip",
		err:  strategy.ErrStrategyNotApplicable{},
	}, 1)

	// Applicable
	expectedUser := &user.User{FirstName: "Alice"}
	expectedSession := &session.Session{Token: "tok"}
	reg.Register(&mockStrategy{
		name:   "match",
		result: &strategy.Result{User: expectedUser, Session: expectedSession},
	}, 2)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "/", nil)
	require.NoError(t, err)
	result, err := reg.Authenticate(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "Alice", result.User.FirstName)
	assert.Equal(t, "tok", result.Session.Token)
}

func TestRegistry_Authenticate_NoApplicable(t *testing.T) {
	reg := strategy.NewRegistry(log.NewNoopLogger())
	reg.Register(&mockStrategy{
		name: "skip",
		err:  strategy.ErrStrategyNotApplicable{},
	}, 1)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "/", nil)
	require.NoError(t, err)
	_, err = reg.Authenticate(context.Background(), req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no applicable")
}

func TestRegistry_Authenticate_PropagatesError(t *testing.T) {
	reg := strategy.NewRegistry(log.NewNoopLogger())
	reg.Register(&mockStrategy{
		name: "broken",
		err:  errors.New("auth failed"),
	}, 1)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "/", nil)
	require.NoError(t, err)
	_, err = reg.Authenticate(context.Background(), req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "auth failed")
}

func TestRegistry_Get(t *testing.T) {
	reg := strategy.NewRegistry(log.NewNoopLogger())
	s := &mockStrategy{name: "password"}
	reg.Register(s, 1)

	got, ok := reg.Get("password")
	assert.True(t, ok)
	assert.Equal(t, "password", got.Name())

	_, ok = reg.Get("nonexistent")
	assert.False(t, ok)
}

func TestRegistry_Empty(t *testing.T) {
	reg := strategy.NewRegistry(log.NewNoopLogger())
	assert.Empty(t, reg.Strategies())

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "/", nil)
	require.NoError(t, err)
	_, err = reg.Authenticate(context.Background(), req)
	assert.Error(t, err)
}
