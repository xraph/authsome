package retention

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
)

func TestExportUserDataIncludesCRMRefs(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	p := New()
	p.store = s

	appID, userID := id.NewAppID(), id.NewUserID()
	require.NoError(t, s.PutRef(ctx, &ContactRef{
		ID: id.NewRetentionRefID(), AppID: appID, UserID: userID,
		Provider: "hubspot", RemoteObjectType: "contact", RemoteID: "501",
		SyncedAt: time.Now(),
	}))

	category, data, err := p.ExportUserData(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, "retention", category)
	assert.NotNil(t, data, "a user must be able to see which CRMs hold their record")

	refs, ok := data.([]*ContactRef)
	require.True(t, ok, "export data should be the ref slice")
	require.Len(t, refs, 1)
	assert.Equal(t, "hubspot", refs[0].Provider)
	assert.Equal(t, "501", refs[0].RemoteID)
}

func TestExportUserDataEmptyWithoutStore(t *testing.T) {
	category, data, err := New().ExportUserData(context.Background(), id.NewUserID())
	require.NoError(t, err)
	assert.Equal(t, "retention", category)
	assert.Nil(t, data)
}

func TestExportUserDataEmptyWhenNoRefs(t *testing.T) {
	p := New()
	p.store = NewMemoryStore()

	category, data, err := p.ExportUserData(context.Background(), id.NewUserID())
	require.NoError(t, err)
	assert.Equal(t, "retention", category)
	assert.Nil(t, data)
}
