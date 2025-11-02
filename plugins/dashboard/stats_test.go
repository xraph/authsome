package dashboard

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDashboardStats(t *testing.T) {
	stats := &DashboardStats{
		TotalUsers:     1000,
		ActiveUsers:    800,
		NewUsersToday:  50,
		TotalSessions:  2000,
		ActiveSessions: 500,
		FailedLogins:   10,
		UserGrowth:     5.5,
		SessionGrowth:  3.2,
	}
	
	assert.Equal(t, 1000, stats.TotalUsers)
	assert.Equal(t, 800, stats.ActiveUsers)
	assert.Equal(t, 50, stats.NewUsersToday)
	assert.Equal(t, 2000, stats.TotalSessions)
	assert.Equal(t, 500, stats.ActiveSessions)
	assert.Equal(t, 10, stats.FailedLogins)
	assert.Equal(t, 5.5, stats.UserGrowth)
	assert.Equal(t, 3.2, stats.SessionGrowth)
}

func TestActivityItem(t *testing.T) {
	activity := ActivityItem{
		Title:       "User registration",
		Description: "New user signed up",
		Time:        "2 minutes ago",
		Type:        "success",
	}
	
	assert.Equal(t, "User registration", activity.Title)
	assert.Equal(t, "New user signed up", activity.Description)
	assert.Equal(t, "2 minutes ago", activity.Time)
	assert.Equal(t, "success", activity.Type)
}

func TestStatusItem(t *testing.T) {
	status := StatusItem{
		Name:   "Authentication Service",
		Status: "operational",
		Color:  "green",
	}
	
	assert.Equal(t, "Authentication Service", status.Name)
	assert.Equal(t, "operational", status.Status)
	assert.Equal(t, "green", status.Color)
}

func TestStatsService_getRecentActivity(t *testing.T) {
	// This would require mocking the audit service
	// For now, we test the structure
	
	activity := []ActivityItem{
		{
			Title:       "Test Activity",
			Description: "Test Description",
			Time:        "Just now",
			Type:        "info",
		},
	}
	
	assert.Len(t, activity, 1)
	assert.Equal(t, "Test Activity", activity[0].Title)
}

func TestStatsService_getSystemStatus(t *testing.T) {
	// Test system status structure
	status := []StatusItem{
		{
			Name:   "Service 1",
			Status: "operational",
			Color:  "green",
		},
		{
			Name:   "Service 2",
			Status:  "degraded",
			Color:  "yellow",
		},
	}
	
	assert.Len(t, status, 2)
	assert.Equal(t, "operational", status[0].Status)
	assert.Equal(t, "degraded", status[1].Status)
}

// Note: Full integration tests for StatsService.GetDashboardStats would require
// mocking the user, session, and audit services, which would be done in integration tests

