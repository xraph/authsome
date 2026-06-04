package social

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/xraph/forge"
	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/user"

	"golang.org/x/oauth2"
)

// resolveUserForCallback resolves (or creates) the AuthSome user behind a
// social identity, using a provider-account-id-first, verified-email-set
// algorithm. It replaces the old "match by a single email" logic that spawned
// duplicate users whenever a provider email changed or an account exposed
// multiple emails.
//
//	STEP 1  Match the stable (provider, provider_user_id) connection — immune
//	        to email changes. On hit: refresh tokens, reconcile emails, done.
//	STEP 2  Else match an existing account by ANY provider-verified email, but
//	        only when the stored address is itself verified (no takeover via an
//	        unverified email). One match links; multiple distinct matches are
//	        ambiguous and refused (409) rather than silently merged.
//	STEP 3  Else create a new user seeded with the provider's known addresses.
func (p *Plugin) resolveUserForCallback(ctx context.Context, appID id.AppID, envID id.EnvironmentID, provider string, pu *ProviderUser, token *oauth2.Token) (*user.User, error) {
	// STEP 1 — provider-account-id match (authoritative).
	if p.oauthStore != nil {
		conn, connErr := p.oauthStore.GetOAuthConnection(ctx, provider, pu.ProviderUserID)
		if connErr == nil {
			u, err := p.store.GetUser(ctx, conn.UserID)
			if err != nil {
				return nil, forge.InternalError(fmt.Errorf("failed to resolve user: %w", err))
			}
			conn.AccessToken = token.AccessToken
			conn.RefreshToken = token.RefreshToken
			conn.ExpiresAt = token.Expiry
			conn.Email = pu.Email
			if updErr := p.oauthStore.UpdateOAuthConnection(ctx, conn); updErr != nil {
				return nil, forge.InternalError(fmt.Errorf("failed to update oauth connection: %w", updErr))
			}
			p.reconcileProviderEmails(ctx, u, appID, envID, provider, pu)
			return u, nil
		}
	}

	// STEP 2 — verified-email match against existing accounts.
	var matched *user.User
	for _, pe := range pu.VerifiedEmails() {
		rec, recErr := p.store.GetUserEmailRecord(ctx, appID, envID, pe.Email)
		if recErr != nil || rec == nil || !rec.Verified {
			// Not found, or the stored address isn't verified -> never link.
			continue
		}
		cand, getErr := p.store.GetUser(ctx, rec.UserID)
		if getErr != nil {
			continue
		}
		switch {
		case matched == nil:
			matched = cand
		case matched.ID.String() != cand.ID.String():
			// Verified provider emails belong to two different accounts.
			// Refuse rather than silently merge.
			return nil, forge.NewHTTPError(http.StatusConflict,
				"this social identity matches multiple accounts; sign in to the intended account and link the provider from settings")
		}
	}
	if matched != nil {
		if err := p.createConnection(ctx, matched, appID, provider, pu, token); err != nil {
			return nil, err
		}
		p.reconcileProviderEmails(ctx, matched, appID, envID, provider, pu)
		return matched, nil
	}

	// STEP 3 — no match: create a new user.
	return p.createUserFromProvider(ctx, appID, envID, provider, pu, token)
}

