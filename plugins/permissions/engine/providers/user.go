package providers

import (
	"context"
	"fmt"
)

// UserService defines the interface for fetching user data
// This should be implemented by AuthSome's user service.
type UserService interface {
	// GetUser fetches a user by ID
	GetUser(ctx context.Context, userID string) (*User, error)

	// GetUsers fetches multiple users by IDs
	GetUsers(ctx context.Context, userIDs []string) ([]*User, error)
}

// User represents user data for attribute resolution.
type User struct {
	ID            string         `json:"id"`
	Email         string         `json:"email"`
	Name          string         `json:"name"`
	Roles         []string       `json:"roles"`
	Groups        []string       `json:"groups"`
	OrgID         string         `json:"org_id"`
	Department    string         `json:"department"`
	Permissions   []string       `json:"permissions"`
	Metadata      map[string]any `json:"metadata"`
	CreatedAt     string         `json:"created_at"`
	EmailVerified bool           `json:"email_verified"`
	Active        bool           `json:"active"`
}

// UserAttributeProvider fetches user attributes from the user service.
type UserAttributeProvider struct {
	userService UserService
}

// NewUserAttributeProvider creates a new user attribute provider.
func NewUserAttributeProvider(userService UserService) *UserAttributeProvider {
	return &UserAttributeProvider{
		userService: userService,
	}
}

// Name returns the provider name.
func (p *UserAttributeProvider) Name() string {
	return "user"
}

// GetAttributes fetches user attributes by user ID.
func (p *UserAttributeProvider) GetAttributes(ctx context.Context, key string) (map[string]any, error) {
	// key is expected to be the user ID
	user, err := p.userService.GetUser(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user: %w", err)
	}

	return userToAttributes(user), nil
}

// GetBatchAttributes fetches attributes for multiple users.
func (p *UserAttributeProvider) GetBatchAttributes(ctx context.Context, keys []string) (map[string]map[string]any, error) {
	// keys are user IDs
	users, err := p.userService.GetUsers(ctx, keys)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch users: %w", err)
	}

	result := make(map[string]map[string]any)
	for _, user := range users {
		result[user.ID] = userToAttributes(user)
	}

	return result, nil
}

// userToAttributes converts a User to an attributes map.
func userToAttributes(user *User) map[string]any {
	if user == nil {
		return make(map[string]any)
	}

	attrs := map[string]any{
		"id":             user.ID,
		"email":          user.Email,
		"name":           user.Name,
		"roles":          user.Roles,
		"groups":         user.Groups,
		"org_id":         user.OrgID,
		"department":     user.Department,
		"permissions":    user.Permissions,
		"created_at":     user.CreatedAt,
		"email_verified": user.EmailVerified,
		"active":         user.Active,
	}

	// Merge metadata
	if user.Metadata != nil {
		for k, v := range user.Metadata {
			// Prefix metadata keys to avoid collisions
			attrs["meta_"+k] = v
		}
	}

	return attrs
}

// MockUserService provides a mock implementation for testing.
type MockUserService struct {
	users map[string]*User
}

// NewMockUserService creates a new mock user service.
func NewMockUserService() *MockUserService {
	return &MockUserService{
		users: make(map[string]*User),
	}
}

// AddUser adds a user to the mock service.
func (m *MockUserService) AddUser(user *User) {
	m.users[user.ID] = user
}

// GetUser fetches a user by ID.
func (m *MockUserService) GetUser(ctx context.Context, userID string) (*User, error) {
	user, exists := m.users[userID]
	if !exists {
		return nil, fmt.Errorf("user not found: %s", userID)
	}

	return user, nil
}

// GetUsers fetches multiple users by IDs.
func (m *MockUserService) GetUsers(ctx context.Context, userIDs []string) ([]*User, error) {
	result := make([]*User, 0, len(userIDs))
	for _, id := range userIDs {
		if user, exists := m.users[id]; exists {
			result = append(result, user)
		}
	}

	return result, nil
}
