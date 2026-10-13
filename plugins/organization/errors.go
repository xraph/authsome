package organization

import "errors"

// ErrSlugTaken is returned when an organization is created with a slug that
// another organization in the same app already holds. The HTTP layer answers
// it with 409 Conflict.
var ErrSlugTaken = errors.New("organization: slug already taken")