// createUserFromProvider creates a fresh user seeded with the provider's known
// addresses. Addresses already owned by another account are skipped (never
// stolen), so an unverified provider email colliding with an existing account
// simply isn't attached.
func (p *Plugin) createUserFromProvider(ctx context.Context, appID id.AppID, envID id.EnvironmentID, provider string, pu *ProviderUser, token *oauth2.Token) (*user.User, error) {
	now := time.Now()
	u := &user.User{
		ID:        id.NewUserID(),
		AppID:     appID,
		EnvID:     envID,
		FirstName: pu.FirstName,
		LastName:  pu.LastName,
		Image:     pu.AvatarURL,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Only claim addresses not already owned by some account.
	claimable := make([]ProviderEmail, 0)
	for _, pe := range pu.AllEmails() {
		if _, recErr := p.store.GetUserEmailRecord(ctx, appID, envID, pe.Email); recErr == nil {
			continue // owned by another account
		}
		claimable = append(claimable, pe)
	}

	source := "social:" + provider
	primary := choosePrimaryEmail(pu, claimable)
	if primary == nil {
		// No claimable email — create a user without an email row.
		if err := p.store.CreateUser(ctx, u); err != nil {
			return nil, forge.InternalError(fmt.Errorf("failed to create user: %w", err))
		}
	} else {
		u.Email = primary.Email
		u.EmailVerified = primary.Verified
		if err := p.store.CreateUserWithPrimaryEmail(ctx, u, &user.UserEmail{
			ID:        id.NewUserEmailID(),
			UserID:    u.ID,
			AppID:     appID,
			EnvID:     envID,
			Email:     primary.Email,
			Verified:  primary.Verified,
			IsPrimary: true,
			Source:    source,
		}); err != nil {
			return nil, forge.InternalError(fmt.Errorf("failed to create user: %w", err))
		}
		for _, pe := range claimable {
			if pe.Email == primary.Email {
				continue
			}
			// Best-effort; a lost race surfaces as ErrEmailTaken and is skipped.
			if err := p.store.AddUserEmail(ctx, &user.UserEmail{
				ID:       id.NewUserEmailID(),
				UserID:   u.ID,
				AppID:    appID,
				EnvID:    envID,
				Email:    pe.Email,
				Verified: pe.Verified,
				Source:   source,
			}); err != nil && p.logger != nil {
				p.logger.Debug("social: skip attaching provider email",
					log.String("email", pe.Email),
					log.String("error", err.Error()),
				)
			}
		}
	}

	if p.engine != nil {
		p.engine.EnsureDefaultRole(ctx, appID, u.ID)
	}
	if err := p.createConnection(ctx, u, appID, provider, pu, token); err != nil {
		return nil, err
	}
	return u, nil
}

// createConnection records the (provider, provider_user_id) link. No-op when
// no connection store is wired.
func (p *Plugin) createConnection(ctx context.Context, u *user.User, appID id.AppID, provider string, pu *ProviderUser, token *oauth2.Token) error {
	if p.oauthStore == nil {
		return nil
	}
	now := time.Now()
	conn := &OAuthConnection{
		ID:             id.NewOAuthConnectionID(),
		AppID:          appID,
		UserID:         u.ID,
		Provider:       provider,
		ProviderUserID: pu.ProviderUserID,
		Email:          pu.Email,
		AccessToken:    token.AccessToken,
		RefreshToken:   token.RefreshToken,
		ExpiresAt:      token.Expiry,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := p.oauthStore.CreateOAuthConnection(ctx, conn); err != nil {
		return forge.InternalError(fmt.Errorf("failed to store oauth connection: %w", err))
	}
	return nil
}

// reconcileProviderEmails attaches newly-seen provider-verified addresses to
// the resolved user and upgrades a previously-unverified address the provider
// now verifies. Addresses owned by a different account are left untouched.
func (p *Plugin) reconcileProviderEmails(ctx context.Context, u *user.User, appID id.AppID, envID id.EnvironmentID, provider string, pu *ProviderUser) {
	source := "social:" + provider
	for _, pe := range pu.AllEmails() {
		rec, recErr := p.store.GetUserEmailRecord(ctx, appID, envID, pe.Email)
		if recErr != nil {
			// Not owned by anyone — attach only if the provider verified it.
			if pe.Verified {
				if err := p.store.AddUserEmail(ctx, &user.UserEmail{
					ID:       id.NewUserEmailID(),
					UserID:   u.ID,
					AppID:    appID,
					EnvID:    envID,
					Email:    pe.Email,
					Verified: true,
					Source:   source,
				}); err != nil && p.logger != nil {
					p.logger.Debug("social: skip attaching provider email",
						log.String("email", pe.Email),
						log.String("error", err.Error()),
					)
				}
			}
			continue
		}
		if rec.UserID.String() == u.ID.String() && !rec.Verified && pe.Verified {
			if err := p.store.MarkUserEmailVerified(ctx, u.ID, pe.Email); err != nil && p.logger != nil {
				p.logger.Debug("social: failed to upgrade email verification",
					log.String("email", pe.Email),
					log.String("error", err.Error()),
				)
			}
		}
	}
}

// choosePrimaryEmail picks the primary address for a new user: the provider's
// designated primary if claimable, else any verified claimable, else the first.
func choosePrimaryEmail(pu *ProviderUser, claimable []ProviderEmail) *ProviderEmail {
	if len(claimable) == 0 {
		return nil
	}
	norm := normalizeProviderEmail(pu.Email)
	if norm != "" {
		for i := range claimable {
			if claimable[i].Email == norm {
				return &claimable[i]
			}
		}
	}
	for i := range claimable {
		if claimable[i].Verified {
			return &claimable[i]
		}
	}
	return &claimable[0]
}
