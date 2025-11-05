package impersonation

import (
	"context"
	"testing"
	"time"

	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/authsome/core/impersonation"
	"github.com/xraph/authsome/schema"
)

// TestIntegration_CompleteFlow tests the complete impersonation lifecycle
func TestIntegration_CompleteFlow(t *testing.T) {
	service, repo, userSvc, _ := setupTestService(t)
	admin, target := createTestUsers(userSvc)
	orgID := xid.New()
	ctx := context.Background()

	// Step 1: Start impersonation
	t.Run("Start", func(t *testing.T) {
		startReq := &impersonation.StartRequest{
			OrganizationID:  orgID,
			ImpersonatorID:  admin.ID,
			TargetUserID:    target.ID,
			Reason:          "Customer reported dashboard issue - investigating TICKET-12345",
			TicketNumber:    "TICKET-12345",
			DurationMinutes: 60,
			IPAddress:       "192.168.1.100",
			UserAgent:       "Mozilla/5.0",
		}

		startResp, err := service.Start(ctx, startReq)

		require.NoError(t, err)
		require.NotNil(t, startResp)
		assert.NotEmpty(t, startResp.ImpersonationID)
		assert.NotEmpty(t, startResp.SessionID)
		assert.NotEmpty(t, startResp.SessionToken)
		assert.False(t, startResp.ExpiresAt.IsZero())
		assert.WithinDuration(t, time.Now().Add(60*time.Minute), startResp.ExpiresAt, 1*time.Minute)

		// Verify impersonation was created
		session, err := repo.Get(ctx, startResp.ImpersonationID, orgID)
		require.NoError(t, err)
		assert.True(t, session.Active)
		assert.Equal(t, admin.ID, session.ImpersonatorID)
		assert.Equal(t, target.ID, session.TargetUserID)
		assert.Equal(t, "TICKET-12345", session.TicketNumber)
	})

	// Step 2: Verify impersonation is active
	t.Run("Verify", func(t *testing.T) {
		// Get the impersonation session
		listReq := &impersonation.ListRequest{
			OrganizationID: orgID,
			ActiveOnly:     true,
			Limit:          10,
		}

		listResp, err := service.List(ctx, listReq)
		require.NoError(t, err)
		require.Len(t, listResp.Sessions, 1)

		impSession := listResp.Sessions[0]

		// Verify the session
		verifyReq := &impersonation.VerifyRequest{
			SessionID: *repo.sessions[impSession.ID.String()].NewSessionID,
		}

		verifyResp, err := service.Verify(ctx, verifyReq)

		require.NoError(t, err)
		assert.True(t, verifyResp.IsImpersonating)
		assert.NotNil(t, verifyResp.ImpersonationID)
		assert.NotNil(t, verifyResp.ImpersonatorID)
		assert.NotNil(t, verifyResp.TargetUserID)
	})

	// Step 3: List active impersonations
	t.Run("List", func(t *testing.T) {
		listReq := &impersonation.ListRequest{
			OrganizationID: orgID,
			ActiveOnly:     true,
			Limit:          10,
		}

		listResp, err := service.List(ctx, listReq)

		require.NoError(t, err)
		assert.Len(t, listResp.Sessions, 1)
		assert.Equal(t, 1, listResp.Total)

		session := listResp.Sessions[0]
		assert.Equal(t, admin.Email, session.ImpersonatorEmail)
		assert.Equal(t, target.Email, session.TargetEmail)
		assert.True(t, session.Active)
	})

	// Step 4: Get specific impersonation
	t.Run("Get", func(t *testing.T) {
		// Get ID from list
		listReq := &impersonation.ListRequest{
			OrganizationID: orgID,
			ActiveOnly:     true,
			Limit:          1,
		}
		listResp, _ := service.List(ctx, listReq)
		impID := listResp.Sessions[0].ID

		getReq := &impersonation.GetRequest{
			ImpersonationID: impID,
			OrganizationID:  orgID,
		}

		getResp, err := service.Get(ctx, getReq)

		require.NoError(t, err)
		assert.Equal(t, impID, getResp.ID)
		assert.Equal(t, "TICKET-12345", getResp.TicketNumber)
		assert.Contains(t, getResp.Reason, "Customer reported")
	})

	// Step 5: End impersonation
	t.Run("End", func(t *testing.T) {
		// Get ID from list
		listReq := &impersonation.ListRequest{
			OrganizationID: orgID,
			ActiveOnly:     true,
			Limit:          1,
		}
		listResp, _ := service.List(ctx, listReq)
		impID := listResp.Sessions[0].ID

		endReq := &impersonation.EndRequest{
			ImpersonationID: impID,
			OrganizationID:  orgID,
			ImpersonatorID:  admin.ID,
			Reason:          "manual",
		}

		endResp, err := service.End(ctx, endReq)

		require.NoError(t, err)
		assert.True(t, endResp.Success)
		assert.Equal(t, impID, endResp.ImpersonationID)
		assert.False(t, endResp.EndedAt.IsZero())

		// Verify impersonation is no longer active
		listReq.ActiveOnly = true
		listResp, _ = service.List(ctx, listReq)
		assert.Len(t, listResp.Sessions, 0) // No active sessions
	})

	// Step 6: Verify audit trail exists
	t.Run("Audit", func(t *testing.T) {
		auditReq := &impersonation.AuditListRequest{
			OrganizationID: orgID,
			Limit:          10,
		}

		events, total, err := service.ListAuditEvents(ctx, auditReq)

		require.NoError(t, err)
		assert.Greater(t, total, 0)
		assert.NotEmpty(t, events)

		// Should have at least "started" and "ended" events
		eventTypes := make(map[string]bool)
		for _, event := range events {
			eventTypes[event.EventType] = true
		}
		assert.True(t, eventTypes["started"])
		assert.True(t, eventTypes["ended"])
	})
}

