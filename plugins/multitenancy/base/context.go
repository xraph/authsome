package base

import (
	"context"

	"github.com/rs/xid"
)

// Context keys for multi-tenancy
type contextKey string

const (
	OrganizationContextKey contextKey = "organization_id"
)