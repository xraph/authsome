package user

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/xraph/authsome/internal/validator"
	"github.com/xraph/authsome/types"
)

// MockRepository is a mock implementation of the Repository interface
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) Create(ctx context.Context, user *User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockRepository) FindByID(ctx context.Context, id xid.ID) (*User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*User), args.Error(1)
}

func (m *MockRepository) FindByEmail(ctx context.Context, email string) (*User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*User), args.Error(1)
}

func (m *MockRepository) FindByUsername(ctx context.Context, username string) (*User, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*User), args.Error(1)
}

func (m *MockRepository) Update(ctx context.Context, user *User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockRepository) Delete(ctx context.Context, id xid.ID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) List(ctx context.Context, limit, offset int) ([]*User, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*User), args.Error(1)
}

func (m *MockRepository) Count(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}

func (m *MockRepository) CountCreatedSince(ctx context.Context, since time.Time) (int, error) {
	args := m.Called(ctx, since)
	return args.Int(0), args.Error(1)
}

func (m *MockRepository) Search(ctx context.Context, query string, limit, offset int) ([]*User, error) {
	args := m.Called(ctx, query, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*User), args.Error(1)
}

func (m *MockRepository) CountSearch(ctx context.Context, query string) (int, error) {
	args := m.Called(ctx, query)
	return args.Int(0), args.Error(1)
}

// Helper function to create a test service
func newTestService(repo Repository) *Service {
	return NewService(repo, Config{
		PasswordRequirements: validator.DefaultPasswordRequirements(),
	}, nil)
}