// TestIntegration_MultipleOrganizations tests multi-tenant isolation
func TestIntegration_MultipleOrganizations(t *testing.T) {
	service, _, userSvc, _ := setupTestService(t)
	ctx := context.Background()

	// Create users in different orgs
	admin1, target1 := createTestUsers(userSvc)
	admin2 := &schema.User{
		ID:    xid.New(),
		Email: "admin2@example.com",
		Name:  "Admin 2",
	}
	target2 := &schema.User{
		ID:    xid.New(),
		Email: "target2@example.com",
		Name:  "Target 2",
	}
	userSvc.users[admin2.ID.String()] = admin2
	userSvc.users[target2.ID.String()] = target2

	org1 := xid.New()
	org2 := xid.New()

	// Start impersonation in org1
	startReq1 := &impersonation.StartRequest{
		OrganizationID: org1,
		ImpersonatorID: admin1.ID,
		TargetUserID:   target1.ID,
		Reason:         "Testing org1 impersonation",
	}
	startResp1, err := service.Start(ctx, startReq1)
	require.NoError(t, err)

	// Start impersonation in org2
	startReq2 := &impersonation.StartRequest{
		OrganizationID: org2,
		ImpersonatorID: admin2.ID,
		TargetUserID:   target2.ID,
		Reason:         "Testing org2 impersonation",
	}
	startResp2, err := service.Start(ctx, startReq2)
	require.NoError(t, err)

	// List org1 sessions - should only see org1
	listReq1 := &impersonation.ListRequest{
		OrganizationID: org1,
		ActiveOnly:     true,
		Limit:          10,
	}
	listResp1, err := service.List(ctx, listReq1)
	require.NoError(t, err)
	assert.Len(t, listResp1.Sessions, 1)
	assert.Equal(t, org1, listResp1.Sessions[0].OrganizationID)
	assert.Equal(t, admin1.ID, listResp1.Sessions[0].ImpersonatorID)

	// List org2 sessions - should only see org2
	listReq2 := &impersonation.ListRequest{
		OrganizationID: org2,
		ActiveOnly:     true,
		Limit:          10,
	}
	listResp2, err := service.List(ctx, listReq2)
	require.NoError(t, err)
	assert.Len(t, listResp2.Sessions, 1)
	assert.Equal(t, org2, listResp2.Sessions[0].OrganizationID)
	assert.Equal(t, admin2.ID, listResp2.Sessions[0].ImpersonatorID)

	// Try to get org1 session with org2 context - should fail
	getReq := &impersonation.GetRequest{
		ImpersonationID: startResp1.ImpersonationID,
		OrganizationID:  org2, // Wrong org
	}
	_, err = service.Get(ctx, getReq)
	assert.Error(t, err)
	assert.Equal(t, impersonation.ErrImpersonationNotFound, err)

	// Try to end org1 session with org2 admin - should fail
	endReq := &impersonation.EndRequest{
		ImpersonationID: startResp1.ImpersonationID,
		OrganizationID:  org1,
		ImpersonatorID:  admin2.ID, // Admin from different org
	}
	_, err = service.End(ctx, endReq)
	assert.Error(t, err)
	assert.Equal(t, impersonation.ErrPermissionDenied, err)
}

