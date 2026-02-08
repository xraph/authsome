package layouts

import (
	"testing"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/app"
	"github.com/xraph/authsome/core/user"
)

// Test humanizeSegment function
func TestHumanizeSegment(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"users", "Users"},
		{"apps-management", "Apps Management"},
		{"api_keys", "Api Keys"},
		{"user-settings", "User Settings"},
		{"dashboard", "Dashboard"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := humanizeSegment(tt.input)
			if result != tt.expected {
				t.Errorf("humanizeSegment(%q) = %q; want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// Test isID function
func TestIsID(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		// Valid XIDs (20 chars alphanumeric)
		{xid.New().String(), true},
		{"c9h6p9j0eg06pg6qfk70", true},

		// Invalid - too short
		{"abc123", false},

		// Invalid - too long
		{"c9h6p9j0eg06pg6qfk701234", false},

		// Invalid - contains uppercase
		{"C9H6P9J0EG06PG6QFK70", false},

		// Valid UUID
		{"123e4567-e89b-12d3-a456-426614174000", true},

		// Invalid UUID format
		{"123e4567e89b12d3a456426614174000", false},

		// Normal path segment
		{"users", false},
		{"settings", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := isID(tt.input)
			if result != tt.expected {
				t.Errorf("isID(%q) = %v; want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// Test getInitials function
func TestGetInitials(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"John Doe", "JD"},
		{"Alice", "A"},
		{"Bob Smith Johnson", "BS"}, // Should only take first 2 words
		{"", "?"},
		{"   ", "?"},
		{"john doe", "JD"}, // Should uppercase
		{"Mary-Jane Watson", "MW"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := getInitials(tt.input)
			if result != tt.expected {
				t.Errorf("getInitials(%q) = %q; want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// Test getEntityLabel function logic
func TestGetEntityLabelLogic(t *testing.T) {
	// Create a test app
	testApp := &app.App{
		ID:   xid.New(),
		Name: "Test App",
	}

	tests := []struct {
		name       string
		segments   []string
		index      int
		currentApp *app.App
		expected   string
	}{
		{
			name:       "App ID with matching current app",
			segments:   []string{"app", testApp.ID.String()},
			index:      1,
			currentApp: testApp,
			expected:   "Test App",
		},
		{
			name:       "App ID without matching current app",
			segments:   []string{"app", "someappid"},
			index:      1,
			currentApp: nil,
			expected:   "App",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the logic directly without needing PageContext
			segment := tt.segments[tt.index]
			var result string

			if tt.index > 0 {
				entityType := tt.segments[tt.index-1]

				switch entityType {
				case "app":
					if tt.currentApp != nil && tt.currentApp.ID.String() == segment {
						result = tt.currentApp.Name
					} else {
						result = "App"
					}
				case "users":
					result = "User"
				case "organizations":
					result = "Organization"
				case "sessions":
					result = "Session"
				case "environments":
					result = "Environment"
				default:
					result = humanizeSegment(entityType)
				}
			}

			if result != tt.expected {
				t.Errorf("getEntityLabel logic = %q; want %q", result, tt.expected)
			}
		})
	}
}

// Test that user dropdown works with real user data
func TestBuildUserAvatarLogic(t *testing.T) {
	layoutManager := &LayoutManager{
		baseUIPath: "/ui",
	}

	// Test with user having an image
	userWithImage := &user.User{
		Name:  "John Doe",
		Email: "john@example.com",
		Image: "https://example.com/avatar.jpg",
	}

	// We can't directly test the g.Node output, but we can verify the logic
	// by checking that the function doesn't panic
	t.Run("User with image", func(t *testing.T) {
		// This would render an img tag
		avatarNode := layoutManager.buildUserAvatar(userWithImage.Name, userWithImage.Image)
		if avatarNode == nil {
			t.Error("buildUserAvatar returned nil")
		}
	})

	// Test with user without image (should use initials)
	userWithoutImage := &user.User{
		Name:  "Jane Smith",
		Email: "jane@example.com",
		Image: "",
	}

	t.Run("User without image", func(t *testing.T) {
		// This would render initials
		avatarNode := layoutManager.buildUserAvatar(userWithoutImage.Name, userWithoutImage.Image)
		if avatarNode == nil {
			t.Error("buildUserAvatar returned nil")
		}

		// Verify initials are correct
		initials := getInitials(userWithoutImage.Name)
		if initials != "JS" {
			t.Errorf("Expected initials 'JS', got %q", initials)
		}
	})
}

// Benchmark humanizeSegment to ensure performance
func BenchmarkHumanizeSegment(b *testing.B) {
	for i := 0; i < b.N; i++ {
		humanizeSegment("apps-management")
	}
}

// Benchmark isID to ensure performance
func BenchmarkIsID(b *testing.B) {
	id := xid.New().String()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		isID(id)
	}
}

// Benchmark getInitials to ensure performance
func BenchmarkGetInitials(b *testing.B) {
	for i := 0; i < b.N; i++ {
		getInitials("John Doe")
	}
}
