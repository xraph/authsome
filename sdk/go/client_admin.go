// Hand-maintained admin SDK helpers that augment the auto-generated
// client.go. These additions either reshape an awkward generated
// signature or add operator-friendly error wrapping. They live in a
// separate file (without the "DO NOT EDIT" banner) so re-running
// sdkgen never clobbers them.

package authclient

import (
	"context"
	"errors"
	"net/http"
)

// AdminCreateAppWithHint is a thin wrapper over the generated
// AdminCreateApp that augments a 403 response with a pointer at the
// most-likely cause: the user the API key is bound to lacks the
// `manage:user` permission. Operators routinely mint a key as a
// regular signed-up user, then spend hours debugging why
// AdminCreateApp returns "insufficient permissions" — this hint
// points them at the role assignment instead.
//
// Common failure modes (see ClientError.StatusCode):
//   - 401 with "X-App-ID was not sent": manifest discovery failed.
//     Pass WithAppID(platformAppID) explicitly.
//   - 401 otherwise: key invalid, revoked, or minted under a
//     different App than the one we're sending in X-App-ID.
//   - 403: authentication succeeded, but the bound user lacks
//     `manage:user`. Wrapped with a hint pointing at platform_admin.
func (c *Client) AdminCreateAppWithHint(ctx context.Context, req *AdminCreateAppRequest) (*AdminAppResponse, error) {
	resp, err := c.AdminCreateApp(ctx, req)
	if err != nil {
		return nil, wrapAdminCreateAppErr(err)
	}
	return resp, nil
}

// wrapAdminCreateAppErr augments a 403 from POST /v1/admin/apps with
// the most-likely cause. Other errors pass through unchanged.
func wrapAdminCreateAppErr(err error) error {
	var ce *ClientError
	if !errors.As(err, &ce) || ce.StatusCode != http.StatusForbidden {
		return err
	}
	hint := "the user this API key is bound to needs a role granting manage:user (e.g. platform_admin)"
	if ce.Message != "" {
		ce.Message = ce.Message + " [" + hint + "]"
	} else {
		ce.Message = hint
	}
	return ce
}
