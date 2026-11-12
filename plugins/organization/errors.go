package organization

import "errors"

// ErrSlugTaken is returned when an organization is created with a slug that
// another organization in the same app already holds. The HTTP layer answers
// it with 409 Conflict.
var ErrSlugTaken = errors.New("organization: slug already taken")

// ErrInvitationExpired is returned when an invitation is accepted after its
// ExpiresAt. The HTTP layer answers it with 410 Gone.
var ErrInvitationExpired = errors.New("organization: invitation has expired")

// ErrInvitationEmailMismatch is returned by AcceptInvitationAs when the
// signed-in user does not own the email the invitation was sent to. The HTTP
// layer answers it with 403 Forbidden.
var ErrInvitationEmailMismatch = errors.New("organization: invitation was sent to a different email address")

// ErrInvitationNotActionable is returned by ResendInvitation and
// RevokeInvitation when the invitation has already been accepted, declined or
// revoked. The HTTP layer answers it with 409 Conflict.
var ErrInvitationNotActionable = errors.New("organization: invitation can no longer be changed")

// ErrInvitationResendTooSoon is returned by ResendInvitation inside
// ResendInvitationInterval of the last send. The HTTP layer answers it with
// 429 Too Many Requests.
var ErrInvitationResendTooSoon = errors.New("organization: invitation was sent a moment ago")
