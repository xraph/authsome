package retention

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
)

// These cover the DropRef plumbing end to end: a provider that reports the
// remote record is gone (classifyHTTPError's 404 case) makes the worker
// delete the local ref, but only when this attempt actually held one going
// in. See ensureRef's hadRef and dropRefIfNeeded in worker.go.

func TestWorkerDropsRefOnDropRefWhenARefAlreadyExisted(t *testing.T) {
	ctx := context.Background()
	mem := NewMemoryStore()
	p := &fakeProvider{
		caps: CapContacts | CapActivities,
		activityErr: []error{&ProviderError{
			Err: errors.New("404 gone"), Retryable: true, DropRef: true,
		}},
	}
	j := enqueued(t, mem, KindActivityLog, "dropref1")

	// Seed an existing ref, as if a previous delivery had already created
	// the contact.
	existing := &ContactRef{
		ID: id.NewRetentionRefID(), AppID: j.AppID, EnvID: j.EnvID, UserID: j.UserID,
		Provider: "fake", RemoteObjectType: "contact", RemoteID: "was-here",
		SyncedAt: time.Now(),
	}
	require.NoError(t, mem.PutRef(ctx, existing))

	newTestWorker(t, mem, p).runOnce(ctx)

	_, err := mem.GetRef(ctx, j.AppID, j.EnvID, j.UserID, "fake")
	assert.ErrorIs(t, err, ErrNotFound, "the stale ref must be dropped so the next attempt recreates the contact")

	stored, err := mem.GetJob(ctx, j.ID)
	require.NoError(t, err)
	assert.Equal(t, StatePending, stored.State, "a retryable error must be queued for retry, not dead-lettered")
	assert.Equal(t, 1, stored.Attempts)
}

func TestWorkerDoesNotDropARefThatNeverExisted(t *testing.T) {
	ctx := context.Background()
	mem := NewMemoryStore()
	p := &fakeProvider{
		caps: CapContacts | CapActivities,
		upsertErr: []error{&ProviderError{
			Err: errors.New("404 on create"), Retryable: true, DropRef: true,
		}},
	}
	// No ref seeded: this is a create, not an update.
	j := enqueued(t, mem, KindContactUpsert, "dropref2")

	newTestWorker(t, mem, p).runOnce(ctx)

	_, err := mem.GetRef(ctx, j.AppID, j.EnvID, j.UserID, "fake")
	assert.ErrorIs(t, err, ErrNotFound, "there was never a ref to hold, so this is just the ordinary absence")

	stored, err := mem.GetJob(ctx, j.ID)
	require.NoError(t, err)
	assert.Equal(t, StatePending, stored.State, "the failure is still retryable on its own terms")
	assert.Equal(t, 1, stored.Attempts)
}