// TestIntegration_ConcurrentImpersonations tests multiple admins impersonating simultaneously
func TestIntegration_ConcurrentImpersonations(t *testing.T) {
	service, _, userSvc, _ := setupTestService(t)
	ctx := context.Background()
	orgID := xid.New()

	// Create multiple admins and targets
	admins := make([]*schema.User, 3)
	targets := make([]*schema.User, 3)
	for i := 0; i < 3; i++ {
		admin := &schema.User{
			ID:    xid.New(),
			Email: "admin" + string(rune('a'+i)) + "@example.com",
			Name:  "Admin " + string(rune('A'+i)),
		}
		target := &schema.User{
			ID:    xid.New(),
			Email: "target" + string(rune('a'+i)) + "@example.com",
			Name:  "Target " + string(rune('A'+i)),
		}
		userSvc.users[admin.ID.String()] = admin
		userSvc.users[target.ID.String()] = target
		admins[i] = admin
		targets[i] = target
	}

	// Start impersonations for all admins
	for i := 0; i < 3; i++ {
		startReq := &impersonation.StartRequest{
			OrganizationID: orgID,
			ImpersonatorID: admins[i].ID,
			TargetUserID:   targets[i].ID,
			Reason:         "Concurrent testing impersonation " + string(rune('A'+i)),
		}
		_, err := service.Start(ctx, startReq)
		require.NoError(t, err)
	}

	// List all active sessions
	listReq := &impersonation.ListRequest{
		OrganizationID: orgID,
		ActiveOnly:     true,
		Limit:          10,
	}
	listResp, err := service.List(ctx, listReq)
	require.NoError(t, err)
	assert.Len(t, listResp.Sessions, 3)

	// Verify each admin can only end their own session
	for i := 0; i < 3; i++ {
		// Find this admin's session
		var sessionID xid.ID
		for _, session := range listResp.Sessions {
			if session.ImpersonatorID == admins[i].ID {
				sessionID = session.ID
				break
			}
		}

		// Try to end with wrong admin - should fail
		wrongAdmin := admins[(i+1)%3]
		endReq := &impersonation.EndRequest{
			ImpersonationID: sessionID,
			OrganizationID:  orgID,
			ImpersonatorID:  wrongAdmin.ID,
		}
		_, err := service.End(ctx, endReq)
		assert.Error(t, err)
		assert.Equal(t, impersonation.ErrPermissionDenied, err)

		// End with correct admin - should succeed
		endReq.ImpersonatorID = admins[i].ID
		_, err = service.End(ctx, endReq)
		require.NoError(t, err)
	}

	// Verify all sessions are ended
	listResp, _ = service.List(ctx, listReq)
	assert.Len(t, listResp.Sessions, 0)
}

