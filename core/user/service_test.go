package user

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/internal/validator"
	"github.com/xraph/authsome/schema"
)

// =============================================================================
// MOCK REPOSITORY
// =============================================================================

// MockRepository is a mock implementation of the Repository interface
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) Create(ctx context.Context, user *schema.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockRepository) FindByID(ctx context.Context, id xid.ID) (*schema.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*schema.User), args.Error(1)
}

func (m *MockRepository) FindByEmail(ctx context.Context, email string) (*schema.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*schema.User), args.Error(1)
}

func (m *MockRepository) FindByAppAndEmail(ctx context.Context, appID xid.ID, email string) (*schema.User, error) {
	args := m.Called(ctx, appID, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*schema.User), args.Error(1)
}

func (m *MockRepository) FindByUsername(ctx context.Context, username string) (*schema.User, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*schema.User), args.Error(1)
}

func (m *MockRepository) Update(ctx context.Context, user *schema.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockRepository) Delete(ctx context.Context, id xid.ID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) ListUsers(ctx context.Context, filter *ListUsersFilter) (*pagination.PageResponse[*schema.User], error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pagination.PageResponse[*schema.User]), args.Error(1)
}

func (m *MockRepository) CountUsers(ctx context.Context, filter *CountUsersFilter) (int, error) {
	args := m.Called(ctx, filter)
	return args.Int(0), args.Error(1)
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// Helper function to create a test service
func newTestService(repo Repository) *Service {
	return NewService(repo, Config{
		PasswordRequirements: validator.DefaultPasswordRequirements(),
	}, nil, nil)
}

func testAppID() xid.ID {
	return xid.New()
}

func testSchemaUser(appID xid.ID) *schema.User {
	id := xid.New()
	now := time.Now().UTC()
	return &schema.User{
		ID:              id,
		AppID:           &appID,
		Email:           "test@example.com",
		Name:            "Test User",
		PasswordHash:    "$2a$10$test",
		Username:        id.String(),
		DisplayUsername: "",
		EmailVerified:   false,
		AuditableModel: schema.AuditableModel{
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
}

// =============================================================================
// CREATE TESTS
// =============================================================================

func TestService_Create(t *testing.T) {
	tests := []struct {
		name    string
		request *CreateUserRequest
		setup   func(*MockRepository)
		wantErr bool
		errType error
	}{
		{
			name: "successful user creation",
			request: &CreateUserRequest{
				AppID:    testAppID(),
				Email:    "newuser@example.com",
				Password: "SecurePass123!",
				Name:     "New User",
			},
			setup: func(m *MockRepository) {
				m.On("FindByAppAndEmail", mock.Anything, mock.Anything, "newuser@example.com").
					Return(nil, sql.ErrNoRows)
				m.On("Create", mock.Anything, mock.Anything).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "email already exists in app",
			request: &CreateUserRequest{
				AppID:    testAppID(),
				Email:    "existing@example.com",
				Password: "SecurePass123!",
				Name:     "Existing User",
			},
			setup: func(m *MockRepository) {
				appID := testAppID()
				existing := testSchemaUser(appID)
				m.On("FindByAppAndEmail", mock.Anything, mock.Anything, "existing@example.com").
					Return(existing, nil)
			},
			wantErr: true,
			errType: ErrEmailAlreadyExists,
		},
		{
			name: "invalid email format",
			request: &CreateUserRequest{
				AppID:    testAppID(),
				Email:    "invalid-email",
				Password: "SecurePass123!",
				Name:     "Test User",
			},
			setup:   func(m *MockRepository) {},
			wantErr: true,
			errType: ErrInvalidEmail,
		},
		{
			name: "weak password",
			request: &CreateUserRequest{
				AppID:    testAppID(),
				Email:    "test@example.com",
				Password: "123",
				Name:     "Test User",
			},
			setup:   func(m *MockRepository) {},
			wantErr: true,
			errType: ErrWeakPassword,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			tt.setup(mockRepo)
			service := newTestService(mockRepo)

			user, err := service.Create(context.Background(), tt.request)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errType != nil {
					assert.ErrorIs(t, err, tt.errType)
				}
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, tt.request.Email, user.Email)
				assert.Equal(t, tt.request.Name, user.Name)
				assert.Equal(t, tt.request.AppID, user.AppID)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

// =============================================================================
// FIND TESTS
// =============================================================================

func TestService_FindByID(t *testing.T) {
	appID := testAppID()
	testUser := testSchemaUser(appID)

	tests := []struct {
		name    string
		id      xid.ID
		setup   func(*MockRepository)
		wantErr bool
	}{
		{
			name: "user found",
			id:   testUser.ID,
			setup: func(m *MockRepository) {
				m.On("FindByID", mock.Anything, testUser.ID).Return(testUser, nil)
			},
			wantErr: false,
		},
		{
			name: "user not found",
			id:   xid.New(),
			setup: func(m *MockRepository) {
				m.On("FindByID", mock.Anything, mock.Anything).Return(nil, sql.ErrNoRows)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			tt.setup(mockRepo)
			service := newTestService(mockRepo)

			user, err := service.FindByID(context.Background(), tt.id)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, testUser.ID, user.ID)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestService_FindByAppAndEmail(t *testing.T) {
	appID := testAppID()
	testUser := testSchemaUser(appID)

	tests := []struct {
		name    string
		appID   xid.ID
		email   string
		setup   func(*MockRepository)
		wantErr bool
	}{
		{
			name:  "user found",
			appID: appID,
			email: testUser.Email,
			setup: func(m *MockRepository) {
				m.On("FindByAppAndEmail", mock.Anything, appID, testUser.Email).
					Return(testUser, nil)
			},
			wantErr: false,
		},
		{
			name:  "user not found",
			appID: appID,
			email: "nonexistent@example.com",
			setup: func(m *MockRepository) {
				m.On("FindByAppAndEmail", mock.Anything, appID, "nonexistent@example.com").
					Return(nil, sql.ErrNoRows)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			tt.setup(mockRepo)
			service := newTestService(mockRepo)

			user, err := service.FindByAppAndEmail(context.Background(), tt.appID, tt.email)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, testUser.Email, user.Email)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

// =============================================================================
// UPDATE TESTS
// =============================================================================

func TestService_Update(t *testing.T) {
	appID := testAppID()

	newName := "Updated Name"
	newEmail := "updated@example.com"

	tests := []struct {
		name    string
		user    *User
		request *UpdateUserRequest
		setup   func(*MockRepository)
		wantErr bool
	}{
		{
			name: "successful name update",
			user: FromSchemaUser(testSchemaUser(appID)),
			request: &UpdateUserRequest{
				Name: &newName,
			},
			setup: func(m *MockRepository) {
				m.On("Update", mock.Anything, mock.Anything).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "successful email update",
			user: FromSchemaUser(testSchemaUser(appID)),
			request: &UpdateUserRequest{
				Email: &newEmail,
			},
			setup: func(m *MockRepository) {
				m.On("FindByAppAndEmail", mock.Anything, appID, newEmail).
					Return(nil, sql.ErrNoRows)
				m.On("Update", mock.Anything, mock.Anything).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "email already taken",
			user: FromSchemaUser(testSchemaUser(appID)),
			request: &UpdateUserRequest{
				Email: &newEmail,
			},
			setup: func(m *MockRepository) {
				existing := testSchemaUser(appID)
				existing.ID = xid.New() // Different user
				m.On("FindByAppAndEmail", mock.Anything, appID, newEmail).
					Return(existing, nil)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			tt.setup(mockRepo)
			service := newTestService(mockRepo)

			user, err := service.Update(context.Background(), tt.user, tt.request)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

// =============================================================================
// DELETE TESTS
// =============================================================================

func TestService_Delete(t *testing.T) {
	appID := testAppID()
	testUser := testSchemaUser(appID)

	tests := []struct {
		name    string
		id      xid.ID
		setup   func(*MockRepository)
		wantErr bool
	}{
		{
			name: "successful deletion",
			id:   testUser.ID,
			setup: func(m *MockRepository) {
				m.On("FindByID", mock.Anything, testUser.ID).Return(testUser, nil)
				m.On("Delete", mock.Anything, testUser.ID).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "user not found",
			id:   xid.New(),
			setup: func(m *MockRepository) {
				m.On("FindByID", mock.Anything, mock.Anything).Return(nil, sql.ErrNoRows)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			tt.setup(mockRepo)
			service := newTestService(mockRepo)

			err := service.Delete(context.Background(), tt.id)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

// =============================================================================
// LIST TESTS
// =============================================================================

func TestService_ListUsers(t *testing.T) {
	appID := testAppID()
	users := []*schema.User{
		testSchemaUser(appID),
		testSchemaUser(appID),
	}

	tests := []struct {
		name    string
		filter  *ListUsersFilter
		setup   func(*MockRepository)
		wantErr bool
	}{
		{
			name: "successful list",
			filter: &ListUsersFilter{
				PaginationParams: pagination.PaginationParams{
					Page:  1,
					Limit: 10,
				},
				AppID: appID,
			},
			setup: func(m *MockRepository) {
				pageResp := &pagination.PageResponse[*schema.User]{
					Data: users,
					Pagination: &pagination.PageMeta{
						Total:       2,
						Limit:       10,
						CurrentPage: 1,
						TotalPages:  1,
					},
				}
				m.On("ListUsers", mock.Anything, mock.Anything).Return(pageResp, nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			tt.setup(mockRepo)
			service := newTestService(mockRepo)

			result, err := service.ListUsers(context.Background(), tt.filter)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Len(t, result.Data, 2)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

// =============================================================================
// COUNT TESTS
// =============================================================================

func TestService_CountUsers(t *testing.T) {
	appID := testAppID()

	tests := []struct {
		name    string
		filter  *CountUsersFilter
		setup   func(*MockRepository)
		want    int
		wantErr bool
	}{
		{
			name: "successful count",
			filter: &CountUsersFilter{
				AppID: appID,
			},
			setup: func(m *MockRepository) {
				m.On("CountUsers", mock.Anything, mock.Anything).Return(5, nil)
			},
			want:    5,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			tt.setup(mockRepo)
			service := newTestService(mockRepo)

			count, err := service.CountUsers(context.Background(), tt.filter)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, count)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}
