package subscription

import (
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/xraph/forge"
	"github.com/xraph/ledger"
	"github.com/xraph/ledger/subscription"

	"github.com/xraph/authsome/middleware"
)

var errNoPortalURL = errors.New("provider returned no URL")

// writeErr turns a ledger refusal into an error the caller can act on. Only
// failures the caller didn't cause become a 500.
func writeErr(ctx forge.Context, err error, action string) error {
	switch {
	case errors.Is(err, ledger.ErrSubscriptionCanceled), errors.Is(err, ledger.ErrSubscriptionExpired):
		return forge.NewHTTPError(http.StatusConflict, "subscription has ended")
	case errors.Is(err, ledger.ErrSubscriptionExists):
		return forge.NewHTTPError(http.StatusConflict, "already subscribed")
	case errors.Is(err, ErrNoScheduledCancel):
		return forge.NewHTTPError(http.StatusConflict, "no cancellation is scheduled")
	case errors.Is(err, ErrCancelDue):
		return forge.NewHTTPError(http.StatusConflict, "the cancellation is already due")
	case errors.Is(err, ledger.ErrPlanArchived):
		return forge.BadRequest("plan is not available")
	case errors.Is(err, ledger.ErrPlanNotFound):
		return forge.NotFound("plan not found")
	case errors.Is(err, ledger.ErrSubscriptionNotFound):
		return forge.NotFound("subscription not found")
	case errors.Is(err, ledger.ErrInvalidInput), errors.Is(err, ledger.ErrInvalidQuantity):
		// Ledger's detail names the plan or feature at fault, which is the
		// caller's own input, so it is safe and useful to pass on.
		return forge.BadRequest(strings.TrimPrefix(err.Error(), "ledger: "))
	}
	return middleware.InternalError(ctx, fmt.Errorf("failed to %s: %w", action, err))
}

// requireStatus refuses with a 409 when sub isn't in one of the given states,
// so a transition ledger would reject reads as a conflict, not bad input.
func requireStatus(sub *subscription.Subscription, allowed ...subscription.Status) error {
	if slices.Contains(allowed, sub.Status) {
		return nil
	}
	return forge.NewHTTPError(http.StatusConflict, fmt.Sprintf("subscription is %s", sub.Status))
}
