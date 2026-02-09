package phone

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRequestResponseTypes tests request and response type serialization.
func TestRequestResponseTypes(t *testing.T) {
	t.Run("SendCodeRequest serialization", func(t *testing.T) {
		req := SendCodeRequest{
			Phone: "+12345678901",
		}

		data, err := json.Marshal(req)
		require.NoError(t, err)

		var decoded SendCodeRequest

		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)
		assert.Equal(t, req.Phone, decoded.Phone)
	})

	t.Run("VerifyRequest serialization", func(t *testing.T) {
		req := VerifyRequest{
			Phone:    "+12345678901",
			Code:     "123456",
			Email:    "test@example.com",
			Remember: true,
		}

		data, err := json.Marshal(req)
		require.NoError(t, err)

		var decoded VerifyRequest

		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)
		assert.Equal(t, req.Phone, decoded.Phone)
		assert.Equal(t, req.Code, decoded.Code)
		assert.Equal(t, req.Email, decoded.Email)
		assert.Equal(t, req.Remember, decoded.Remember)
	})

	t.Run("SendCodeResponse serialization", func(t *testing.T) {
		resp := SendCodeResponse{
			Status:  "sent",
			DevCode: "123456",
		}

		data, err := json.Marshal(resp)
		require.NoError(t, err)

		var decoded SendCodeResponse

		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)
		assert.Equal(t, resp.Status, decoded.Status)
		assert.Equal(t, resp.DevCode, decoded.DevCode)
	})
}
