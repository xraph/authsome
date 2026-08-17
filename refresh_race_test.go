package authsome_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/account"
)

// TestRefresh_ConcurrentRotation_ExactlyOneWinner exercises the refresh-token
// rotation TOCTOU: many goroutines present the SAME valid refresh token at
// once. With the compare-and-swap rotation, exactly one may win and persist its
// rotated tokens; every other caller is refused rather than returning tokens
// that were never persisted (which previously silently signed clients out).
// Also run under -race to confirm the store no longer shares a live pointer.
func TestRefresh_ConcurrentRotation_ExactlyOneWinner(t *testing.T) {
	eng, _ := newTestEngine(t)
	ctx := context.Background()
	appID := testAppID(t)

	_, sess, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID:     appID,
		Email:     "refresh-race@example.com",
		Password:  "SecureP@ss1",
		FirstName: "Race",
	})
	require.NoError(t, err)
	rt := sess.RefreshToken

	const n = 12
	var wg sync.WaitGroup
	errs := make([]error, n)
	start := make(chan struct{})
	for i := range n {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			_, errs[i] = eng.Refresh(ctx, rt)
		}(i)
	}
	close(start)
	wg.Wait()

	successes := 0
	for _, e := range errs {
		switch {
		case e == nil:
			successes++
		case errors.Is(e, account.ErrInvalidCredentials):
			// expected loser
		default:
			t.Fatalf("unexpected refresh error: %v", e)
		}
	}

	require.Equal(t, 1, successes, "exactly one concurrent refresh may win the rotation")
}
