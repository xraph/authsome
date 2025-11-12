# Organization Plugin

The Organization plugin provides Clerk.js-style user-created organizations (workspaces) for AuthSome. It allows authenticated users to create their own organizations, invite members, manage teams, and control access through role-based permissions.

## Features

- **User-Created Organizations**: Any authenticated user can create their own organization
- **Member Management**: Invite users, manage roles (owner, admin, member)
- **Team Management**: Create teams within organizations for better structure
- **Role-Based Access Control**: Owner, Admin, and Member roles with appropriate permissions
- **Invitation System**: Secure token-based invitations with expiry
- **Multi-Organization Support**: Users can belong to multiple organizations
- **Slug-Based Access**: Access organizations via unique slugs

## Architecture

This plugin is separate from the platform-level **App** (formerly Organization in multitenancy plugin). The hierarchy is:

```
Platform App (managed by multitenancy plugin)
└── User Organizations (managed by this plugin)
    ├── Members (users with roles)
    └── Teams (groups of members)
```

## Models

### Organization
User-created workspace/organization that belongs to a platform App.

```go
type Organization struct {
    ID        xid.ID
    AppID     xid.ID  // Platform app this org belongs to
    Name      string
    Slug      string  // Unique slug for URL access
    Logo      *string
    Metadata  map[string]interface{}
    CreatedBy xid.ID  // User who created it
    // ... timestamps
}
```

### OrganizationMember
Represents a user's membership in an organization.

```go
type OrganizationMember struct {
    ID             xid.ID
    OrganizationID xid.ID
    UserID         xid.ID
    Role           string  // owner, admin, member
    Status         string  // active, suspended, pending
    // ... timestamps
}
```

### OrganizationTeam
Teams within an organization for better member organization.

```go
type OrganizationTeam struct {
    ID             xid.ID
    OrganizationID xid.ID
    Name           string
    Description    *string
    Metadata       map[string]interface{}
    // ... timestamps
}
```

### OrganizationInvitation
Invitation to join an organization.

```go
type OrganizationInvitation struct {
    ID             xid.ID
    OrganizationID xid.ID
    Email          string
    Role           string
    Token          string  // Secure invitation token
    Status         string  // pending, accepted, declined, expired
    InvitedBy      xid.ID
    ExpiresAt      time.Time
    // ... timestamps
}
```

## Roles

### Owner
- Full control over the organization
- Can delete the organization
- Can manage all members and teams
- Cannot be removed or have role changed
- Assigned to creator automatically

### Admin
- Can invite and manage members
- Can create and manage teams
- Can update organization settings
- Cannot delete organization

### Member
- Can view organization
- Can view members and teams
- Limited modification rights

## API Endpoints

### Organization Management

```
POST   /api/organizations                          # Create organization
GET    /api/organizations                          # List user's organizations
GET    /api/organizations/:id                      # Get organization details
GET    /api/organizations/slug/:slug               # Get organization by slug
PATCH  /api/organizations/:id                      # Update organization
DELETE /api/organizations/:id                      # Delete organization (owner only)
```

### Member Management

```
GET    /api/organizations/:id/members              # List members
POST   /api/organizations/:id/members/invite       # Invite member
PATCH  /api/organizations/:id/members/:memberId   # Update member role
DELETE /api/organizations/:id/members/:memberId   # Remove member
```

### Team Management

```
GET    /api/organizations/:id/teams                # List teams
POST   /api/organizations/:id/teams                # Create team
PATCH  /api/organizations/:id/teams/:teamId       # Update team
DELETE /api/organizations/:id/teams/:teamId       # Delete team
```

### Invitations

```
POST   /api/organization-invitations/:token/accept    # Accept invitation
POST   /api/organization-invitations/:token/decline   # Decline invitation
```

## Configuration

```yaml
auth:
  organization:
    maxOrganizationsPerUser: 5       # Max orgs a user can create
    maxMembersPerOrganization: 50    # Max members per org
    maxTeamsPerOrganization: 20      # Max teams per org
    enableUserCreation: true         # Allow users to create orgs
    requireInvitation: true          # Require invitation to join
    invitationExpiryHours: 72        # Invitation validity (3 days)
```

## Usage

### Installation

```go
import (
    "github.com/xraph/authsome"
    "github.com/xraph/authsome/plugins/organization"
)

auth := authsome.New(
    // ... other options
    authsome.WithPlugins(
        organization.NewPlugin(
            organization.WithMaxOrganizationsPerUser(10),
            organization.WithMaxMembersPerOrganization(100),
            organization.WithEnableUserCreation(true),
        ),
    ),
)
```

### Creating an Organization

```bash
POST /api/organizations
Content-Type: application/json

{
  "name": "Acme Corporation",
  "slug": "acme",
  "logo": "https://example.com/logo.png",
  "metadata": {
    "industry": "Technology"
  }
}
```

### Inviting a Member

```bash
POST /api/organizations/:orgId/members/invite
Content-Type: application/json

{
  "email": "user@example.com",
  "role": "member"
}
```

### Accepting an Invitation

```bash
POST /api/organization-invitations/:token/accept
```

## Permissions

| Action | Owner | Admin | Member |
|--------|-------|-------|--------|
| View organization | ✅ | ✅ | ✅ |
| Update organization | ✅ | ✅ | ❌ |
| Delete organization | ✅ | ❌ | ❌ |
| Invite members | ✅ | ✅ | ❌ |
| Remove members | ✅ | ✅ | ❌ |
| Update member roles | ✅ | ✅ | ❌ |
| Create teams | ✅ | ✅ | ✅ |
| Manage teams | ✅ | ✅ | ❌ |

## Integration with Other Plugins

### SCIM Plugin
The SCIM plugin will be updated to provision users into organizations:
```
/scim/v2/organizations/:orgId/Users
/scim/v2/organizations/:orgId/Groups
```

### Subscription Plugin (Future)
Organizations can be linked to subscriptions for billing:
```go
type OrganizationSubscription struct {
    OrganizationID xid.ID
    PlanID         string
    Status         string
    // ...
}
```

## Database Tables

- `user_organizations` - Organization entities
- `user_organization_members` - Member relationships
- `user_organization_teams` - Team entities
- `user_organization_team_members` - Team-member relationships (many-to-many)
- `user_organization_invitations` - Invitation tokens

## Differences from Platform Apps

| Feature | Platform App (Multitenancy) | User Organization |
|---------|----------------------------|-------------------|
| Created by | Platform admin | Any authenticated user |
| Scope | Platform-wide tenant | User workspace |
| Members | All platform users | Invited users only |
| SCIM | Platform-level provisioning | Org-scoped provisioning |
| Billing | Platform subscription | Per-org subscription |

## TODO

- [ ] Implement repository layer
- [ ] Add middleware for organization context
- [ ] Integrate with user service for user ID extraction
- [ ] Add organization switcher API
- [ ] Add organization statistics/metrics
- [ ] Add audit log for organization actions
- [ ] Add webhooks for organization events
- [ ] Add organization settings/preferences

## License

Same as AuthSome project.

