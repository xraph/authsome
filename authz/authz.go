package authz

import (
	"context"
	"errors"
	"fmt"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/warden"
)

// ErrForbidden is returned when an authorization check fails.
var ErrForbidden = errors.New("authsome: forbidden")

// Checker wraps Warden authorization checks for authsome service methods.
type Checker struct {
	warden      *warden.Engine
	platformApp id.AppID
}

// NewChecker creates a new authorization checker backed by a Warden engine.
func NewChecker(w *warden.Engine, platformApp id.AppID) *Checker {
	return &Checker{warden: w, platformApp: platformApp}
}

// Authorize checks if the current user (from context) can perform the given
// action on the specified resource. Returns nil if allowed, ErrForbidden if denied.
// If no user is present in the context (internal/system call), the check is skipped.
func (c *Checker) Authorize(ctx context.Context, action, resourceType, resourceID string) error {
	userID, ok := middleware.UserIDFrom(ctx)
	if !ok || userID.IsNil() {
		// No user in context — this is an internal/system call (e.g., bootstrap).
		// Allow it; HTTP handlers always set user via middleware.
		return nil
	}

	result, err := c.warden.Check(ctx, &warden.CheckRequest{
		Subject:  warden.Subject{Kind: warden.SubjectUser, ID: userID.String()},
		Action:   warden.Action{Name: action},
		Resource: warden.Resource{Type: resourceType, ID: resourceID},
	})
	if err != nil {
		return fmt.Errorf("authsome: authz check: %w", err)
	}
	if !result.Allowed {
		return ErrForbidden
	}
	return nil
}

// AuthorizeOrSelf allows the operation if the current user is the resource
// owner OR has the required permission. This handles the common pattern of
// "users can manage their own resources".
func (c *Checker) AuthorizeOrSelf(ctx context.Context, ownerID id.UserID, action, resourceType, resourceID string) error {
	userID, ok := middleware.UserIDFrom(ctx)
	if !ok || userID.IsNil() {
		// No user in context — internal/system call. Allow it, consistent
		// with Authorize(). HTTP handlers always set user via middleware.
		return nil
	}
	// Self-access is always allowed.
	if userID == ownerID {
		return nil
	}
	return c.Authorize(ctx, action, resourceType, resourceID)
}
