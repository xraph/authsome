package subscription

import (
	"github.com/xraph/authsome/plugins/subscription/core"
)

// Re-export Config types for convenience.
type (
	Config       = core.Config
	StripeConfig = core.StripeConfig
)

// DefaultConfig returns the default plugin configuration.
var DefaultConfig = core.DefaultConfig