// TestIntegration_AutoExpiry tests automatic session expiration
func TestIntegration_AutoExpiry(t *testing.T) {
	service, repo, userSvc, _ := setupTestService(t)
	admin, target := createTestUsers(userSvc)
	orgID := xid.New()
	ctx := context.Background()

	// Create expired sessions
	for i := 0; i < 3; i++ {
		session := &schema.ImpersonationSession{
			ID:             xid.New(),
			OrganizationID: orgID,
			ImpersonatorID: admin.ID,
			TargetUserID:   target.ID,
			Active:         true,
			ExpiresAt:      time.Now().Add(-1 * time.Hour), // Already expired
			Reason:         "Test expired session",
			CreatedAt:      time.Now(),
		}
		repo.sessions[session.ID.String()] = session
	}

	// Create active non-expired session
	activeSession := &schema.ImpersonationSession{
		ID:             xid.New(),
		OrganizationID: orgID,
		ImpersonatorID: admin.ID,
		TargetUserID:   target.ID,
		Active:         true,
		ExpiresAt:      time.Now().Add(1 * time.Hour), // Still valid
		Reason:         "Test active session",
		CreatedAt:      time.Now(),
	}
	repo.sessions[activeSession.ID.String()] = activeSession

	// Before cleanup - should see 4 sessions (3 expired + 1 active)
	listReq := &impersonation.ListRequest{
		OrganizationID: orgID,
		ActiveOnly:     false,
		Limit:          10,
	}
	listResp, _ := service.List(ctx, listReq)
	assert.Len(t, listResp.Sessions, 4)

	// Run cleanup
	count, err := service.ExpireSessions(ctx)
	require.NoError(t, err)
	assert.Equal(t, 3, count)

	// After cleanup - active session should still be active
	listReq.ActiveOnly = true
	listResp, _ = service.List(ctx, listReq)
	assert.Len(t, listResp.Sessions, 1)
	assert.Equal(t, activeSession.ID, listResp.Sessions[0].ID)

	// Expired sessions should be marked as ended
	listReq.ActiveOnly = false
	listResp, _ = service.List(ctx, listReq)
	endedCount := 0
	for _, session := range listResp.Sessions {
		if !session.Active && session.EndedAt != nil && session.EndReason == "timeout" {
			endedCount++
		}
	}
	assert.Equal(t, 3, endedCount)
}

// TestIntegration_FilterByUser tests filtering impersonations by specific users
func TestIntegration_FilterByUser(t *testing.T) {
	service, _, userSvc, _ := setupTestService(t)
	ctx := context.Background()
	orgID := xid.New()

	// Create multiple admins and targets
	admin1 := &schema.User{ID: xid.New(), Email: "admin1@example.com", Name: "Admin 1"}
	admin2 := &schema.User{ID: xid.New(), Email: "admin2@example.com", Name: "Admin 2"}
	target1 := &schema.User{ID: xid.New(), Email: "target1@example.com", Name: "Target 1"}
	target2 := &schema.User{ID: xid.New(), Email: "target2@example.com", Name: "Target 2"}

	for _, user := range []*schema.User{admin1, admin2, target1, target2} {
		userSvc.users[user.ID.String()] = user
	}

	// Create various impersonation combinations
	combinations := []struct {
		admin  *schema.User
		target *schema.User
	}{
		{admin1, target1},
		{admin1, target2},
		{admin2, target1},
		{admin2, target2},
	}

	for _, combo := range combinations {
		startReq := &impersonation.StartRequest{
			OrganizationID: orgID,
			ImpersonatorID: combo.admin.ID,
			TargetUserID:   combo.target.ID,
			Reason:         "Testing filter combinations",
		}
		_, err := service.Start(ctx, startReq)
		require.NoError(t, err)
	}

	// Filter by impersonator (admin1)
	listReq := &impersonation.ListRequest{
		OrganizationID: orgID,
		ImpersonatorID: &admin1.ID,
		ActiveOnly:     true,
		Limit:          10,
	}
	listResp, err := service.List(ctx, listReq)
	require.NoError(t, err)
	assert.Len(t, listResp.Sessions, 2) // admin1 -> target1 and admin1 -> target2

	// Filter by target (target1)
	listReq = &impersonation.ListRequest{
		OrganizationID: orgID,
		TargetUserID:   &target1.ID,
		ActiveOnly:     true,
		Limit:          10,
	}
	listResp, err = service.List(ctx, listReq)
	require.NoError(t, err)
	assert.Len(t, listResp.Sessions, 2) // admin1 -> target1 and admin2 -> target1

	// Filter by both impersonator and target
	listReq = &impersonation.ListRequest{
		OrganizationID: orgID,
		ImpersonatorID: &admin1.ID,
		TargetUserID:   &target1.ID,
		ActiveOnly:     true,
		Limit:          10,
	}
	listResp, err = service.List(ctx, listReq)
	require.NoError(t, err)
	assert.Len(t, listResp.Sessions, 1) // Only admin1 -> target1
	assert.Equal(t, admin1.ID, listResp.Sessions[0].ImpersonatorID)
	assert.Equal(t, target1.ID, listResp.Sessions[0].TargetUserID)
}
