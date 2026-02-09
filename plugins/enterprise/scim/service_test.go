package scim

import (
	"testing"
	"time"

	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/authsome/core/user"
)

// Test basic SCIM user mapping
func TestMapSCIMToAuthSomeUser(t *testing.T) {
	service := &Service{
		config: DefaultConfig(),
	}

	tests := []struct {
		name         string
		scimUser     *SCIMUser
		wantEmail    string
		wantSCIMName string
	}{
		{
			name: "Basic user with display name",
			scimUser: &SCIMUser{
				UserName:    "john.doe",
				DisplayName: "John Doe",
				Active:      true,
				Emails: []Email{
					{Value: "john.doe@example.com", Primary: true, Type: "work"},
				},
			},
			wantEmail:    "john.doe@example.com",
			wantSCIMName: "John Doe",
		},
		{
			name: "User with structured name",
			scimUser: &SCIMUser{
				UserName: "jane.smith",
				Active:   true,
				Name: &SCIMName{
					GivenName:  "Jane",
					FamilyName: "Smith",
				},
				Emails: []Email{
					{Value: "jane.smith@example.com", Primary: true},
				},
			},
			wantEmail:    "jane.smith@example.com",
			wantSCIMName: "Jane Smith",
		},
		{
			name: "Inactive user",
			scimUser: &SCIMUser{
				UserName: "inactive@example.com",
				Active:   false,
				Emails: []Email{
					{Value: "inactive@example.com", Primary: true},
				},
			},
			wantEmail:    "inactive@example.com",
			wantSCIMName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := service.mapSCIMToAuthSomeUser(tt.scimUser, xid.New())

			require.NoError(t, err)
			assert.Equal(t, tt.wantEmail, user.Email)
			assert.Equal(t, tt.wantSCIMName, user.Name)
			assert.Equal(t, tt.scimUser.Active, user.EmailVerified)
		})
	}
}

// Test SCIM user to AuthSome user mapping
func TestMapAuthSomeToSCIMUser(t *testing.T) {
	service := &Service{
		config: DefaultConfig(),
	}

	userID := xid.New()
	now := time.Now()

	authUser := &user.User{
		ID:            userID,
		Email:         "test@example.com",
		Name:          "Test User",
		EmailVerified: true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	scimUser := service.mapAuthSomeToSCIMUser(authUser, "ext123")

	assert.Equal(t, userID.String(), scimUser.ID)
	assert.Equal(t, "ext123", scimUser.ExternalID)
	assert.Equal(t, "test@example.com", scimUser.UserName)
	assert.Equal(t, "Test User", scimUser.DisplayName)
	assert.True(t, scimUser.Active)
	assert.Len(t, scimUser.Emails, 1)
	assert.Equal(t, "test@example.com", scimUser.Emails[0].Value)
	assert.True(t, scimUser.Emails[0].Primary)
	assert.NotNil(t, scimUser.Meta)
}

// Test helper functions
func TestGetPrimaryEmail(t *testing.T) {
	service := &Service{
		config: DefaultConfig(),
	}

	tests := []struct {
		name      string
		scimUser  *SCIMUser
		wantEmail string
	}{
		{
			name: "Single primary email",
			scimUser: &SCIMUser{
				Emails: []Email{
					{Value: "primary@example.com", Primary: true},
				},
			},
			wantEmail: "primary@example.com",
		},
		{
			name: "Multiple emails with primary",
			scimUser: &SCIMUser{
				Emails: []Email{
					{Value: "secondary@example.com", Primary: false},
					{Value: "primary@example.com", Primary: true},
				},
			},
			wantEmail: "primary@example.com",
		},
		{
			name: "No primary, use first",
			scimUser: &SCIMUser{
				Emails: []Email{
					{Value: "first@example.com", Primary: false},
					{Value: "second@example.com", Primary: false},
				},
			},
			wantEmail: "first@example.com",
		},
		{
			name: "No emails",
			scimUser: &SCIMUser{
				Emails: []Email{},
			},
			wantEmail: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			email := service.getPrimaryEmail(tt.scimUser)
			assert.Equal(t, tt.wantEmail, email)
		})
	}
}

