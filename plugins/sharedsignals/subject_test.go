package sharedsignals

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/sharedsignals/caep"
	"github.com/xraph/authsome/store/memory"
	"github.com/xraph/authsome/user"
)

type resolveFixture struct {
	plugin *Plugin
	stream *InboundStream
	user   *user.User
	appID  id.AppID
	envID  id.EnvironmentID
}

func newResolveFixture(t *testing.T, verifiedEmail bool) resolveFixture {
	t.Helper()
	ctx := context.Background()
	appID, envID := id.NewAppID(), id.NewEnvironmentID()

	authStore := memory.New()
	u := &user.User{
		ID: id.NewUserID(), AppID: appID, EnvID: envID,
		Email: "target@corp.com", EmailVerified: verifiedEmail,
		Phone: "+15551234567", PhoneVerified: verifiedEmail,
	}
	require.NoError(t, authStore.CreateUserWithPrimaryEmail(ctx, u, &user.UserEmail{
		ID: id.NewUserEmailID(), UserID: u.ID, AppID: appID, EnvID: envID,
		Email: u.Email, Verified: verifiedEmail, IsPrimary: true,
	}))

	p := New()
	p.store = NewMemoryStore()
	p.authStore = authStore

	stream := &InboundStream{
		ID: id.NewSSFStreamID(), AppID: appID, EnvID: envID,
		Issuer: "https://org.okta.com",
		AllowedSubjectFormats: []string{
			caep.FormatIssSub, caep.FormatOpaque, caep.FormatEmail,
			caep.FormatPhoneNumber, caep.FormatAliases,
		},
		VerifiedDomains: []string{"corp.com"},
		Status:          StatusEnabled,
	}
	require.NoError(t, p.store.CreateInboundStream(ctx, stream))

	require.NoError(t, p.store.UpsertSubjectLink(ctx, &SubjectLink{
		ID: id.NewSSFLinkID(), AppID: appID, EnvID: envID,
		Issuer: "https://org.okta.com", Subject: "okta-user-1",
		UserID: u.ID, Source: SourceSSO,
	}))

	return resolveFixture{plugin: p, stream: stream, user: u, appID: appID, envID: envID}
}

func TestResolveSubject_IssSub(t *testing.T) {
	f := newResolveFixture(t, true)
	got, err := f.plugin.resolveSubject(context.Background(), f.stream, caep.SubjectID{
		Format: caep.FormatIssSub, Issuer: "https://org.okta.com", Subject: "okta-user-1",
	})
	require.NoError(t, err)
	assert.Equal(t, OutcomeApplied, got.Outcome)
	assert.Equal(t, f.user.ID, got.UserID)
}

// An IdP may only speak for its own subjects. A SET from Okta claiming an
// Entra subject is a lateral-movement attempt.
func TestResolveSubject_IssSubRejectsForeignIssuer(t *testing.T) {
	f := newResolveFixture(t, true)
	got, err := f.plugin.resolveSubject(context.Background(), f.stream, caep.SubjectID{
		Format: caep.FormatIssSub, Issuer: "https://login.microsoftonline.com", Subject: "okta-user-1",
	})
	require.NoError(t, err)
	assert.Equal(t, OutcomeRejected, got.Outcome)
	assert.True(t, got.UserID.IsNil())
}

func TestResolveSubject_UnknownSubjectIsUnresolved(t *testing.T) {
	f := newResolveFixture(t, true)
	got, err := f.plugin.resolveSubject(context.Background(), f.stream, caep.SubjectID{
		Format: caep.FormatIssSub, Issuer: "https://org.okta.com", Subject: "nobody",
	})
	require.NoError(t, err)
	assert.Equal(t, OutcomeUnresolved, got.Outcome)
}

func TestResolveSubject_EmailInVerifiedDomain(t *testing.T) {
	f := newResolveFixture(t, true)
	got, err := f.plugin.resolveSubject(context.Background(), f.stream, caep.SubjectID{
		Format: caep.FormatEmail, Email: "target@corp.com",
	})
	require.NoError(t, err)
	assert.Equal(t, OutcomeApplied, got.Outcome)
	assert.Equal(t, f.user.ID, got.UserID)
}

func TestResolveSubject_EmailOutsideVerifiedDomainRejected(t *testing.T) {
	f := newResolveFixture(t, true)
	got, err := f.plugin.resolveSubject(context.Background(), f.stream, caep.SubjectID{
		Format: caep.FormatEmail, Email: "target@notours.com",
	})
	require.NoError(t, err)
	assert.Equal(t, OutcomeRejected, got.Outcome)
}

