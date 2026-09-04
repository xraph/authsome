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
	p.providers = newProviderRegistry(map[string]Provider{"fake": &fakeProvider{caps: CapContacts | CapActivities}})
	return p
}

func TestAfterSignUpEnqueuesContactAndActivity(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	p := newHookPlugin(s)
	appID, userID := id.NewAppID(), id.NewUserID()

	require.NoError(t, p.afterSignUpFor(ctx, appID, id.EnvironmentID{}, userID, "ases_1"))

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

	require.NoError(t, p.afterSignInFor(ctx, appID, id.EnvironmentID{}, userID, "ases_1"))

	jobs, err := s.ClaimDue(ctx, 10, 0, timeNow())
	require.NoError(t, err)
	require.Len(t, jobs, 1, "sign-in must not read the ref table to decide on an upsert")
	assert.Equal(t, KindActivityLog, jobs[0].Kind)
	assert.Equal(t, "logged_in", jobs[0].Payload["activity_type"])
}

func TestHookSwallowsStoreErrors(t *testing.T) {
	ctx := context.Background()
	p := newHookPlugin(failingStore{Store: NewMemoryStore()})
	err := p.afterSignInFor(ctx, id.NewAppID(), id.EnvironmentID{}, id.NewUserID(), "ases_1")
	assert.NoError(t, err, "a retention failure must never fail a login")
}

func TestHookNoOpWithoutProviders(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	p := New()
	p.store = s
	p.providers = newProviderRegistry(nil)

	require.NoError(t, p.afterSignInFor(ctx, id.NewAppID(), id.EnvironmentID{}, id.NewUserID(), "ases_1"))
	jobs, err := s.ClaimDue(ctx, 10, 0, timeNow())
	require.NoError(t, err)
	assert.Empty(t, jobs, "no configured provider means no queued work")
}

func TestAfterUserUpdateEnqueuesContactUpsert(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	p := newHookPlugin(s)
	appID, userID := id.NewAppID(), id.NewUserID()

	require.NoError(t, p.afterUserUpdateFor(ctx, appID, id.EnvironmentID{}, userID, "2026-09-03T10:00:00Z"))

	jobs, err := s.ClaimDue(ctx, 10, 0, timeNow())
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	assert.Equal(t, KindContactUpsert, jobs[0].Kind)
	assert.Equal(t, "profile_updated", jobs[0].Payload["activity_type"])
}

// hooks.go passes user.EnvID through untouched, and every other test in
// this file uses the zero environment. In a multi-environment deployment
// it is never zero, and an environment lost between the hook and the
// outbox row is a contact written against the wrong environment's ref.
func TestEnqueueCarriesTheEnvironmentID(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	p := newHookPlugin(s)
	appID, userID := id.NewAppID(), id.NewUserID()
	envID := id.NewEnvironmentID()

	require.NoError(t, p.afterSignInFor(ctx, appID, envID, userID, "ases_env"))

	jobs, err := s.ClaimDue(ctx, 10, 0, timeNow())
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	assert.Equal(t, envID.String(), jobs[0].EnvID.String())
}

func TestIdempotencyKeyIsStableAndDistinct(t *testing.T) {
	a := idempotencyKey("hubspot", "ausr_1", "logged_in", "ases_1")
	b := idempotencyKey("hubspot", "ausr_1", "logged_in", "ases_1")
	c := idempotencyKey("hubspot", "ausr_1", "logged_in", "ases_2")
	assert.Equal(t, a, b)
	assert.NotEqual(t, a, c)
}

// This is the case the pure-function test above cannot reach: it goes through
// the real enqueue path twice for one event, which is where a clock read would
// have leaked into the key.
func TestEnqueueForSameEventTwiceEnqueuesOnce(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	p := newHookPlugin(s)
	appID, userID := id.NewAppID(), id.NewUserID()

	p.enqueueFor(ctx, appID, id.EnvironmentID{}, userID, KindActivityLog, "logged_in", "ases_dupe")
	p.enqueueFor(ctx, appID, id.EnvironmentID{}, userID, KindActivityLog, "logged_in", "ases_dupe")

	jobs, err := s.ClaimDue(ctx, 10, 0, timeNow())
	require.NoError(t, err)
	assert.Len(t, jobs, 1, "the same logical event must enqueue once, not once per dispatch")
}

func TestEnqueueForDistinctEventsEnqueuesBoth(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	p := newHookPlugin(s)
	appID, userID := id.NewAppID(), id.NewUserID()

	p.enqueueFor(ctx, appID, id.EnvironmentID{}, userID, KindActivityLog, "logged_in", "ases_1")
	p.enqueueFor(ctx, appID, id.EnvironmentID{}, userID, KindActivityLog, "logged_in", "ases_2")

	jobs, err := s.ClaimDue(ctx, 10, 0, timeNow())
	require.NoError(t, err)
	assert.Len(t, jobs, 2, "two separate logins must not collapse into one job")
}

func TestPluginImplementsHookInterfaces(t *testing.T) {
	p := New()
	assert.Implements(t, (*plugin.AfterSignUp)(nil), p)
	assert.Implements(t, (*plugin.AfterSignIn)(nil), p)
	assert.Implements(t, (*plugin.AfterUserUpdate)(nil), p)
}
