package dpop

import "strings"

// Mode is how strictly DPoP applies at an issuance point.
type Mode string

const (
	// ModeOff ignores proofs entirely. Tokens are issued unbound.
	ModeOff Mode = "off"
	// ModeOptional binds when the client presents a valid proof and issues an
	// ordinary bearer token when it does not. This is the migration setting.
	ModeOptional Mode = "optional"
	// ModeRequired refuses to issue an unbound token.
	ModeRequired Mode = "required"
)

// ParseMode converts a stored string into a Mode. Anything unrecognised,
// including the empty string, becomes ModeOff, so a typo or a corrupt row
// fails towards today's behaviour rather than towards locking clients out.
func ParseMode(s string) Mode {
	switch Mode(strings.ToLower(strings.TrimSpace(s))) {
	case ModeOptional:
		return ModeOptional
	case ModeRequired:
		return ModeRequired
	default:
		return ModeOff
	}
}

func (m Mode) rank() int {
	switch m {
	case ModeRequired:
		return 2
	case ModeOptional:
		return 1
	default:
		return 0
	}
}

// MaxMode returns the stricter of two modes.
//
// Resolution is monotonic on purpose. Setting an app to required states a
// mandate for every client under it, and a per-client field able to quietly
// undo that would turn the strongest setting in the system into a suggestion.
// A legacy client that cannot cope is handled by moving the app back to
// optional, which is explicit and audited.
func MaxMode(a, b Mode) Mode {
	if b.rank() > a.rank() {
		return b
	}
	return a
}
