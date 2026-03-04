package password

import (
	"context"
	"net/http"

	"github.com/xraph/authsome/strategy"
)

// passwordStrategy is the authentication strategy for email/password login.
// It delegates the actual authentication to the engine's SignIn method.
//
// In the current architecture, the core engine already handles email/password
// authentication in its SignIn method. This strategy is registered to indicate
// that password-based auth is available, and to participate in the strategy
// priority chain. Future versions may move the full auth logic here.
type passwordStrategy struct{}

var _ strategy.Strategy = (*passwordStrategy)(nil)

// Name returns the strategy name.
func (s *passwordStrategy) Name() string { return "password" }

// Authenticate performs password-based authentication.
// Currently returns ErrStrategyNotApplicable to let the engine's built-in
// SignIn flow handle the actual credential verification. The strategy
// primarily serves as a registration marker for feature discovery.
func (s *passwordStrategy) Authenticate(_ context.Context, _ *http.Request) (*strategy.Result, error) {
	return nil, strategy.ErrStrategyNotApplicable{}
}
