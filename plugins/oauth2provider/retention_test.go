package oauth2provider_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/oauth2provider"
	"github.com/xraph/authsome/store"
)

// The plugin sweeps both code kinds in batches until short, and leaves a
// kind alone when its cutoff is the zero time.
func TestSweepRetention(t *testing.T) {
	ctx := context.Background()
	st := oauth2provider.NewMemoryStore()
	p := oauth2provider.New()
	p.SetOAuth2Store(st)
	appID, userID := id.NewAppID(), id.NewUserID()
	old := time.Now().Add(-48 * time.Hour)

	for i := range 5 {
		require.NoError(t, st.CreateAuthCode(ctx, &oauth2provider.AuthorizationCode{
			ID: id.NewAuthCodeID(), Code: "ac-" + string(rune('a'+i)), ClientID: "c", UserID: userID, AppID: appID,
			RedirectURI: "https://x.test/cb", CodeChallenge: "x", CodeChallengeMethod: "S256", ExpiresAt: old, CreatedAt: old,
		}))
	}
	require.NoError(t, st.CreateDeviceCode(ctx, &oauth2provider.DeviceCode{
		ID: id.NewDeviceCodeID(), DeviceCode: "dc", UserCode: "uc", ClientID: "c", AppID: appID,
		ExpiresAt: old, Status: oauth2provider.DeviceCodeStatusPending, CreatedAt: old,
	}))

	cutoff := func(kind string) time.Time {
		if kind == store.RetentionAuthCodes {
			return time.Now().Add(-24 * time.Hour)
		}
		return time.Time{}
	}
	n, err := p.SweepRetention(ctx, cutoff, 2)
	require.NoError(t, err)
	assert.EqualValues(t, 5, n, "every expired auth code goes, two at a time")
	_, err = st.GetDeviceCodeByDeviceCode(ctx, "dc")
	assert.NoError(t, err, "device codes were not asked for")

	n, err = p.SweepRetention(ctx, func(string) time.Time { return time.Now() }, 0)
	require.NoError(t, err)
	assert.EqualValues(t, 1, n, "now the device code goes too")
}
