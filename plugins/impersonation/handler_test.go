package impersonation

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/authsome/core/impersonation"
	"github.com/xraph/authsome/schema"
	"github.com/xraph/forge"
)

// mockForgeContext implements forge.Context for testing
type mockForgeContext struct {
	request  *http.Request
	response http.ResponseWriter
	params   map[string]string
	status   int
	body     interface{}
}

func newMockForgeContext(method, path string, body interface{}) *mockForgeContext {
	var bodyBytes []byte
	if body != nil {
		bodyBytes, _ = json.Marshal(body)
	}

	req := httptest.NewRequest(method, path, bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	return &mockForgeContext{
		request:  req,
		response: httptest.NewRecorder(),
		params:   make(map[string]string),
	}
}

func (c *mockForgeContext) Request() *http.Request {
	return c.request
}

func (c *mockForgeContext) Response() http.ResponseWriter {
	return c.response
}

func (c *mockForgeContext) Param(name string) string {
	return c.params[name]
}

func (c *mockForgeContext) SetParam(name, value string) {
	c.params[name] = value
}

func (c *mockForgeContext) JSON(status int, data interface{}) error {
	c.status = status
	c.body = data
	return nil
}

func (c *mockForgeContext) GetStatus() int {
	return c.status
}

func (c *mockForgeContext) GetBody() interface{} {
	return c.body
}

// Test helpers
func setupTestHandler(t *testing.T) (*Handler, *impersonation.Service, *mockImpersonationRepository, *mockUserService) {
	service, repo, userSvc, _ := setupTestService(t)
	config := DefaultConfig()
	handler := NewHandler(service, config)
	return handler, service, repo, userSvc
}

// Tests

func TestHandler_StartImpersonation_Success(t *testing.T) {
	handler, _, _, userSvc := setupTestHandler(t)
	admin, target := createTestUsers(userSvc)
	orgID := xid.New()

	reqBody := impersonation.StartRequest{
		OrganizationID:  orgID,
		ImpersonatorID:  admin.ID,
		TargetUserID:    target.ID,
		Reason:          "Customer support ticket #12345 - investigating login issue",
		TicketNumber:    "TICKET-12345",
		DurationMinutes: 30,
	}

	ctx := newMockForgeContext("POST", "/api/impersonation/start", reqBody)

	err := handler.StartImpersonation(ctx)

	require.NoError(t, err)
	assert.Equal(t, 200, ctx.GetStatus())

	// Verify response structure
	body := ctx.GetBody().(map[string]interface{})
	assert.NotEmpty(t, body["impersonation_id"])
	assert.NotEmpty(t, body["session_id"])
	assert.NotEmpty(t, body["session_token"])
	assert.NotEmpty(t, body["expires_at"])
	assert.NotEmpty(t, body["message"])
}

func TestHandler_StartImpersonation_InvalidJSON(t *testing.T) {
	handler, _, _, _ := setupTestHandler(t)

	// Send invalid JSON
	req := httptest.NewRequest("POST", "/api/impersonation/start", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	ctx := &mockForgeContext{
		request:  req,
		response: httptest.NewRecorder(),
		params:   make(map[string]string),
	}

	err := handler.StartImpersonation(ctx)

	require.NoError(t, err)
	assert.Equal(t, 400, ctx.GetStatus())

	body := ctx.GetBody().(map[string]string)
	assert.Equal(t, "Invalid request body", body["error"])
}

func TestHandler_StartImpersonation_CannotImpersonateSelf(t *testing.T) {
	handler, _, _, userSvc := setupTestHandler(t)
	admin, _ := createTestUsers(userSvc)
	orgID := xid.New()

	reqBody := impersonation.StartRequest{
		OrganizationID: orgID,
		ImpersonatorID: admin.ID,
		TargetUserID:   admin.ID, // Same as impersonator
		Reason:         "Testing self impersonation",
	}

	ctx := newMockForgeContext("POST", "/api/impersonation/start", reqBody)

	err := handler.StartImpersonation(ctx)

	require.NoError(t, err)
	assert.Equal(t, 400, ctx.GetStatus())

	body := ctx.GetBody().(map[string]string)
	assert.Contains(t, body["error"], "cannot impersonate yourself")
}

func TestHandler_StartImpersonation_AlreadyImpersonating(t *testing.T) {
	handler, _, repo, userSvc := setupTestHandler(t)
	admin, target := createTestUsers(userSvc)
	orgID := xid.New()

	// Create existing active impersonation
	existingSession := &schema.ImpersonationSession{
		ID:             xid.New(),
		OrganizationID: orgID,
		ImpersonatorID: admin.ID,
		TargetUserID:   target.ID,
		Active:         true,
		ExpiresAt:      time.Now().Add(1 * time.Hour),
		CreatedAt:      time.Now(),
	}
	repo.sessions[existingSession.ID.String()] = existingSession

	reqBody := impersonation.StartRequest{
		OrganizationID: orgID,
		ImpersonatorID: admin.ID,
		TargetUserID:   target.ID,
		Reason:         "Testing concurrent impersonations",
	}

	ctx := newMockForgeContext("POST", "/api/impersonation/start", reqBody)

	err := handler.StartImpersonation(ctx)

	require.NoError(t, err)
	assert.Equal(t, 409, ctx.GetStatus())

	body := ctx.GetBody().(map[string]string)
	assert.Contains(t, body["error"], "already impersonating")
}

func TestHandler_EndImpersonation_Success(t *testing.T) {
	handler, _, repo, userSvc := setupTestHandler(t)
	admin, target := createTestUsers(userSvc)
	orgID := xid.New()

	// Create active impersonation
	impSession := &schema.ImpersonationSession{
		ID:             xid.New(),
		OrganizationID: orgID,
		ImpersonatorID: admin.ID,
		TargetUserID:   target.ID,
		Active:         true,
		ExpiresAt:      time.Now().Add(1 * time.Hour),
		Reason:         "Test impersonation",
		CreatedAt:      time.Now(),
	}
	repo.sessions[impSession.ID.String()] = impSession

	reqBody := impersonation.EndRequest{
		ImpersonationID: impSession.ID,
		OrganizationID:  orgID,
		ImpersonatorID:  admin.ID,
		Reason:          "manual",
	}

	ctx := newMockForgeContext("POST", "/api/impersonation/end", reqBody)

	err := handler.EndImpersonation(ctx)

	require.NoError(t, err)
	assert.Equal(t, 200, ctx.GetStatus())

	body := ctx.GetBody().(map[string]interface{})
	assert.True(t, body["success"].(bool))
	assert.NotEmpty(t, body["ended_at"])
}

func TestHandler_GetImpersonation_Success(t *testing.T) {
	handler, _, repo, userSvc := setupTestHandler(t)
	admin, target := createTestUsers(userSvc)
	orgID := xid.New()

	// Create impersonation
	impSession := &schema.ImpersonationSession{
		ID:             xid.New(),
		OrganizationID: orgID,
		ImpersonatorID: admin.ID,
		TargetUserID:   target.ID,
		Active:         true,
		ExpiresAt:      time.Now().Add(1 * time.Hour),
		Reason:         "Test impersonation",
		TicketNumber:   "TICKET-123",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	repo.sessions[impSession.ID.String()] = impSession

	ctx := newMockForgeContext("GET", "/api/impersonation/"+impSession.ID.String(), nil)
	ctx.SetParam("id", impSession.ID.String())
	ctx.request.URL.RawQuery = "org_id=" + orgID.String()

	err := handler.GetImpersonation(ctx)

	require.NoError(t, err)
	assert.Equal(t, 200, ctx.GetStatus())

	body := ctx.GetBody().(*impersonation.SessionInfo)
	assert.Equal(t, impSession.ID, body.ID)
	assert.Equal(t, "Test impersonation", body.Reason)
	assert.Equal(t, "TICKET-123", body.TicketNumber)
	assert.Equal(t, admin.Email, body.ImpersonatorEmail)
	assert.Equal(t, target.Email, body.TargetEmail)
}

func TestHandler_GetImpersonation_InvalidID(t *testing.T) {
	handler, _, _, _ := setupTestHandler(t)

	ctx := newMockForgeContext("GET", "/api/impersonation/invalid", nil)
	ctx.SetParam("id", "invalid-id")

	err := handler.GetImpersonation(ctx)

	require.NoError(t, err)
	assert.Equal(t, 400, ctx.GetStatus())

	body := ctx.GetBody().(map[string]string)
	assert.Equal(t, "Invalid impersonation ID", body["error"])
}

func TestHandler_GetImpersonation_MissingOrgID(t *testing.T) {
	handler, _, _, _ := setupTestHandler(t)
	impID := xid.New()

	ctx := newMockForgeContext("GET", "/api/impersonation/"+impID.String(), nil)
	ctx.SetParam("id", impID.String())
	// No org_id in query params

	err := handler.GetImpersonation(ctx)

	require.NoError(t, err)
	assert.Equal(t, 400, ctx.GetStatus())

	body := ctx.GetBody().(map[string]string)
	assert.Equal(t, "Organization ID is required", body["error"])
}

func TestHandler_ListImpersonations_Success(t *testing.T) {
	handler, _, repo, userSvc := setupTestHandler(t)
	admin, target := createTestUsers(userSvc)
	orgID := xid.New()

	// Create multiple impersonations
	for i := 0; i < 3; i++ {
		session := &schema.ImpersonationSession{
			ID:             xid.New(),
			OrganizationID: orgID,
			ImpersonatorID: admin.ID,
			TargetUserID:   target.ID,
			Active:         true,
			ExpiresAt:      time.Now().Add(1 * time.Hour),
			Reason:         "Test session",
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		repo.sessions[session.ID.String()] = session
	}

	ctx := newMockForgeContext("GET", "/api/impersonation", nil)
	ctx.request.URL.RawQuery = "org_id=" + orgID.String() + "&limit=10&offset=0"

	err := handler.ListImpersonations(ctx)

	require.NoError(t, err)
	assert.Equal(t, 200, ctx.GetStatus())

	body := ctx.GetBody().(*impersonation.ListResponse)
	assert.Len(t, body.Sessions, 3)
	assert.Equal(t, 3, body.Total)
	assert.Equal(t, 10, body.Limit)
	assert.Equal(t, 0, body.Offset)
}

func TestHandler_ListImpersonations_WithFilters(t *testing.T) {
	handler, _, repo, userSvc := setupTestHandler(t)
	admin, target := createTestUsers(userSvc)
	otherUser := &schema.User{
		ID:    xid.New(),
		Email: "other@example.com",
		Name:  "Other User",
	}
	userSvc.users[otherUser.ID.String()] = otherUser
	orgID := xid.New()

	// Create impersonations with different targets
	for i := 0; i < 5; i++ {
		targetID := target.ID
		if i >= 3 {
			targetID = otherUser.ID
		}

		session := &schema.ImpersonationSession{
			ID:             xid.New(),
			OrganizationID: orgID,
			ImpersonatorID: admin.ID,
			TargetUserID:   targetID,
			Active:         true,
			ExpiresAt:      time.Now().Add(1 * time.Hour),
			Reason:         "Test session",
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		repo.sessions[session.ID.String()] = session
	}

	// Filter by target user
	ctx := newMockForgeContext("GET", "/api/impersonation", nil)
	ctx.request.URL.RawQuery = "org_id=" + orgID.String() +
		"&target_user_id=" + target.ID.String() +
		"&limit=10"

	err := handler.ListImpersonations(ctx)

	require.NoError(t, err)
	assert.Equal(t, 200, ctx.GetStatus())

	body := ctx.GetBody().(*impersonation.ListResponse)
	assert.Len(t, body.Sessions, 3) // Only first 3 sessions
	assert.Equal(t, 3, body.Total)
}

func TestHandler_VerifyImpersonation_Active(t *testing.T) {
	handler, _, repo, userSvc := setupTestHandler(t)
	admin, target := createTestUsers(userSvc)
	orgID := xid.New()

	sessionID := xid.New()
	impSession := &schema.ImpersonationSession{
		ID:             xid.New(),
		OrganizationID: orgID,
		ImpersonatorID: admin.ID,
		TargetUserID:   target.ID,
		NewSessionID:   &sessionID,
		Active:         true,
		ExpiresAt:      time.Now().Add(1 * time.Hour),
		CreatedAt:      time.Now(),
	}
	repo.sessions[impSession.ID.String()] = impSession

	reqBody := impersonation.VerifyRequest{
		SessionID: sessionID,
	}

	ctx := newMockForgeContext("POST", "/api/impersonation/verify", reqBody)

	err := handler.VerifyImpersonation(ctx)

	require.NoError(t, err)
	assert.Equal(t, 200, ctx.GetStatus())

	body := ctx.GetBody().(*impersonation.VerifyResponse)
	assert.True(t, body.IsImpersonating)
	assert.NotNil(t, body.ImpersonationID)
	assert.Equal(t, admin.ID, *body.ImpersonatorID)
	assert.Equal(t, target.ID, *body.TargetUserID)
}

func TestHandler_VerifyImpersonation_NotImpersonating(t *testing.T) {
	handler, _, _, _ := setupTestHandler(t)

	reqBody := impersonation.VerifyRequest{
		SessionID: xid.New(), // Non-existent
	}

	ctx := newMockForgeContext("POST", "/api/impersonation/verify", reqBody)

	err := handler.VerifyImpersonation(ctx)

	require.NoError(t, err)
	assert.Equal(t, 200, ctx.GetStatus())

	body := ctx.GetBody().(*impersonation.VerifyResponse)
	assert.False(t, body.IsImpersonating)
	assert.Nil(t, body.ImpersonationID)
}

func TestHandler_ListAuditEvents_Success(t *testing.T) {
	handler, _, repo, userSvc := setupTestHandler(t)
	admin, target := createTestUsers(userSvc)
	orgID := xid.New()
	impID := xid.New()

	// Create audit events
	for i := 0; i < 3; i++ {
		event := &schema.ImpersonationAuditEvent{
			ID:              xid.New(),
			ImpersonationID: impID,
			OrganizationID:  orgID,
			EventType:       "test_event",
			IPAddress:       "192.168.1.1",
			UserAgent:       "Test Agent",
			Details: map[string]string{
				"target_user_id":  target.ID.String(),
				"impersonator_id": admin.ID.String(),
			},
			CreatedAt: time.Now(),
		}
		repo.auditEvents[event.ID.String()] = event
	}

	ctx := newMockForgeContext("GET", "/api/impersonation/audit", nil)
	ctx.request.URL.RawQuery = "org_id=" + orgID.String() + "&limit=10&offset=0"

	err := handler.ListAuditEvents(ctx)

	require.NoError(t, err)
	assert.Equal(t, 200, ctx.GetStatus())

	body := ctx.GetBody().(map[string]interface{})
	events := body["events"].([]*impersonation.AuditEvent)
	assert.Len(t, events, 3)
	assert.Equal(t, 3, body["total"])
}

func TestHandler_ListAuditEvents_FilterByImpersonationID(t *testing.T) {
	handler, _, repo, _ := setupTestHandler(t)
	orgID := xid.New()
	impID1 := xid.New()
	impID2 := xid.New()

	// Create events for two different impersonations
	for i := 0; i < 5; i++ {
		impID := impID1
		if i >= 3 {
			impID = impID2
		}

		event := &schema.ImpersonationAuditEvent{
			ID:              xid.New(),
			ImpersonationID: impID,
			OrganizationID:  orgID,
			EventType:       "test_event",
			CreatedAt:       time.Now(),
		}
		repo.auditEvents[event.ID.String()] = event
	}

	ctx := newMockForgeContext("GET", "/api/impersonation/audit", nil)
	ctx.request.URL.RawQuery = "org_id=" + orgID.String() +
		"&impersonation_id=" + impID1.String() +
		"&limit=10"

	err := handler.ListAuditEvents(ctx)

	require.NoError(t, err)
	assert.Equal(t, 200, ctx.GetStatus())

	body := ctx.GetBody().(map[string]interface{})
	events := body["events"].([]*impersonation.AuditEvent)
	assert.Len(t, events, 3) // Only first 3 events
}

func TestHandler_PaginationDefaults(t *testing.T) {
	handler, _, _, _ := setupTestHandler(t)
	orgID := xid.New()

	ctx := newMockForgeContext("GET", "/api/impersonation", nil)
	ctx.request.URL.RawQuery = "org_id=" + orgID.String()
	// No limit/offset specified

	err := handler.ListImpersonations(ctx)

	require.NoError(t, err)
	assert.Equal(t, 200, ctx.GetStatus())

	body := ctx.GetBody().(*impersonation.ListResponse)
	assert.Equal(t, 20, body.Limit) // Default limit
	assert.Equal(t, 0, body.Offset) // Default offset
}
