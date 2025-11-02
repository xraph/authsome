package idverification

import (
	"testing"
)

func TestMeetsVerificationLevel(t *testing.T) {
	tests := []struct {
		current  string
		required string
		expected bool
	}{
		{"full", "full", true},
		{"full", "enhanced", true},
		{"full", "basic", true},
		{"full", "none", true},
		{"enhanced", "full", false},
		{"enhanced", "enhanced", true},
		{"enhanced", "basic", true},
		{"enhanced", "none", true},
		{"basic", "full", false},
		{"basic", "enhanced", false},
		{"basic", "basic", true},
		{"basic", "none", true},
		{"none", "full", false},
		{"none", "enhanced", false},
		{"none", "basic", false},
		{"none", "none", true},
		{"invalid", "full", false},
		{"full", "invalid", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.current+"_vs_"+tt.required, func(t *testing.T) {
			result := meetsVerificationLevel(tt.current, tt.required)
			if result != tt.expected {
				t.Errorf("Expected %v for %s vs %s, got %v", tt.expected, tt.current, tt.required, result)
			}
		})
	}
}

func TestVerificationLevelHierarchy(t *testing.T) {
	levels := []string{"none", "basic", "enhanced", "full"}
	
	for i, currentLevel := range levels {
		for j, requiredLevel := range levels {
			t.Run(currentLevel+"_vs_"+requiredLevel, func(t *testing.T) {
				result := meetsVerificationLevel(currentLevel, requiredLevel)
				expected := i >= j
				if result != expected {
					t.Errorf("Expected %v for %s (level %d) vs %s (level %d), got %v", 
						expected, currentLevel, i, requiredLevel, j, result)
				}
			})
		}
	}
}
