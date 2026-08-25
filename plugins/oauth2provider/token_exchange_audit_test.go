package oauth2provider_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/xraph/authsome"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/oauth2provider"
	"github.com/xraph/authsome/principal"
	"github.com/xraph/authsome/session"
)

// Every exchange attempt writes exactly one security event. The failures
// matter more than the successes: scope_escalation and cross_app are attack
// signatures rather than user error, and the closed denial-reason vocabulary
// is what makes them alertable instead of greppable.

func TestRFC8693_SuccessWritesSecurityEvent(t *testing.T) {
	f := newExchangeFixture(t)
	f.grantSucceeds([]string{"a"})
	subject := f.seedSubject(t, f.appID, []string{"a", "b"}, "")

	decodeTokenResponseMap(t, f.exchange(t, map[string]string{
		"subject_token":      subject,
		"subject_token_type": accessTokenType,
		"scope":              "a",
	}))

	require.Len(t, f.events.events, 1)
	ev := f.events.events[0]
	assert.Equal(t, "oauth2.token_exchange", ev.Action)
	assert.Equal(t, "success", ev.Outcome)
	// AppID must be populated. The hook-bus bridge never sets it and
	// securityevent.Query filters on it, which is why this writes to the
	// store directly rather than emitting a hook.
	assert.Equal(t, f.appID, ev.AppID)
	assert.Equal(t, "a", ev.Metadata["granted_scopes"])
	assert.Equal(t, xchgClientID, ev.Metadata["client_id"])
	assert.NotEmpty(t, ev.Metadata["issued_session_id"])
	assert.NotEmpty(t, ev.Metadata["subject_session_id"])
	assert.NotEmpty(t, ev.Metadata["actor_principal_id"])
	assert.NotContains(t, ev.Metadata, "denial_reason")
}

func TestRFC8693_DenialReasons(t *testing.T) {
	tests := []struct {
		name   string
		reason string
		drive  func(t *testing.T, f *xchgFixture)
	}{
		{
			name:   "scope the subject does not hold",
			reason: "scope_escalation",
			drive: func(t *testing.T, f *xchgFixture) {
				subject := f.seedSubject(t, f.appID, []string{"a"}, "")
				f.exchange(t, map[string]string{
					"subject_token": subject, "subject_token_type": accessTokenType, "scope": "b",
				})
			},
		},
		{
			name:   "subject from another app",
			reason: "cross_app",
			drive: func(t *testing.T, f *xchgFixture) {
				subject := f.seedSubject(t, id.NewAppID(), []string{"a"}, "")
				f.exchange(t, map[string]string{
					"subject_token": subject, "subject_token_type": accessTokenType, "scope": "a",
				})
			},
		},
		{
			name:   "no live delegation grant",
			reason: "no_grant",
			drive: func(t *testing.T, f *xchgFixture) {
				f.engine.exchangeErr = authsome.ErrExchangeRefused
				subject := f.seedSubject(t, f.appID, []string{"a"}, "")
				f.exchange(t, map[string]string{
					"subject_token": subject, "subject_token_type": accessTokenType, "scope": "a",
				})
			},
		},
		{
			name:   "unsupported subject token type",
			reason: "unsupported_token_type",
			drive: func(t *testing.T, f *xchgFixture) {
				subject := f.seedSubject(t, f.appID, []string{"a"}, "")
				f.exchange(t, map[string]string{
					"subject_token":      subject,
					"subject_token_type": "urn:ietf:params:oauth:token-type:refresh_token",
					"scope":              "a",
				})
			},
		},
		{
			name:   "DPoP-bound subject would be unbound",
			reason: "binding_downgrade",
			drive: func(t *testing.T, f *xchgFixture) {
				subject := f.seedSubject(t, f.appID, []string{"a"}, "some-jkt")
				f.exchange(t, map[string]string{
					"subject_token": subject, "subject_token_type": accessTokenType, "scope": "a",
				})
			},
		},
		{
			name:   "client has no linked principal",
			reason: "client_has_no_principal",
			drive: func(t *testing.T, f *xchgFixture) {
				hashed, err := bcrypt.GenerateFromPassword([]byte("s"), bcrypt.MinCost)
				require.NoError(t, err)
				require.NoError(t, f.oauth.CreateClient(context.Background(), &oauth2provider.OAuth2Client{
					ID: id.NewOAuth2ClientID(), AppID: f.appID, ClientID: "unlinked-audit",
					ClientSecret: string(hashed), Name: "Unlinked", Scopes: []string{"a"},
					GrantTypes: []string{exchangeGrant},
				}))
				subject := f.seedSubject(t, f.appID, []string{"a"}, "")
				postToken(t, f.mux, map[string]string{
					"grant_type": exchangeGrant, "client_id": "unlinked-audit", "client_secret": "s",
					"subject_token": subject, "subject_token_type": accessTokenType, "scope": "a",
				})
			},
		},
		{
			name:   "client principal cannot be resolved",
			reason: "principal_not_found",
			drive: func(t *testing.T, f *xchgFixture) {
				f.engine.principal = nil // ResolvePrincipal now returns ErrNotFound
				subject := f.seedSubject(t, f.appID, []string{"a"}, "")
				f.exchange(t, map[string]string{
					"subject_token": subject, "subject_token_type": accessTokenType, "scope": "a",
				})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newExchangeFixture(t)
			tt.drive(t, f)

			require.Len(t, f.events.events, 1, "exactly one event per attempt")
			ev := f.events.events[0]
			assert.Equal(t, "oauth2.token_exchange", ev.Action)
			assert.Equal(t, "failure", ev.Outcome)
			assert.Equal(t, tt.reason, ev.Metadata["denial_reason"])
			assert.Equal(t, f.appID, ev.AppID, "AppID must be set or the event is unqueryable")
			assert.NotContains(t, ev.Metadata, "issued_session_id")
		})
	}
}

// A subject that was itself minted by an exchange records its chain depth, so
// an auditor can see how far a credential has travelled.
func TestRFC8693_RecordsChainDepth(t *testing.T) {
	f := newExchangeFixture(t)
	f.grantSucceeds([]string{"a"})

	tok := "subject-depth"
	require.NoError(t, f.core.CreateSession(context.Background(), &session.Session{
		ID:         id.NewSessionID(),
		AppID:      f.appID,
		UserID:     id.NewUserID(),
		Token:      tok,
		Scopes:     []string{"a"},
		Actors:     principal.Chain{{Kind: principal.KindAgent, ID: "svc_prior"}},
		ActorGrant: principal.GrantDelegation,
		ExpiresAt:  time.Now().Add(time.Hour),
	}))

	f.exchange(t, map[string]string{
		"subject_token": tok, "subject_token_type": accessTokenType, "scope": "a",
	})

	require.Len(t, f.events.events, 1)
	assert.Equal(t, "2", f.events.events[0].Metadata["chain_depth"])
}
