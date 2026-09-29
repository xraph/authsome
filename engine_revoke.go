package authsome

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/xraph/authsome/hook"
	"github.com/xraph/authsome/id"
)

// RevokeUserAccess ends every live credential a user holds: all of their
// sessions and every API key bound to them. Bans and deactivations call it
// so that removing a person's access is one operation, not a session sweep
// that forgets the keys.
func (e *Engine) RevokeUserAccess(ctx context.Context, userID id.UserID) error {
	u, err := e.store.GetUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("authsome: revoke user access: %w", err)
	}
	if err := e.store.DeleteUserSessions(ctx, userID); err != nil {
		return fmt.Errorf("authsome: revoke user access: sessions: %w", err)
	}

	keys := e.APIKeyStore()
	list, err := keys.ListAPIKeysByUser(ctx, u.AppID, userID)
	if err != nil {
		return fmt.Errorf("authsome: revoke user access: list keys: %w", err)
	}
	now := time.Now()
	revoked := 0
	for _, k := range list {
		if k.Revoked {
			continue
		}
		k.Revoked = true
		k.UpdatedAt = now
		if err := keys.UpdateAPIKey(ctx, k); err != nil {
			return fmt.Errorf("authsome: revoke user access: key %s: %w", k.ID, err)
		}
		revoked++
	}

	e.hooks.Emit(ctx, &hook.Event{
		Action:     hook.ActionSessionRevoke,
		Resource:   hook.ResourceUser,
		ResourceID: userID.String(),
		ActorID:    userID.String(),
		Tenant:     u.AppID.String(),
		Category:   "access",
		Severity:   hook.SeverityWarning,
		Reason:     "access revoked",
		Metadata: map[string]string{
			"scope":        "all",
			"keys_revoked": strconv.Itoa(revoked),
		},
	})
	return nil
}

// RevokeOtherUserSessions ends every session of the user except keep. A
// credential change (password, second factor, passkey) calls it so a session
// an attacker already holds does not survive the change that was meant to
// lock them out. A nil keep revokes every session.
func (e *Engine) RevokeOtherUserSessions(ctx context.Context, userID id.UserID, keep id.SessionID) error {
	sessions, err := e.store.ListUserSessions(ctx, userID)
	if err != nil {
		return fmt.Errorf("authsome: revoke other sessions: %w", err)
	}
	var appID id.AppID
	revoked := 0
	for _, s := range sessions {
		if s.ID == keep {
			continue
		}
		if err := e.store.DeleteSession(ctx, s.ID); err != nil {
			return fmt.Errorf("authsome: revoke other sessions: %s: %w", s.ID, err)
		}
		e.plugins.EmitAfterSessionRevoke(ctx, s.ID)
		appID = s.AppID
		revoked++
	}
	if revoked == 0 {
		return nil
	}

	e.hooks.Emit(ctx, &hook.Event{
		Action:     hook.ActionSessionRevoke,
		Resource:   hook.ResourceUser,
		ResourceID: userID.String(),
		ActorID:    userID.String(),
		Tenant:     appID.String(),
		Category:   "access",
		Reason:     "credential change",
		Metadata: map[string]string{
			"scope":   "others",
			"revoked": strconv.Itoa(revoked),
			"kept":    keep.String(),
		},
	})
	return nil
}
