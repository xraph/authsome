package impersonation

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/authsome/core/impersonation"
	"github.com/xraph/authsome/schema"
)

func TestMiddleware_Handle_WithImpersonation(t *testing.T) {
	service, repo, userSvc, sessionSvc := setupTestService(t)
	admin, target := createTestUsers(userSvc)
	orgID := xid.New()

	// Create impersonation session
	sessionID := xid.New()
	sessionToken := "token_test_123"
	session := &schema.Session{
		ID:        sessionID,
		Token:     sessionToken,
		UserID:    target.ID,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	sessionSvc.sessions[sessionID.String()] = session

	impSession := &schema.ImpersonationSession{
		ID:              xid.New(),
		OrganizationID:  orgID,
		ImpersonatorID:  admin.ID,
		TargetUserID:    target.ID,
		NewSessionID:    &sessionID,
		SessionToken:    sessionToken,
		Active:          true,
		ExpiresAt:       time.Now().Add(1 * time.Hour),
		CreatedAt:       time.Now(),
	}
	repo.sessions[impSession.ID.String()] = impSession

	config := DefaultConfig()
	config.ShowIndicator = true
	middleware := NewMiddleware(service, config)

	// Create test request with session cookie
	req := httptest.NewRequest("GET", "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  "session_token",
		Value: sessionID.String(),
	})

	ctx := &mockForgeContext{
		request:  req,
		response: httptest.NewRecorder(),
		params:   make(map[string]string),
	}

	handler := middleware.Handle()
	err := handler(ctx)

	require.NoError(t, err)

	// Verify impersonation headers were set
	recorder := ctx.response.(*httptest.ResponseRecorder)
	assert.Equal(t, "true", recorder.Header().Get("X-Impersonating"))
	assert.Equal(t, admin.ID.String(), recorder.Header().Get("X-Impersonator-ID"))
	assert.Equal(t, target.ID.String(), recorder.Header().Get("X-Target-User-ID"))
}

func TestMiddleware_Handle_WithoutImpersonation(t *testing.T) {
	service, _, _, sessionSvc := setupTestService(t)

	// Create regular (non-impersonation) session
	sessionID := xid.New()
	sessionToken := "token_regular_123"
	session := &schema.Session{
		ID:        sessionID,
		Token:     sessionToken,
		UserID:    xid.New(),
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	sessionSvc.sessions[sessionID.String()] = session

	config := DefaultConfig()
	middleware := NewMiddleware(service, config)

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  "session_token",
		Value: sessionID.String(),
	})

	ctx := &mockForgeContext{
		request:  req,
		response: httptest.NewRecorder(),
		params:   make(map[string]string),
	}

	handler := middleware.Handle()
	err := handler(ctx)

	require.NoError(t, err)

	// Verify no impersonation headers were set
	recorder := ctx.response.(*httptest.ResponseRecorder)
	assert.Empty(t, recorder.Header().Get("X-Impersonating"))
	assert.Empty(t, recorder.Header().Get("X-Impersonator-ID"))
	assert.Empty(t, recorder.Header().Get("X-Target-User-ID"))
}

func TestMiddleware_RequireNoImpersonation_Allowed(t *testing.T) {
	service, _, _, _ := setupTestService(t)
	config := DefaultConfig()
	middleware := NewMiddleware(service, config)

	// Create context without impersonation
	req := httptest.NewRequest("GET", "/test", nil)
	ctx := &mockForgeContext{
		request:  req,
		response: httptest.NewRecorder(),
		params:   make(map[string]string),
	}

	handler := middleware.RequireNoImpersonation()
	err := handler(ctx)

	require.NoError(t, err)
	// Should pass through without error
}

func TestMiddleware_RequireNoImpersonation_Blocked(t *testing.T) {
	service, _, _, _ := setupTestService(t)
	config := DefaultConfig()
	middleware := NewMiddleware(service, config)

	// Create context with impersonation
	impCtx := &ImpersonationContext{
		IsImpersonating: true,
		ImpersonationID: func() *xid.ID { id := xid.New(); return &id }(),
		ImpersonatorID:  func() *xid.ID { id := xid.New(); return &id }(),
		TargetUserID:    func() *xid.ID { id := xid.New(); return &id }(),
	}

	req := httptest.NewRequest("GET", "/test", nil)
	reqCtx := context.WithValue(req.Context(), ImpersonationContextKey, impCtx)
	req = req.WithContext(reqCtx)

	ctx := &mockForgeContext{
		request:  req,
		response: httptest.NewRecorder(),
		params:   make(map[string]string),
	}

	handler := middleware.RequireNoImpersonation()
	err := handler(ctx)

	require.NoError(t, err)
	assert.Equal(t, 403, ctx.GetStatus())

	body := ctx.GetBody().(map[string]string)
	assert.Equal(t, "This action is not allowed during impersonation", body["error"])
}

