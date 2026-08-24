package sharedsignals

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
)

func TestLinkSubject_CreatesAndResolves(t *testing.T) {
	ctx := context.Background()
	p := New()
	p.store = NewMemoryStore()
	appID, envID := id.NewAppID(), id.NewEnvironmentID()
	userID := id.NewUserID()

	require.NoError(t, p.LinkSubject(ctx, appID, envID,
		"https://org.okta.com", "okta-user-1", userID, SourceSSO))

	got, err := p.store.GetSubjectLink(ctx, appID, envID, "https://org.okta.com", "okta-user-1")
	require.NoError(t, err)
	assert.Equal(t, userID, got.UserID)
	assert.Equal(t, SourceSSO, got.Source)
}

// Signing in twice must refresh the link rather than pile up rows.
func TestLinkSubject_IsIdempotent(t *testing.T) {
	ctx := context.Background()
	p := New()
	p.store = NewMemoryStore()
	appID, envID := id.NewAppID(), id.NewEnvironmentID()
	userID := id.NewUserID()

	for i := 0; i < 3; i++ {
		require.NoError(t, p.LinkSubject(ctx, appID, envID,
			"https://i", "u1", userID, SourceSSO))
	}
	got, err := p.store.GetSubjectLink(ctx, appID, envID, "https://i", "u1")
	require.NoError(t, err)
	assert.Equal(t, userID, got.UserID)
}

func TestLinkSubject_RejectsEmptyArguments(t *testing.T) {
	ctx := context.Background()
	p := New()
	p.store = NewMemoryStore()
	appID, envID := id.NewAppID(), id.NewEnvironmentID()

	require.Error(t, p.LinkSubject(ctx, appID, envID, "", "u1", id.NewUserID(), SourceSSO))
	require.Error(t, p.LinkSubject(ctx, appID, envID, "https://i", "", id.NewUserID(), SourceSSO))
	require.Error(t, p.LinkSubject(ctx, appID, envID, "https://i", "u1", id.Nil, SourceSSO))
}

// The interface is what sso asserts against, so it has to be satisfied by the
// concrete plugin or the wiring silently does nothing.
func TestPlugin_IsSubjectLinker(t *testing.T) {
	var _ SubjectLinker = New()
}
