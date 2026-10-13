package organization

import "errors"

// ErrSlugTaken is returned when an organization is created with a slug that
// another organization in the same app already holds. The HTTP layer answers
// it with 409 Conflict.
var ErrSlugTaken = errors.New("organization: slug already taken")

// ErrInvitationExpired is returned when an invitation is accepted after its
// ExpiresAt. The HTTP layer answers it with 410 Gone.
var ErrInvitationExpired = errors.New("organization: invitation has expired")