func TestMiddleware_RequireImpersonation_Allowed(t *testing.T) {
	service, _, _, _ := setupTestService(t)
	config := DefaultConfig()
	middleware := NewMiddleware(service, config)

	// Create context with impersonation
	impCtx := &ImpersonationContext{
		IsImpersonating: true,
		ImpersonationID: func() *xid.ID { id := xid.New(); return &id }(),
	}

	req := httptest.NewRequest("GET", "/test", nil)
	reqCtx := context.WithValue(req.Context(), ImpersonationContextKey, impCtx)
	req = req.WithContext(reqCtx)

	ctx := &mockForgeContext{
		request:  req,
		response: httptest.NewRecorder(),
		params:   make(map[string]string),
	}

	handler := middleware.RequireImpersonation()
	err := handler(ctx)

	require.NoError(t, err)
	// Should pass through without error
}

func TestMiddleware_RequireImpersonation_Blocked(t *testing.T) {
	service, _, _, _ := setupTestService(t)
	config := DefaultConfig()
	middleware := NewMiddleware(service, config)

	// Create context without impersonation
	req := httptest.NewRequest("GET", "/test", nil)
	ctx := &mockForgeContext{
		request:  req,
		response: httptest.NewRecorder(),
		params:   make(map[string]string),
	}

	handler := middleware.RequireImpersonation()
	err := handler(ctx)

	require.NoError(t, err)
	assert.Equal(t, 403, ctx.GetStatus())

	body := ctx.GetBody().(map[string]string)
	assert.Equal(t, "This action requires an active impersonation session", body["error"])
}

func TestMiddleware_GetImpersonationContext(t *testing.T) {
	impCtx := &ImpersonationContext{
		IsImpersonating: true,
		ImpersonationID: func() *xid.ID { id := xid.New(); return &id }(),
		ImpersonatorID:  func() *xid.ID { id := xid.New(); return &id }(),
		TargetUserID:    func() *xid.ID { id := xid.New(); return &id }(),
		IndicatorMsg:    "Test message",
	}

	req := httptest.NewRequest("GET", "/test", nil)
	reqCtx := context.WithValue(req.Context(), ImpersonationContextKey, impCtx)
	req = req.WithContext(reqCtx)

	ctx := &mockForgeContext{
		request:  req,
		response: httptest.NewRecorder(),
		params:   make(map[string]string),
	}

	retrieved := GetImpersonationContext(ctx)

	require.NotNil(t, retrieved)
	assert.True(t, retrieved.IsImpersonating)
	assert.NotNil(t, retrieved.ImpersonationID)
	assert.NotNil(t, retrieved.ImpersonatorID)
	assert.NotNil(t, retrieved.TargetUserID)
	assert.Equal(t, "Test message", retrieved.IndicatorMsg)
}

func TestMiddleware_GetImpersonationContext_NotPresent(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	ctx := &mockForgeContext{
		request:  req,
		response: httptest.NewRecorder(),
		params:   make(map[string]string),
	}

	retrieved := GetImpersonationContext(ctx)

	assert.Nil(t, retrieved)
}