// Test validation
func TestValidateUserAttributes(t *testing.T) {
	service := &Service{
		config: DefaultConfig(),
	}
	service.config.UserProvisioning.RequiredAttributes = []string{"userName", "emails"}

	tests := []struct {
		name      string
		scimUser  *SCIMUser
		wantError bool
	}{
		{
			name: "Valid user",
			scimUser: &SCIMUser{
				UserName: "valid@example.com",
				Emails: []Email{
					{Value: "valid@example.com", Primary: true},
				},
			},
			wantError: false,
		},
		{
			name: "Missing userName",
			scimUser: &SCIMUser{
				Emails: []Email{
					{Value: "test@example.com"},
				},
			},
			wantError: true,
		},
		{
			name: "Missing emails",
			scimUser: &SCIMUser{
				UserName: "test@example.com",
				Emails:   []Email{},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.validateUserAttributes(tt.scimUser)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Test patch operations
func TestApplyPatchOperationToRequest(t *testing.T) {
	service := &Service{
		config: DefaultConfig(),
	}

	authUser := &user.User{
		ID:            xid.New(),
		Email:         "test@example.com",
		EmailVerified: true,
	}

	tests := []struct {
		name      string
		operation *PatchOperation
		checkFunc func(*testing.T, *user.UpdateUserRequest)
	}{
		{
			name: "Replace active status",
			operation: &PatchOperation{
				Op:    "replace",
				Path:  "active",
				Value: false,
			},
			checkFunc: func(t *testing.T, req *user.UpdateUserRequest) {
				require.NotNil(t, req.EmailVerified)
				assert.False(t, *req.EmailVerified)
			},
		},
		{
			name: "Replace display name",
			operation: &PatchOperation{
				Op:    "replace",
				Path:  "displayName",
				Value: "New Name",
			},
			checkFunc: func(t *testing.T, req *user.UpdateUserRequest) {
				require.NotNil(t, req.Name)
				assert.Equal(t, "New Name", *req.Name)
			},
		},
		{
			name: "Replace email",
			operation: &PatchOperation{
				Op:    "replace",
				Path:  "emails[primary eq true].value",
				Value: "new@example.com",
			},
			checkFunc: func(t *testing.T, req *user.UpdateUserRequest) {
				require.NotNil(t, req.Email)
				assert.Equal(t, "new@example.com", *req.Email)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updateReq := &user.UpdateUserRequest{}
			err := service.applyPatchOperationToRequest(authUser, tt.operation, updateReq)
			require.NoError(t, err)
			tt.checkFunc(t, updateReq)
		})
	}
}

// Race condition tests
func TestConcurrentGetPrimaryEmail(t *testing.T) {
	service := &Service{
		config: DefaultConfig(),
	}

	scimUser := &SCIMUser{
		Emails: []Email{
			{Value: "test@example.com", Primary: true},
		},
	}

	const goroutines = 100
	done := make(chan string, goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			email := service.getPrimaryEmail(scimUser)
			done <- email
		}()
	}

	for i := 0; i < goroutines; i++ {
		email := <-done
		assert.Equal(t, "test@example.com", email)
	}
}

func TestConcurrentMapSCIMToAuthSomeUser(t *testing.T) {
	service := &Service{
		config: DefaultConfig(),
	}

	scimUser := &SCIMUser{
		UserName:    "concurrent@example.com",
		DisplayName: "Concurrent User",
		Active:      true,
		Emails: []Email{
			{Value: "concurrent@example.com", Primary: true},
		},
	}

	const goroutines = 100
	done := make(chan bool, goroutines)
	errors := make(chan error, goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			_, err := service.mapSCIMToAuthSomeUser(scimUser, xid.New())
			if err != nil {
				errors <- err
			}
			done <- true
		}()
	}

	for i := 0; i < goroutines; i++ {
		<-done
	}

	close(errors)
	for err := range errors {
		t.Errorf("Unexpected error: %v", err)
	}
}

// Benchmark tests
func BenchmarkMapSCIMToAuthSomeUser(b *testing.B) {
	service := &Service{
		config: DefaultConfig(),
	}

	scimUser := &SCIMUser{
		UserName:    "bench@example.com",
		DisplayName: "Benchmark User",
		Active:      true,
		Emails: []Email{
			{Value: "bench@example.com", Primary: true},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.mapSCIMToAuthSomeUser(scimUser, xid.New())
	}
}

func BenchmarkMapAuthSomeToSCIMUser(b *testing.B) {
	service := &Service{
		config: DefaultConfig(),
	}

	authUser := &user.User{
		ID:            xid.New(),
		Email:         "bench@example.com",
		Name:          "Benchmark User",
		EmailVerified: true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.mapAuthSomeToSCIMUser(authUser, "ext123")
	}
}

func BenchmarkGetPrimaryEmail(b *testing.B) {
	service := &Service{
		config: DefaultConfig(),
	}

	scimUser := &SCIMUser{
		Emails: []Email{
			{Value: "primary@example.com", Primary: true},
			{Value: "secondary@example.com", Primary: false},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.getPrimaryEmail(scimUser)
	}
}

func BenchmarkValidateUserAttributes(b *testing.B) {
	service := &Service{
		config: DefaultConfig(),
	}
	service.config.UserProvisioning.RequiredAttributes = []string{"userName", "emails"}

	scimUser := &SCIMUser{
		UserName: "bench@example.com",
		Emails: []Email{
			{Value: "bench@example.com", Primary: true},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.validateUserAttributes(scimUser)
	}
}
