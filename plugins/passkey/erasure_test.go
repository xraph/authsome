package passkey

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
)

// A deleted user's passkeys go with the account.
func TestOnBeforeUserDelete_RemovesCredentials(t *testing.T) {
	p := New()
	st := NewMemoryStore()
	p.SetStore(st)
	ctx := context.Background()
	userID, other := id.NewUserID(), id.NewUserID()
	appID := id.NewAppID()
	require.NoError(t, st.CreateCredential(ctx, &Credential{ID: id.NewPasskeyID(), UserID: userID, AppID: appID, CredentialID: []byte("mine")}))
	require.NoError(t, st.CreateCredential(ctx, &Credential{ID: id.NewPasskeyID(), UserID: other, AppID: appID, CredentialID: []byte("theirs")}))

	require.NoError(t, p.OnBeforeUserDelete(ctx, userID))

	mine, err := st.ListUserCredentials(ctx, userID)
	require.NoError(t, err)
	assert.Empty(t, mine)
	theirs, err := st.ListUserCredentials(ctx, other)
	require.NoError(t, err)
	assert.Len(t, theirs, 1, "another user's passkeys stay")
}
