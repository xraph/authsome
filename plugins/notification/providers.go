package notification

// This file re-exports provider types and constructors from the providers subpackage
// for convenience and backward compatibility

import (
	"github.com/xraph/authsome/plugins/notification/providers"
)

// Re-export provider config types
type (
	ResendConfig       = providers.ResendConfig
	MailerSendConfig   = providers.MailerSendConfig
	PostmarkConfig     = providers.PostmarkConfig
)

// Re-export provider constructors
var (
	NewResendProvider     = providers.NewResendProvider
	NewMailerSendProvider = providers.NewMailerSendProvider
	NewPostmarkProvider   = providers.NewPostmarkProvider
)

