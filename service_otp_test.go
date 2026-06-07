package authsome_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/account"
)

func wrongCode(code string) string {
	if code == "000000" {
		return "111111"
	}
	return "000000"
}

// SignUp should mint an email-verification OTP for the new (unverified) user.
func TestSignUp_IssuesEmailVerificationCode(t *testing.T) {
	eng, st := newTestEngine(t)
	ctx := context.Background()
	appID := testAppID(t)

	u, _, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID:    appID,
		Email:    "bob@example.com",
		Password: "SecureP@ss1",
		Username: "bob",
	})
	require.NoError(t, err)

	v, err := st.GetActiveEmailVerification(ctx, u.ID)
	require.NoError(t, err)
	assert.Len(t, v.Token, 6, "verification should carry a 6-digit code")
	assert.Equal(t, account.VerificationEmail, v.Type)
	assert.Equal(t, 0, v.Attempts)
}

func TestVerifyEmailCode_WrongThenRight(t *testing.T) {
	eng, st := newTestEngine(t)
	ctx := context.Background()
	appID := testAppID(t)

	u, _, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID:    appID,
		Email:    "carol@example.com",
		Password: "SecureP@ss1",
		Username: "carol",
	})
	require.NoError(t, err)

	v, err := st.GetActiveEmailVerification(ctx, u.ID)
	require.NoError(t, err)
	code := v.Token

	// Wrong code fails and increments attempts.
	err = eng.VerifyEmailCode(ctx, u.ID, wrongCode(code))
	require.ErrorIs(t, err, account.ErrInvalidCredentials)

	v2, err := st.GetActiveEmailVerification(ctx, u.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, v2.Attempts, "failed attempt should be recorded")

	// Correct code verifies the email and consumes the code.
	require.NoError(t, eng.VerifyEmailCode(ctx, u.ID, code))

	gu, err := st.GetUser(ctx, u.ID)
	require.NoError(t, err)
	assert.True(t, gu.EmailVerified)

	_, err = st.GetActiveEmailVerification(ctx, u.ID)
	require.Error(t, err, "code should be consumed after success")
}

func TestVerifyEmailCode_MaxAttempts(t *testing.T) {
	eng, st := newTestEngine(t)
	ctx := context.Background()
	appID := testAppID(t)

	u, _, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID:    appID,
		Email:    "dave@example.com",
		Password: "SecureP@ss1",
		Username: "dave",
	})
	require.NoError(t, err)

	v, err := st.GetActiveEmailVerification(ctx, u.ID)
	require.NoError(t, err)
	code := v.Token
	wrong := wrongCode(code)

	// Exhaust the attempt budget with wrong codes.
	for i := 0; i < 5; i++ {
		require.ErrorIs(t, eng.VerifyEmailCode(ctx, u.ID, wrong), account.ErrInvalidCredentials)
	}

	// Even the correct code is now rejected (locked out until resend).
	require.ErrorIs(t, eng.VerifyEmailCode(ctx, u.ID, code), account.ErrTooManyAttempts)
}
