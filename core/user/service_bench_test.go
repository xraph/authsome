package user

import (
	"context"
	"errors"
	"testing"

	"github.com/rs/xid"
	"github.com/xraph/authsome/internal/validator"
	"github.com/xraph/authsome/types"
)

// BenchmarkService_Create benchmarks user creation
func BenchmarkService_Create(b *testing.B) {
	mockRepo := new(MockRepository)
	mockRepo.On("FindByEmail", MatchAny(), MatchAny()).Return(nil, errors.New("not found"))
	mockRepo.On("Create", MatchAny(), MatchAny()).Return(nil)

	svc := NewService(mockRepo, Config{
		PasswordRequirements: validator.DefaultPasswordRequirements(),
	}, nil)

	req := &CreateUserRequest{
		Email:    "benchmark@example.com",
		Password: "SecurePass123!",
		Name:     "Benchmark User",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = svc.Create(context.Background(), req)
	}
}

// BenchmarkService_FindByID benchmarks finding a user by ID
func BenchmarkService_FindByID(b *testing.B) {
	userID := xid.New()
	existingUser := &User{
		ID:    userID,
		Email: "test@example.com",
		Name:  "Test User",
	}

	mockRepo := new(MockRepository)
	mockRepo.On("FindByID", MatchAny(), userID).Return(existingUser, nil)

	svc := NewService(mockRepo, Config{}, nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = svc.FindByID(context.Background(), userID)
	}
}

// BenchmarkService_FindByEmail benchmarks finding a user by email
func BenchmarkService_FindByEmail(b *testing.B) {
	existingUser := &User{
		ID:    xid.New(),
		Email: "test@example.com",
		Name:  "Test User",
	}

	mockRepo := new(MockRepository)
	mockRepo.On("FindByEmail", MatchAny(), "test@example.com").Return(existingUser, nil)

	svc := NewService(mockRepo, Config{}, nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = svc.FindByEmail(context.Background(), "test@example.com")
	}
}

// BenchmarkService_Update benchmarks user updates
func BenchmarkService_Update(b *testing.B) {
	userID := xid.New()
	user := &User{
		ID:       userID,
		Email:    "test@example.com",
		Name:     "Old Name",
		Username: userID.String(),
	}

	newName := "New Name"
	req := &UpdateUserRequest{
		Name: &newName,
	}

	mockRepo := new(MockRepository)
	mockRepo.On("Update", MatchAny(), MatchAny()).Return(nil)

	svc := NewService(mockRepo, Config{}, nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		userCopy := *user
		_, _ = svc.Update(context.Background(), &userCopy, req)
	}
}

// BenchmarkService_List benchmarks listing users with pagination
func BenchmarkService_List(b *testing.B) {
	users := make([]*User, 20)
	for i := 0; i < 20; i++ {
		users[i] = &User{
			ID:    xid.New(),
			Email: "user@example.com",
			Name:  "User",
		}
	}

	mockRepo := new(MockRepository)
	mockRepo.On("List", MatchAny(), 20, 0).Return(users, nil)
	mockRepo.On("Count", MatchAny()).Return(100, nil)

	svc := NewService(mockRepo, Config{}, nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _, _ = svc.List(context.Background(), types.PaginationOptions{Page: 1, PageSize: 20})
	}
}

// BenchmarkService_Create_Parallel benchmarks concurrent user creation
func BenchmarkService_Create_Parallel(b *testing.B) {
	mockRepo := new(MockRepository)
	mockRepo.On("FindByEmail", MatchAny(), MatchAny()).Return(nil, errors.New("not found"))
	mockRepo.On("Create", MatchAny(), MatchAny()).Return(nil)

	svc := NewService(mockRepo, Config{
		PasswordRequirements: validator.DefaultPasswordRequirements(),
	}, nil)

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		req := &CreateUserRequest{
			Email:    "benchmark@example.com",
			Password: "SecurePass123!",
			Name:     "Benchmark User",
		}

		for pb.Next() {
			_, _ = svc.Create(context.Background(), req)
		}
	})
}

// Helper types for benchmarks
type PaginationOptions struct {
	Page     int
	PageSize int
}

// MatchAny is a helper for matching any argument in mocks
func MatchAny() interface{} {
	return interface{}(nil)
}
