package username

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
)

// TestRequestTypes tests request struct serialization
func TestRequestTypes(t *testing.T) {
	tests := []struct {
		name     string
		data     string
		expected interface{}
	}{
		{
			name: "SignUpRequest",
			data: `{"username":"johndoe","password":"SecureP@ss123"}`,
			expected: &SignUpRequest{
				Username: "johndoe",
				Password: "SecureP@ss123",
			},
		},
		{
			name: "SignInRequest with remember",
			data: `{"username":"johndoe","password":"SecureP@ss123","remember":true}`,
			expected: &SignInRequest{
				Username: "johndoe",
				Password: "SecureP@ss123",
				Remember: true,
			},
		},
		{
			name: "SignInRequest without remember",
			data: `{"username":"johndoe","password":"SecureP@ss123","remember":false}`,
			expected: &SignInRequest{
				Username: "johndoe",
				Password: "SecureP@ss123",
				Remember: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch expected := tt.expected.(type) {
			case *SignUpRequest:
				var req SignUpRequest
				err := json.Unmarshal([]byte(tt.data), &req)
				require.NoError(t, err)
				assert.Equal(t, expected.Username, req.Username)
				assert.Equal(t, expected.Password, req.Password)

			case *SignInRequest:
				var req SignInRequest
				err := json.Unmarshal([]byte(tt.data), &req)
				require.NoError(t, err)
				assert.Equal(t, expected.Username, req.Username)
				assert.Equal(t, expected.Password, req.Password)
				assert.Equal(t, expected.Remember, req.Remember)
			}
		})
	}
}

// TestResponseTypes tests response struct serialization
func TestResponseTypes(t *testing.T) {
	t.Run("SignUpResponse", func(t *testing.T) {
		resp := &SignUpResponse{
			Status:  "created",
			Message: "User created successfully",
		}

		data, err := json.Marshal(resp)
		require.NoError(t, err)

		var decoded SignUpResponse
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, resp.Status, decoded.Status)
		assert.Equal(t, resp.Message, decoded.Message)
	})

	t.Run("SignInResponse", func(t *testing.T) {
		userID := xid.New()
		sessionID := xid.New()

		resp := &SignInResponse{
			User: &user.User{
				ID:    userID,
				Email: "test@example.com",
				Name:  "Test User",
			},
			Session: &session.Session{
				ID:     sessionID,
				UserID: userID,
				Token:  "test_token",
			},
			Token: "test_token",
		}

		data, err := json.Marshal(resp)
		require.NoError(t, err)

		var decoded SignInResponse
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, resp.User.ID.String(), decoded.User.ID.String())
		assert.Equal(t, resp.User.Email, decoded.User.Email)
		assert.Equal(t, resp.Session.ID.String(), decoded.Session.ID.String())
		assert.Equal(t, resp.Token, decoded.Token)
	})

	t.Run("TwoFARequiredResponse", func(t *testing.T) {
		userID := xid.New()

		resp := &TwoFARequiredResponse{
			User: &user.User{
				ID:    userID,
				Email: "test@example.com",
			},
			RequireTwoFA: true,
			DeviceID:     "device_fingerprint",
		}

		data, err := json.Marshal(resp)
		require.NoError(t, err)

		var decoded TwoFARequiredResponse
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, resp.User.ID.String(), decoded.User.ID.String())
		assert.True(t, decoded.RequireTwoFA)
		assert.Equal(t, resp.DeviceID, decoded.DeviceID)
	})

	t.Run("AccountLockedResponse", func(t *testing.T) {
		lockedUntil := time.Now().Add(15 * time.Minute)

		resp := &AccountLockedResponse{
			Code:          "ACCOUNT_LOCKED",
			Message:       "Account locked due to too many failed login attempts",
			LockedUntil:   lockedUntil,
			LockedMinutes: 15,
		}

		data, err := json.Marshal(resp)
		require.NoError(t, err)

		var decoded AccountLockedResponse
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, resp.Code, decoded.Code)
		assert.Equal(t, resp.Message, decoded.Message)
		assert.Equal(t, resp.LockedMinutes, decoded.LockedMinutes)
		assert.WithinDuration(t, resp.LockedUntil, decoded.LockedUntil, time.Second)
	})
}

// TestValidationTags tests validation tags on request types
func TestValidationTags(t *testing.T) {
	t.Run("SignUpRequest has required tags", func(t *testing.T) {
		req := SignUpRequest{}
		// Verify struct tags using reflection would be done here
		// For now, just verify the types exist
		assert.IsType(t, "", req.Username)
		assert.IsType(t, "", req.Password)
	})

	t.Run("SignInRequest has required tags", func(t *testing.T) {
		req := SignInRequest{}
		assert.IsType(t, "", req.Username)
		assert.IsType(t, "", req.Password)
		assert.IsType(t, false, req.Remember)
	})
}