func TestService_Create(t *testing.T) {
	tests := []struct {
		name    string
		req     *CreateUserRequest
		setup   func(*MockRepository)
		wantErr bool
		errMsg  string
	}{
		{
			name: "successful user creation",
			req: &CreateUserRequest{
				Email:    "test@example.com",
				Password: "SecurePass123!",
				Name:     "Test User",
			},
			setup: func(m *MockRepository) {
				m.On("FindByEmail", mock.Anything, "test@example.com").Return(nil, errors.New("not found"))
				m.On("Create", mock.Anything, mock.AnythingOfType("*user.User")).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "invalid email",
			req: &CreateUserRequest{
				Email:    "invalid-email",
				Password: "SecurePass123!",
				Name:     "Test User",
			},
			setup:   func(m *MockRepository) {},
			wantErr: true,
			errMsg:  "invalid email",
		},
		{
			name: "weak password",
			req: &CreateUserRequest{
				Email:    "test@example.com",
				Password: "weak",
				Name:     "Test User",
			},
			setup:   func(m *MockRepository) {},
			wantErr: true,
			errMsg:  "password",
		},
		{
			name: "email already exists",
			req: &CreateUserRequest{
				Email:    "existing@example.com",
				Password: "SecurePass123!",
				Name:     "Test User",
			},
			setup: func(m *MockRepository) {
				existingUser := &User{
					ID:    xid.New(),
					Email: "existing@example.com",
				}
				m.On("FindByEmail", mock.Anything, "existing@example.com").Return(existingUser, nil)
			},
			wantErr: true,
			errMsg:  "email already exists",
		},
		{
			name: "repository create error",
			req: &CreateUserRequest{
				Email:    "test@example.com",
				Password: "SecurePass123!",
				Name:     "Test User",
			},
			setup: func(m *MockRepository) {
				m.On("FindByEmail", mock.Anything, "test@example.com").Return(nil, errors.New("not found"))
				m.On("Create", mock.Anything, mock.AnythingOfType("*user.User")).Return(errors.New("database error"))
			},
			wantErr: true,
			errMsg:  "database error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			tt.setup(mockRepo)
			svc := newTestService(mockRepo)

			user, err := svc.Create(context.Background(), tt.req)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, tt.req.Email, user.Email)
				assert.Equal(t, tt.req.Name, user.Name)
				assert.NotEmpty(t, user.ID)
				assert.NotEmpty(t, user.PasswordHash)
				assert.NotEqual(t, tt.req.Password, user.PasswordHash) // Password should be hashed
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestService_FindByID(t *testing.T) {
	userID := xid.New()
	expectedUser := &User{
		ID:    userID,
		Email: "test@example.com",
		Name:  "Test User",
	}

	tests := []struct {
		name    string
		id      xid.ID
		setup   func(*MockRepository)
		want    *User
		wantErr bool
	}{
		{
			name: "user found",
			id:   userID,
			setup: func(m *MockRepository) {
				m.On("FindByID", mock.Anything, userID).Return(expectedUser, nil)
			},
			want:    expectedUser,
			wantErr: false,
		},
		{
			name: "user not found",
			id:   xid.New(),
			setup: func(m *MockRepository) {
				m.On("FindByID", mock.Anything, mock.Anything).Return(nil, errors.New("not found"))
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			tt.setup(mockRepo)
			svc := newTestService(mockRepo)

			user, err := svc.FindByID(context.Background(), tt.id)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, user)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestService_FindByEmail(t *testing.T) {
	expectedUser := &User{
		ID:    xid.New(),
		Email: "test@example.com",
		Name:  "Test User",
	}

	tests := []struct {
		name    string
		email   string
		setup   func(*MockRepository)
		want    *User
		wantErr bool
	}{
		{
			name:  "user found",
			email: "test@example.com",
			setup: func(m *MockRepository) {
				m.On("FindByEmail", mock.Anything, "test@example.com").Return(expectedUser, nil)
			},
			want:    expectedUser,
			wantErr: false,
		},
		{
			name:  "user not found",
			email: "nonexistent@example.com",
			setup: func(m *MockRepository) {
				m.On("FindByEmail", mock.Anything, "nonexistent@example.com").Return(nil, errors.New("not found"))
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			tt.setup(mockRepo)
			svc := newTestService(mockRepo)

			user, err := svc.FindByEmail(context.Background(), tt.email)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, user)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestService_Update(t *testing.T) {
	userID := xid.New()
	existingUser := &User{
		ID:        userID,
		Email:     "test@example.com",
		Name:      "Old Name",
		Username:  userID.String(),
		CreatedAt: time.Now().Add(-24 * time.Hour),
		UpdatedAt: time.Now().Add(-24 * time.Hour),
	}

	newName := "New Name"
	newImage := "https://example.com/image.jpg"
	newUsername := "newusername"

	tests := []struct {
		name    string
		user    *User
		req     *UpdateUserRequest
		setup   func(*MockRepository)
		wantErr bool
		errMsg  string
		check   func(*testing.T, *User)
	}{
		{
			name: "update name",
			user: &User{
				ID:        existingUser.ID,
				Email:     existingUser.Email,
				Name:      existingUser.Name,
				UpdatedAt: existingUser.UpdatedAt,
			},
			req: &UpdateUserRequest{
				Name: &newName,
			},
			setup: func(m *MockRepository) {
				m.On("Update", mock.Anything, mock.AnythingOfType("*user.User")).Return(nil)
			},
			wantErr: false,
			check: func(t *testing.T, u *User) {
				assert.Equal(t, newName, u.Name)
				assert.True(t, u.UpdatedAt.After(existingUser.UpdatedAt))
			},
		},
		{
			name: "update image",
			user: &User{
				ID:        existingUser.ID,
				Email:     existingUser.Email,
				Name:      existingUser.Name,
				UpdatedAt: existingUser.UpdatedAt,
			},
			req: &UpdateUserRequest{
				Image: &newImage,
			},
			setup: func(m *MockRepository) {
				m.On("Update", mock.Anything, mock.AnythingOfType("*user.User")).Return(nil)
			},
			wantErr: false,
			check: func(t *testing.T, u *User) {
				assert.Equal(t, newImage, u.Image)
			},
		},
		{
			name: "update username - valid",
			user: &User{
				ID:        existingUser.ID,
				Email:     existingUser.Email,
				Name:      existingUser.Name,
				Username:  existingUser.Username,
				UpdatedAt: existingUser.UpdatedAt,
			},
			req: &UpdateUserRequest{
				Username: &newUsername,
			},
			setup: func(m *MockRepository) {
				m.On("FindByUsername", mock.Anything, newUsername).Return(nil, errors.New("not found"))
				m.On("Update", mock.Anything, mock.AnythingOfType("*user.User")).Return(nil)
			},
			wantErr: false,
			check: func(t *testing.T, u *User) {
				assert.Equal(t, newUsername, u.Username)
				assert.Equal(t, newUsername, u.DisplayUsername)
			},
		},
		{
			name: "update username - invalid characters",
			user: &User{
				ID:       existingUser.ID,
				Email:    existingUser.Email,
				Name:     existingUser.Name,
				Username: existingUser.Username,
			},
			req: &UpdateUserRequest{
				Username: stringPtr("invalid user!"),
			},
			setup:   func(m *MockRepository) {},
			wantErr: true,
			errMsg:  "invalid username",
		},
		{
			name: "update username - already taken",
			user: &User{
				ID:       existingUser.ID,
				Email:    existingUser.Email,
				Name:     existingUser.Name,
				Username: existingUser.Username,
			},
			req: &UpdateUserRequest{
				Username: &newUsername,
			},
			setup: func(m *MockRepository) {
				existingUserWithUsername := &User{
					ID:       xid.New(),
					Username: newUsername,
				}
				m.On("FindByUsername", mock.Anything, newUsername).Return(existingUserWithUsername, nil)
			},
			wantErr: true,
			errMsg:  "already taken",
		},
		{
			name: "update username - empty string",
			user: &User{
				ID:       existingUser.ID,
				Email:    existingUser.Email,
				Name:     existingUser.Name,
				Username: existingUser.Username,
			},
			req: &UpdateUserRequest{
				Username: stringPtr("   "),
			},
			setup:   func(m *MockRepository) {},
			wantErr: true,
			errMsg:  "cannot be empty",
		},
		{
			name: "repository update error",
			user: &User{
				ID:        existingUser.ID,
				Email:     existingUser.Email,
				Name:      existingUser.Name,
				UpdatedAt: existingUser.UpdatedAt,
			},
			req: &UpdateUserRequest{
				Name: &newName,
			},
			setup: func(m *MockRepository) {
				m.On("Update", mock.Anything, mock.AnythingOfType("*user.User")).Return(errors.New("database error"))
			},
			wantErr: true,
			errMsg:  "database error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			tt.setup(mockRepo)
			svc := newTestService(mockRepo)

			user, err := svc.Update(context.Background(), tt.user, tt.req)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
				if tt.check != nil {
					tt.check(t, user)
				}
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestService_Delete(t *testing.T) {
	userID := xid.New()
	existingUser := &User{
		ID:    userID,
		Email: "test@example.com",
		Name:  "Test User",
	}

	tests := []struct {
		name    string
		id      xid.ID
		setup   func(*MockRepository)
		wantErr bool
	}{
		{
			name: "successful deletion",
			id:   userID,
			setup: func(m *MockRepository) {
				m.On("FindByID", mock.Anything, userID).Return(existingUser, nil)
				m.On("Delete", mock.Anything, userID).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "user not found",
			id:   xid.New(),
			setup: func(m *MockRepository) {
				m.On("FindByID", mock.Anything, mock.Anything).Return(nil, errors.New("not found"))
			},
			wantErr: true,
		},
		{
			name: "repository delete error",
			id:   userID,
			setup: func(m *MockRepository) {
				m.On("FindByID", mock.Anything, userID).Return(existingUser, nil)
				m.On("Delete", mock.Anything, userID).Return(errors.New("database error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			tt.setup(mockRepo)
			svc := newTestService(mockRepo)

			err := svc.Delete(context.Background(), tt.id)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestService_List(t *testing.T) {
	users := []*User{
		{ID: xid.New(), Email: "user1@example.com", Name: "User 1"},
		{ID: xid.New(), Email: "user2@example.com", Name: "User 2"},
		{ID: xid.New(), Email: "user3@example.com", Name: "User 3"},
	}

	tests := []struct {
		name      string
		opts      types.PaginationOptions
		setup     func(*MockRepository)
		wantCount int
		wantTotal int
		wantErr   bool
	}{
		{
			name: "list users with default pagination",
			opts: types.PaginationOptions{Page: 1, PageSize: 20},
			setup: func(m *MockRepository) {
				m.On("List", mock.Anything, 20, 0).Return(users, nil)
				m.On("Count", mock.Anything).Return(3, nil)
			},
			wantCount: 3,
			wantTotal: 3,
			wantErr:   false,
		},
		{
			name: "list users with custom page size",
			opts: types.PaginationOptions{Page: 1, PageSize: 2},
			setup: func(m *MockRepository) {
				m.On("List", mock.Anything, 2, 0).Return(users[:2], nil)
				m.On("Count", mock.Anything).Return(3, nil)
			},
			wantCount: 2,
			wantTotal: 3,
			wantErr:   false,
		},
		{
			name: "list users page 2",
			opts: types.PaginationOptions{Page: 2, PageSize: 2},
			setup: func(m *MockRepository) {
				m.On("List", mock.Anything, 2, 2).Return(users[2:], nil)
				m.On("Count", mock.Anything).Return(3, nil)
			},
			wantCount: 1,
			wantTotal: 3,
			wantErr:   false,
		},
		{
			name: "repository list error",
			opts: types.PaginationOptions{Page: 1, PageSize: 20},
			setup: func(m *MockRepository) {
				m.On("List", mock.Anything, 20, 0).Return(nil, errors.New("database error"))
			},
			wantErr: true,
		},
		{
			name: "repository count error",
			opts: types.PaginationOptions{Page: 1, PageSize: 20},
			setup: func(m *MockRepository) {
				m.On("List", mock.Anything, 20, 0).Return(users, nil)
				m.On("Count", mock.Anything).Return(0, errors.New("database error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			tt.setup(mockRepo)
			svc := newTestService(mockRepo)

			list, total, err := svc.List(context.Background(), tt.opts)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, list, tt.wantCount)
				assert.Equal(t, tt.wantTotal, total)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

// Helper function
func stringPtr(s string) *string {
	return &s
}

