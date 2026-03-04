package rbac

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/xraph/authsome/id"
)

func TestRole_Fields(t *testing.T) {
	r := &Role{
		ID:          id.NewRoleID().String(),
		AppID:       id.NewAppID().String(),
		Name:        "Admin",
		Slug:        "admin",
		Description: "Full access role",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	assert.NotEmpty(t, r.ID)
	assert.Equal(t, "Admin", r.Name)
	assert.Equal(t, "admin", r.Slug)
	assert.Equal(t, "Full access role", r.Description)
}

func TestPermission_Fields(t *testing.T) {
	p := &Permission{
		ID:       id.NewPermissionID().String(),
		RoleID:   id.NewRoleID().String(),
		Action:   "read",
		Resource: "user",
	}
	assert.NotEmpty(t, p.ID)
	assert.Equal(t, "read", p.Action)
	assert.Equal(t, "user", p.Resource)
}

func TestUserRole_Fields(t *testing.T) {
	ur := &UserRole{
		UserID:     id.NewUserID().String(),
		RoleID:     id.NewRoleID().String(),
		OrgID:      "",
		AssignedAt: time.Now(),
	}
	assert.NotEmpty(t, ur.UserID)
	assert.NotEmpty(t, ur.RoleID)
	assert.Empty(t, ur.OrgID)
}

func TestUserRole_WithOrgScope(t *testing.T) {
	orgID := id.NewOrgID().String()
	ur := &UserRole{
		UserID:     id.NewUserID().String(),
		RoleID:     id.NewRoleID().String(),
		OrgID:      orgID,
		AssignedAt: time.Now(),
	}
	assert.NotEmpty(t, ur.OrgID)
}