// The subtle one. If we matched an unverified address, anyone could attach
// ceo@corp.com to their own account and quietly absorb the CEO's revocation
// events, leaving the real account signed in.
func TestResolveSubject_EmailMustBeVerified(t *testing.T) {
	f := newResolveFixture(t, false)
	got, err := f.plugin.resolveSubject(context.Background(), f.stream, caep.SubjectID{
		Format: caep.FormatEmail, Email: "target@corp.com",
	})
	require.NoError(t, err)
	assert.Equal(t, OutcomeRejected, got.Outcome,
		"an unverified address must never resolve a subject")
}

func TestResolveSubject_EmailFormatNotAllowedOnStream(t *testing.T) {
	f := newResolveFixture(t, true)
	f.stream.AllowedSubjectFormats = []string{caep.FormatIssSub}
	got, err := f.plugin.resolveSubject(context.Background(), f.stream, caep.SubjectID{
		Format: caep.FormatEmail, Email: "target@corp.com",
	})
	require.NoError(t, err)
	assert.Equal(t, OutcomeRejected, got.Outcome)
}

func TestResolveSubject_UnsupportedFormatsRejected(t *testing.T) {
	f := newResolveFixture(t, true)
	for _, subj := range []caep.SubjectID{
		{Format: caep.FormatAccount, URI: "acct:target@corp.com"},
		{Format: caep.FormatURI, URI: "https://corp.com/u/1"},
		{Format: caep.FormatDID, URL: "did:example:1"},
		{Format: "something-invented"},
	} {
		got, err := f.plugin.resolveSubject(context.Background(), f.stream, subj)
		require.NoError(t, err)
		assert.Equal(t, OutcomeRejected, got.Outcome, "format %q", subj.Format)
	}
}

func TestResolveSubject_ComplexSubjectUsesUserMember(t *testing.T) {
	f := newResolveFixture(t, true)
	got, err := f.plugin.resolveSubject(context.Background(), f.stream, caep.SubjectID{
		Members: map[string]caep.SubjectID{
			"user": {Format: caep.FormatIssSub, Issuer: "https://org.okta.com", Subject: "okta-user-1"},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, OutcomeApplied, got.Outcome)
	assert.Equal(t, f.user.ID, got.UserID)
}

func TestResolveSubject_ComplexSubjectWithoutUserMemberIsRejected(t *testing.T) {
	f := newResolveFixture(t, true)
	got, err := f.plugin.resolveSubject(context.Background(), f.stream, caep.SubjectID{
		Members: map[string]caep.SubjectID{
			"tenant": {Format: caep.FormatOpaque, ID: "t1"},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, OutcomeRejected, got.Outcome)
}

func TestResolveSubject_AliasesAgreeing(t *testing.T) {
	f := newResolveFixture(t, true)
	got, err := f.plugin.resolveSubject(context.Background(), f.stream, caep.SubjectID{
		Format: caep.FormatAliases,
		Identifiers: []caep.SubjectID{
			{Format: caep.FormatIssSub, Issuer: "https://org.okta.com", Subject: "okta-user-1"},
			{Format: caep.FormatEmail, Email: "target@corp.com"},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, OutcomeApplied, got.Outcome)
	assert.Equal(t, f.user.ID, got.UserID)
}

// Two aliases naming two different users is a contradiction, and guessing
// which one was meant is how the wrong person gets signed out.
func TestResolveSubject_AliasesDisagreeingRejected(t *testing.T) {
	ctx := context.Background()
	f := newResolveFixture(t, true)

	other := &user.User{
		ID: id.NewUserID(), AppID: f.appID, EnvID: f.envID,
		Email: "other@corp.com", EmailVerified: true,
	}
	authStore, ok := f.plugin.authStore.(*memory.Store)
	require.True(t, ok)
	require.NoError(t, authStore.CreateUserWithPrimaryEmail(ctx, other, &user.UserEmail{
		ID: id.NewUserEmailID(), UserID: other.ID, AppID: f.appID, EnvID: f.envID,
		Email: other.Email, Verified: true, IsPrimary: true,
	}))

	got, err := f.plugin.resolveSubject(ctx, f.stream, caep.SubjectID{
		Format: caep.FormatAliases,
		Identifiers: []caep.SubjectID{
			{Format: caep.FormatIssSub, Issuer: "https://org.okta.com", Subject: "okta-user-1"},
			{Format: caep.FormatEmail, Email: "other@corp.com"},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, OutcomeRejected, got.Outcome)
}