func TestMiddleware_IsImpersonating(t *testing.T) {
	tests := []struct {
		name     string
		context  *ImpersonationContext
		expected bool
	}{
		{
			name: "impersonating",
			context: &ImpersonationContext{
				IsImpersonating: true,
			},
			expected: true,
		},
		{
			name: "not impersonating",
			context: &ImpersonationContext{
				IsImpersonating: false,
			},
			expected: false,
		},
		{
			name:     "no context",
			context:  nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)

			if tt.context != nil {
				reqCtx := context.WithValue(req.Context(), ImpersonationContextKey, tt.context)
				req = req.WithContext(reqCtx)
			}

			ctx := &mockForgeContext{
				request:  req,
				response: httptest.NewRecorder(),
				params:   make(map[string]string),
			}

			result := IsImpersonating(ctx)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMiddleware_GetImpersonatorID(t *testing.T) {
	expectedID := xid.New()

	impCtx := &ImpersonationContext{
		IsImpersonating: true,
		ImpersonatorID:  &expectedID,
	}

	req := httptest.NewRequest("GET", "/test", nil)
	reqCtx := context.WithValue(req.Context(), ImpersonationContextKey, impCtx)
	req = req.WithContext(reqCtx)

	ctx := &mockForgeContext{
		request:  req,
		response: httptest.NewRecorder(),
		params:   make(map[string]string),
	}

	result := GetImpersonatorID(ctx)

	require.NotNil(t, result)
	assert.Equal(t, expectedID, *result)
}

func TestMiddleware_GetImpersonatorID_NotImpersonating(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	ctx := &mockForgeContext{
		request:  req,
		response: httptest.NewRecorder(),
		params:   make(map[string]string),
	}

	result := GetImpersonatorID(ctx)

	assert.Nil(t, result)
}

func TestMiddleware_GetTargetUserID(t *testing.T) {
	expectedID := xid.New()

	impCtx := &ImpersonationContext{
		IsImpersonating: true,
		TargetUserID:    &expectedID,
	}

	req := httptest.NewRequest("GET", "/test", nil)
	reqCtx := context.WithValue(req.Context(), ImpersonationContextKey, impCtx)
	req = req.WithContext(reqCtx)

	ctx := &mockForgeContext{
		request:  req,
		response: httptest.NewRecorder(),
		params:   make(map[string]string),
	}

	result := GetTargetUserID(ctx)

	require.NotNil(t, result)
	assert.Equal(t, expectedID, *result)
}

func TestMiddleware_AuditImpersonationAction(t *testing.T) {
	service, _, _, _ := setupTestService(t)
	config := DefaultConfig()
	config.AuditAllActions = true
	middleware := NewMiddleware(service, config)

	// Create context with impersonation
	impCtx := &ImpersonationContext{
		IsImpersonating: true,
		ImpersonationID: func() *xid.ID { id := xid.New(); return &id }(),
	}

	req := httptest.NewRequest("GET", "/test/path", nil)
	reqCtx := context.WithValue(req.Context(), ImpersonationContextKey, impCtx)
	req = req.WithContext(reqCtx)

	ctx := &mockForgeContext{
		request:  req,
		response: httptest.NewRecorder(),
		params:   make(map[string]string),
	}

	handler := middleware.AuditImpersonationAction()
	err := handler(ctx)

	require.NoError(t, err)

	// Verify action was added to context
	action := ctx.request.Context().Value("impersonation_action")
	require.NotNil(t, action)
	assert.Contains(t, action.(string), "GET")
	assert.Contains(t, action.(string), "/test/path")
}

func TestMiddleware_AuditImpersonationAction_Disabled(t *testing.T) {
	service, _, _, _ := setupTestService(t)
	config := DefaultConfig()
	config.AuditAllActions = false // Disabled
	middleware := NewMiddleware(service, config)

	impCtx := &ImpersonationContext{
		IsImpersonating: true,
	}

	req := httptest.NewRequest("GET", "/test", nil)
	reqCtx := context.WithValue(req.Context(), ImpersonationContextKey, impCtx)
	req = req.WithContext(reqCtx)

	ctx := &mockForgeContext{
		request:  req,
		response: httptest.NewRecorder(),
		params:   make(map[string]string),
	}

	handler := middleware.AuditImpersonationAction()
	err := handler(ctx)

	require.NoError(t, err)

	// Verify action was NOT added to context
	action := ctx.request.Context().Value("impersonation_action")
	assert.Nil(t, action)
}

func TestMiddleware_IndicatorMessage_Custom(t *testing.T) {
	service, repo, userSvc, sessionSvc := setupTestService(t)
	admin, target := createTestUsers(userSvc)
	orgID := xid.New()

	customMessage := "🚨 ADMIN MODE: Viewing as customer"

	// Create impersonation session
	sessionID := xid.New()
	sessionToken := "token_test_456"
	session := &schema.Session{
		ID:        sessionID,
		Token:     sessionToken,
		UserID:    target.ID,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	sessionSvc.sessions[sessionID.String()] = session

	impSession := &schema.ImpersonationSession{
		ID:              xid.New(),
		OrganizationID:  orgID,
		ImpersonatorID:  admin.ID,
		TargetUserID:    target.ID,
		NewSessionID:    &sessionID,
		SessionToken:    sessionToken,
		Active:          true,
		ExpiresAt:       time.Now().Add(1 * time.Hour),
		CreatedAt:       time.Now(),
	}
	repo.sessions[impSession.ID.String()] = impSession

	config := DefaultConfig()
	config.ShowIndicator = true
	config.IndicatorMessage = customMessage
	middleware := NewMiddleware(service, config)

	req := httptest.NewRequest("GET", "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  "session_token",
		Value: sessionID.String(),
	})

	// Simulate middleware adding impersonation context
	impCtx := &ImpersonationContext{
		IsImpersonating: true,
		IndicatorMsg:    customMessage,
	}
	reqCtx := context.WithValue(req.Context(), ImpersonationContextKey, impCtx)
	req = req.WithContext(reqCtx)

	ctx := &mockForgeContext{
		request:  req,
		response: httptest.NewRecorder(),
		params:   make(map[string]string),
	}

	retrieved := GetImpersonationContext(ctx)
	require.NotNil(t, retrieved)
	assert.Equal(t, customMessage, retrieved.IndicatorMsg)
}

