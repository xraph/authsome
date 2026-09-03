package retention

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugin"
)

// timeNow returns a moment safely past "now" so ClaimDue's due-time check
// treats everything a hook just enqueued as due, without a sleep.
func timeNow() time.Time {
	return time.Now().Add(time.Minute)
}

// failingStore fails every write, to prove a hook still returns nil.
type failingStore struct{ Store }

func (failingStore) Enqueue(context.Context, *Job) error { return errors.New("db down") }

func newHookPlugin(s Store) *Plugin {
	p := New(Config{Providers: []ProviderConfig{{Name: "fake", Type: "generic"}}})
	p.store = s
	// enqueueFor logs on a store error, and these tests exercise that path,
	// so the logger must be set here rather than left to OnInit.
	p.logger = log.NewNoopLogger()
	p.providers = map[string]Provider{"fake": &fakeProvider{caps: CapContacts | CapActivities}}
	return p
}

func TestAfterSignUpEnqueuesContactAndActivity(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	p := newHookPlugin(s)
	appID, userID := id.NewAppID(), id.NewUserID()

	require.NoError(t, p.afterSignUpFor(ctx, appID, id.EnvironmentID{}, userID))

	jobs, err := s.ClaimDue(ctx, 10, 0, timeNow())
	require.NoError(t, err)
	require.Len(t, jobs, 2)

	kinds := map[string]bool{jobs[0].Kind: true, jobs[1].Kind: true}
	assert.True(t, kinds[KindContactUpsert])
	assert.True(t, kinds[KindActivityLog])
}

func TestAfterSignInEnqueuesActivityOnly(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	p := newHookPlugin(s)
	appID, userID := id.NewAppID(), id.NewUserID()

	require.NoError(t, p.afterSignInFor(ctx, appID, id.EnvironmentID{}, userID))

	jobs, err := s.ClaimDue(ctx, 10, 0, timeNow())
	require.NoError(t, err)
	require.Len(t, jobs, 1, "sign-in must not read the ref table to decide on an upsert")
	assert.Equal(t, KindActivityLog, jobs[0].Kind)
	assert.Equal(t, "logged_in", jobs[0].Payload["activity_type"])
}

func TestHookSwallowsStoreErrors(t *testing.T) {
	ctx := context.Background()
	p := newHookPlugin(failingStore{Store: NewMemoryStore()})
	err := p.afterSignInFor(ctx, id.NewAppID(), id.EnvironmentID{}, id.NewUserID())
	assert.NoError(t, err, "a retention failure must never fail a login")
}

func TestHookNoOpWithoutProviders(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	p := New()
	p.store = s
	p.providers = map[string]Provider{}

	require.NoError(t, p.afterSignInFor(ctx, id.NewAppID(), id.EnvironmentID{}, id.NewUserID()))
	jobs, err := s.ClaimDue(ctx, 10, 0, timeNow())
	require.NoError(t, err)
	assert.Empty(t, jobs, "no configured provider means no queued work")
}

func TestAfterUserUpdateEnqueuesContactUpsert(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	p := newHookPlugin(s)
	appID, userID := id.NewAppID(), id.NewUserID()

	require.NoError(t, p.afterUserUpdateFor(ctx, appID, id.EnvironmentID{}, userID))

	jobs, err := s.ClaimDue(ctx, 10, 0, timeNow())
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	assert.Equal(t, KindContactUpsert, jobs[0].Kind)
	assert.Equal(t, "profile_updated", jobs[0].Payload["activity_type"])
}

func TestIdempotencyKeyIsStableAndDistinct(t *testing.T) {
	a := idempotencyKey("hubspot", "ausr_1", "logged_in", "2026-09-03T10:00:00Z")
	b := idempotencyKey("hubspot", "ausr_1", "logged_in", "2026-09-03T10:00:00Z")
	c := idempotencyKey("hubspot", "ausr_1", "logged_in", "2026-09-03T10:00:01Z")
	assert.Equal(t, a, b)
	assert.NotEqual(t, a, c)
}

func TestPluginImplementsHookInterfaces(t *testing.T) {
	p := New()
	assert.Implements(t, (*plugin.AfterSignUp)(nil), p)
	assert.Implements(t, (*plugin.AfterSignIn)(nil), p)
	assert.Implements(t, (*plugin.AfterUserUpdate)(nil), p)
}
