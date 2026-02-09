package user

import (
	"context"
	"database/sql"
	"testing"

	"github.com/rs/xid"
	"github.com/stretchr/testify/mock"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/internal/validator"
	"github.com/xraph/authsome/schema"
)

// =============================================================================
// BENCHMARK TESTS
// =============================================================================

// BenchmarkService_Create benchmarks user creation.
func BenchmarkService_Create(b *testing.B) {
	appID := testAppID()
	mockRepo := new(MockRepository)
	mockRepo.On("FindByAppAndEmail", mock.Anything, mock.Anything, mock.Anything).
		Return(nil, sql.ErrNoRows)
	mockRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	svc := NewService(mockRepo, Config{
		PasswordRequirements: validator.DefaultPasswordRequirements(),
	}, nil, nil)

	req := &CreateUserRequest{
		AppID:    appID,
		Email:    "benchmark@example.com",
		Password: "SecurePass123!",
		Name:     "Benchmark User",
	}

	b.ReportAllocs()

	for b.Loop() {
		_, _ = svc.Create(context.Background(), req)
	}
}

// BenchmarkService_FindByID benchmarks finding a user by ID.
func BenchmarkService_FindByID(b *testing.B) {
	appID := testAppID()
	userID := xid.New()
	existingUser := testSchemaUser(appID)
	existingUser.ID = userID

	mockRepo := new(MockRepository)
	mockRepo.On("FindByID", mock.Anything, userID).Return(existingUser, nil)

	svc := NewService(mockRepo, Config{}, nil, nil)

	b.ReportAllocs()

	for b.Loop() {
		_, _ = svc.FindByID(context.Background(), userID)
	}
}

// BenchmarkService_FindByAppAndEmail benchmarks finding a user by app and email.
func BenchmarkService_FindByAppAndEmail(b *testing.B) {
	appID := testAppID()
	existingUser := testSchemaUser(appID)

	mockRepo := new(MockRepository)
	mockRepo.On("FindByAppAndEmail", mock.Anything, appID, "test@example.com").
		Return(existingUser, nil)

	svc := NewService(mockRepo, Config{}, nil, nil)

	b.ReportAllocs()

	for b.Loop() {
		_, _ = svc.FindByAppAndEmail(context.Background(), appID, "test@example.com")
	}
}

// BenchmarkService_Update benchmarks user updates.
func BenchmarkService_Update(b *testing.B) {
	appID := testAppID()
	userID := xid.New()
	user := FromSchemaUser(testSchemaUser(appID))
	user.ID = userID

	newName := "New Name"
	req := &UpdateUserRequest{
		Name: &newName,
	}

	mockRepo := new(MockRepository)
	mockRepo.On("Update", mock.Anything, mock.Anything).Return(nil)

	svc := NewService(mockRepo, Config{}, nil, nil)

	b.ReportAllocs()

	for b.Loop() {
		userCopy := *user
		_, _ = svc.Update(context.Background(), &userCopy, req)
	}
}

// BenchmarkService_ListUsers benchmarks listing users with pagination.
func BenchmarkService_ListUsers(b *testing.B) {
	appID := testAppID()

	users := make([]*schema.User, 20)
	for i := range 20 {
		users[i] = testSchemaUser(appID)
	}

	pageResp := &pagination.PageResponse[*schema.User]{
		Data: users,
		Pagination: &pagination.PageMeta{
			Total:       100,
			Limit:       20,
			CurrentPage: 1,
			TotalPages:  5,
		},
	}

	mockRepo := new(MockRepository)
	mockRepo.On("ListUsers", mock.Anything, mock.Anything).Return(pageResp, nil)

	svc := NewService(mockRepo, Config{}, nil, nil)

	filter := &ListUsersFilter{
		PaginationParams: pagination.PaginationParams{
			Page:  1,
			Limit: 20,
		},
		AppID: appID,
	}

	b.ReportAllocs()

	for b.Loop() {
		_, _ = svc.ListUsers(context.Background(), filter)
	}
}

// BenchmarkService_CountUsers benchmarks counting users.
func BenchmarkService_CountUsers(b *testing.B) {
	appID := testAppID()
	mockRepo := new(MockRepository)
	mockRepo.On("CountUsers", mock.Anything, mock.Anything).Return(100, nil)

	svc := NewService(mockRepo, Config{}, nil, nil)

	filter := &CountUsersFilter{
		AppID: appID,
	}

	b.ReportAllocs()

	for b.Loop() {
		_, _ = svc.CountUsers(context.Background(), filter)
	}
}

// BenchmarkService_Create_Parallel benchmarks concurrent user creation.
func BenchmarkService_Create_Parallel(b *testing.B) {
	appID := testAppID()
	mockRepo := new(MockRepository)
	mockRepo.On("FindByAppAndEmail", mock.Anything, mock.Anything, mock.Anything).
		Return(nil, sql.ErrNoRows)
	mockRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	svc := NewService(mockRepo, Config{
		PasswordRequirements: validator.DefaultPasswordRequirements(),
	}, nil, nil)

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		req := &CreateUserRequest{
			AppID:    appID,
			Email:    "benchmark@example.com",
			Password: "SecurePass123!",
			Name:     "Benchmark User",
		}

		for pb.Next() {
			_, _ = svc.Create(context.Background(), req)
		}
	})
}

// BenchmarkFromSchemaUser benchmarks DTO conversion.
func BenchmarkFromSchemaUser(b *testing.B) {
	appID := testAppID()
	schemaUser := testSchemaUser(appID)

	b.ReportAllocs()

	for b.Loop() {
		_ = FromSchemaUser(schemaUser)
	}
}

// BenchmarkToSchema benchmarks schema conversion.
func BenchmarkToSchema(b *testing.B) {
	appID := testAppID()
	user := FromSchemaUser(testSchemaUser(appID))

	b.ReportAllocs()

	for b.Loop() {
		_ = user.ToSchema()
	}
}
