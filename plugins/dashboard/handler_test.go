package dashboard

import (
	"embed"
	"html/template"
	"testing"

	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/core/rbac"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
)

func TestNewHandler(t *testing.T) {
	tmpl := template.Must(template.New("test").Parse("test"))
	assets := embed.FS{}
	userSvc := &user.Service{}
	sessionSvc := &session.Service{}
	auditSvc := &audit.Service{}
	rbacSvc := &rbac.Service{}
	basePath := "/api/auth"
	
	handler := NewHandler(tmpl, assets, userSvc, sessionSvc, auditSvc, rbacSvc, basePath)
	
	require.NotNil(t, handler)
	assert.Equal(t, tmpl, handler.templates)
	assert.Equal(t, userSvc, handler.userSvc)
	assert.Equal(t, sessionSvc, handler.sessionSvc)
	assert.Equal(t, auditSvc, handler.auditSvc)
	assert.Equal(t, rbacSvc, handler.rbacSvc)
	assert.Equal(t, basePath, handler.basePath)
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
