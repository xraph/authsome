package dashboard

import (
	"embed"
	"testing"

	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core/apikey"
	"github.com/xraph/authsome/core/app"
	"github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/core/rbac"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
)

func TestNewHandler(t *testing.T) {
	assets := embed.FS{}
	userSvc := &user.Service{}
	sessionSvc := &session.Service{}
	auditSvc := &audit.Service{}
	rbacSvc := &rbac.Service{}
	apikeyService := &apikey.Service{}
	orgService := &app.Service{}
	db := &bun.DB{}
	isSaaSMode := false
	basePath := "/api/auth"

	handler := NewHandler(
		assets,
		userSvc,
		sessionSvc,
		auditSvc,
		rbacSvc,
		apikeyService,
		orgService,
		db,
		isSaaSMode,
		basePath,
	)

	require.NotNil(t, handler)
	assert.Equal(t, userSvc, handler.userSvc)
	assert.Equal(t, sessionSvc, handler.sessionSvc)
	assert.Equal(t, auditSvc, handler.auditSvc)
	assert.Equal(t, rbacSvc, handler.rbacSvc)
	assert.Equal(t, apikeyService, handler.apikeyService)
	assert.Equal(t, orgService, handler.orgService)
	assert.Equal(t, basePath, handler.basePath)
	assert.Equal(t, isSaaSMode, handler.isSaaSMode)
}

func TestGetContentType(t *testing.T) {
	tests := []struct {
		ext      string
		expected string
	}{
		{".css", "text/css; charset=utf-8"},
		{".js", "application/javascript; charset=utf-8"},
		{".png", "image/png"},
		{".jpg", "image/jpeg"},
		{".jpeg", "image/jpeg"},
		{".gif", "image/gif"},
		{".svg", "image/svg+xml"},
		{".ico", "image/x-icon"},
		{".woff", "font/woff"},
		{".woff2", "font/woff2"},
		{".ttf", "font/ttf"},
		{".xyz", "application/octet-stream"},
	}

	for _, tt := range tests {
		t.Run(tt.ext, func(t *testing.T) {
			result := getContentType(tt.ext)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPageData(t *testing.T) {
	// Test PageData structure
	testUser := &user.User{
		ID:    xid.New(),
		Email: "test@example.com",
	}

	data := PageData{
		Title:      "Test Page",
		ActivePage: "dashboard",
		User:       testUser,
		CSRFToken:  "test-token",
		BasePath:   "/api/auth",
		Error:      "test error",
		Success:    "test success",
	}

	assert.Equal(t, "Test Page", data.Title)
	assert.Equal(t, "dashboard", data.ActivePage)
	assert.Equal(t, testUser, data.User)
	assert.Equal(t, "test-token", data.CSRFToken)
	assert.Equal(t, "/api/auth", data.BasePath)
	assert.Equal(t, "test error", data.Error)
	assert.Equal(t, "test success", data.Success)
}

// Note: Full integration tests for handlers would require mocking forge.Context
// and setting up a complete test environment, which would be done in integration tests
