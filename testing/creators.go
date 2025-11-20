package testing

import (
	"github.com/rs/xid"
	"github.com/xraph/authsome/schema"
)

// AppBuilder helps create test apps with fluent API
type AppBuilder struct {
	mock *Mock
	app  *schema.App
}

// NewApp creates a new AppBuilder
func (m *Mock) NewApp() *AppBuilder {
	return &AppBuilder{
		mock: m,
		app: &schema.App{
			ID:   xid.New(),
			Name: "Test App",
			Slug: "test-app",
		},
	}
}

// WithName sets the app name
func (b *AppBuilder) WithName(name string) *AppBuilder {
	b.app.Name = name
	return b
}

// WithSlug sets the app slug
func (b *AppBuilder) WithSlug(slug string) *AppBuilder {
	b.app.Slug = slug
	return b
}

// Build saves the app and returns it
func (b *AppBuilder) Build() *schema.App {
	b.mock.mu.Lock()
	defer b.mock.mu.Unlock()
	b.mock.apps[b.app.ID] = b.app
	return b.app
}

// EnvironmentBuilder helps create test environments with fluent API
type EnvironmentBuilder struct {
	mock *Mock
	env  *schema.Environment
}

// NewEnvironment creates a new EnvironmentBuilder
func (m *Mock) NewEnvironment(appID xid.ID) *EnvironmentBuilder {
	return &EnvironmentBuilder{
		mock: m,
		env: &schema.Environment{
			ID:    xid.New(),
			AppID: appID,
			Name:  "test",
			Slug:  "test",
		},
	}
}

// WithName sets the environment name
func (b *EnvironmentBuilder) WithName(name string) *EnvironmentBuilder {
	b.env.Name = name
	return b
}

// WithSlug sets the environment slug
func (b *EnvironmentBuilder) WithSlug(slug string) *EnvironmentBuilder {
	b.env.Slug = slug
	return b
}

// Build saves the environment and returns it
func (b *EnvironmentBuilder) Build() *schema.Environment {
	b.mock.mu.Lock()
	defer b.mock.mu.Unlock()
	b.mock.environments[b.env.ID] = b.env
	return b.env
}

// OrganizationBuilder helps create test organizations with fluent API
type OrganizationBuilder struct {
	mock *Mock
	org  *schema.Organization
}

// NewOrganization creates a new OrganizationBuilder
func (m *Mock) NewOrganization() *OrganizationBuilder {
	return &OrganizationBuilder{
		mock: m,
		org: &schema.Organization{
			ID:            xid.New(),
			AppID:         m.defaultApp.ID,
			EnvironmentID: m.defaultEnv.ID,
			Name:          "Test Organization",
			Slug:          "test-org",
			Metadata:      map[string]interface{}{},
		},
	}
}

// WithName sets the organization name
func (b *OrganizationBuilder) WithName(name string) *OrganizationBuilder {
	b.org.Name = name
	return b
}

// WithSlug sets the organization slug
func (b *OrganizationBuilder) WithSlug(slug string) *OrganizationBuilder {
	b.org.Slug = slug
	return b
}

// WithApp sets the app ID
func (b *OrganizationBuilder) WithApp(appID xid.ID) *OrganizationBuilder {
	b.org.AppID = appID
	return b
}

// WithEnvironment sets the environment ID
func (b *OrganizationBuilder) WithEnvironment(envID xid.ID) *OrganizationBuilder {
	b.org.EnvironmentID = envID
	return b
}

// WithMetadata sets metadata
func (b *OrganizationBuilder) WithMetadata(metadata map[string]interface{}) *OrganizationBuilder {
	b.org.Metadata = metadata
	return b
}

// Build saves the organization and returns it
func (b *OrganizationBuilder) Build() *schema.Organization {
	b.mock.mu.Lock()
	defer b.mock.mu.Unlock()
	b.mock.orgs[b.org.ID] = b.org
	return b.org
}

// UserBuilder helps create test users with fluent API
type UserBuilder struct {
	mock *Mock
	user *schema.User
	role string
}

// NewUser creates a new UserBuilder
func (m *Mock) NewUser() *UserBuilder {
	return &UserBuilder{
		mock: m,
		user: &schema.User{
			ID:            xid.New(),
			Email:         "user@example.com",
			Name:          "Test User",
			EmailVerified: true,
		},
		role: "member",
	}
}

// WithEmail sets the user email
func (b *UserBuilder) WithEmail(email string) *UserBuilder {
	b.user.Email = email
	return b
}

// WithName sets the user name
func (b *UserBuilder) WithName(name string) *UserBuilder {
	b.user.Name = name
	return b
}

// WithEmailVerified sets email verification status
func (b *UserBuilder) WithEmailVerified(verified bool) *UserBuilder {
	b.user.EmailVerified = verified
	return b
}

// WithRole sets the role for the default organization
func (b *UserBuilder) WithRole(role string) *UserBuilder {
	b.role = role
	return b
}

// Build saves the user and returns it
func (b *UserBuilder) Build() *schema.User {
	b.mock.mu.Lock()
	defer b.mock.mu.Unlock()

	b.mock.users[b.user.ID] = b.user

	// Add to default org
	member := &schema.OrganizationMember{
		ID:             xid.New(),
		OrganizationID: b.mock.defaultOrg.ID,
		UserID:         b.user.ID,
		Role:           b.role,
		Status:         "active",
	}
	b.mock.members[b.mock.defaultOrg.ID] = append(b.mock.members[b.mock.defaultOrg.ID], member)

	return b.user
}
