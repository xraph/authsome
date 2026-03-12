package organization

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/xraph/authsome/id"
)

// ──────────────────────────────────────────────────
// Organization struct tests
// ──────────────────────────────────────────────────

func TestOrganization_FieldsPopulated(t *testing.T) {
	o := &Organization{
		ID:        id.NewOrgID(),
		AppID:     id.NewAppID(),
		Name:      "Test Org",
		Slug:      "test-org",
		CreatedBy: id.NewUserID(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	assert.NotEmpty(t, o.ID.String())
	assert.NotEmpty(t, o.AppID.String())
	assert.NotEmpty(t, o.CreatedBy.String())
	assert.Equal(t, "Test Org", o.Name)
	assert.Equal(t, "test-org", o.Slug)
	assert.False(t, o.CreatedAt.IsZero())
	assert.False(t, o.UpdatedAt.IsZero())
}

func TestOrganization_Metadata(t *testing.T) {
	m := Metadata{"key": "value", "foo": "bar"}
	assert.Equal(t, "value", m["key"])
	assert.Equal(t, "bar", m["foo"])
}

// ──────────────────────────────────────────────────
// Member role constants
// ──────────────────────────────────────────────────

func TestMemberRoleConstants(t *testing.T) {
	assert.Equal(t, MemberRole("owner"), RoleOwner)
	assert.Equal(t, MemberRole("admin"), RoleAdmin)
	assert.Equal(t, MemberRole("member"), RoleMember)
}

func TestMember_FieldsPopulated(t *testing.T) {
	m := &Member{
		ID:        id.NewMemberID(),
		OrgID:     id.NewOrgID(),
		UserID:    id.NewUserID(),
		Role:      RoleMember,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	assert.NotEmpty(t, m.ID.String())
	assert.NotEmpty(t, m.OrgID.String())
	assert.NotEmpty(t, m.UserID.String())
	assert.Equal(t, RoleMember, m.Role)
	assert.False(t, m.CreatedAt.IsZero())
	assert.False(t, m.UpdatedAt.IsZero())
}

// ──────────────────────────────────────────────────
// Invitation status constants
// ──────────────────────────────────────────────────

func TestInvitationStatusConstants(t *testing.T) {
	assert.Equal(t, InvitationStatus("pending"), InvitationPending)
	assert.Equal(t, InvitationStatus("accepted"), InvitationAccepted)
	assert.Equal(t, InvitationStatus("expired"), InvitationExpired)
	assert.Equal(t, InvitationStatus("declined"), InvitationDeclined)
}

func TestInvitation_FieldsPopulated(t *testing.T) {
	inv := &Invitation{
		ID:        id.NewInvitationID(),
		OrgID:     id.NewOrgID(),
		Email:     "test@example.com",
		Role:      RoleMember,
		InviterID: id.NewUserID(),
		Status:    InvitationPending,
		Token:     "some-token",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
	}
	assert.NotEmpty(t, inv.ID.String())
	assert.NotEmpty(t, inv.OrgID.String())
	assert.NotEmpty(t, inv.InviterID.String())
	assert.Equal(t, InvitationPending, inv.Status)
	assert.Equal(t, "test@example.com", inv.Email)
	assert.Equal(t, RoleMember, inv.Role)
	assert.Equal(t, "some-token", inv.Token)
	assert.False(t, inv.ExpiresAt.IsZero())
	assert.False(t, inv.CreatedAt.IsZero())
}

// ──────────────────────────────────────────────────
// Team struct tests
// ──────────────────────────────────────────────────

func TestTeam_FieldsPopulated(t *testing.T) {
	tm := &Team{
		ID:        id.NewTeamID(),
		OrgID:     id.NewOrgID(),
		Name:      "Engineering",
		Slug:      "engineering",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	assert.NotEmpty(t, tm.ID.String())
	assert.NotEmpty(t, tm.OrgID.String())
	assert.Equal(t, "Engineering", tm.Name)
	assert.Equal(t, "engineering", tm.Slug)
	assert.False(t, tm.CreatedAt.IsZero())
	assert.False(t, tm.UpdatedAt.IsZero())
}
