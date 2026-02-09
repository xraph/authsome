package deviceflow

import (
	"strings"
	"testing"
)

func TestCodeGenerator_GenerateDeviceCode(t *testing.T) {
	gen := NewCodeGenerator(8, "XXXX-XXXX")

	// Test that device code is generated
	code1, err := gen.GenerateDeviceCode()
	if err != nil {
		t.Fatalf("GenerateDeviceCode() failed: %v", err)
	}

	if code1 == "" {
		t.Error("GenerateDeviceCode() returned empty string")
	}

	// Test that device codes are unique
	code2, err := gen.GenerateDeviceCode()
	if err != nil {
		t.Fatalf("GenerateDeviceCode() failed: %v", err)
	}

	if code1 == code2 {
		t.Error("GenerateDeviceCode() returned duplicate codes")
	}

	// Test device code length (should be URL-safe base64 of 32 bytes)
	// 32 bytes = 43 characters in base64 (without padding)
	if len(code1) < 40 {
		t.Errorf("GenerateDeviceCode() returned code too short: %d characters", len(code1))
	}
}

func TestCodeGenerator_GenerateUserCode(t *testing.T) {
	tests := []struct {
		name           string
		userCodeLength int
		userCodeFormat string
		wantLen        int
		wantFormat     bool
	}{
		{
			name:           "default format",
			userCodeLength: 8,
			userCodeFormat: "XXXX-XXXX",
			wantLen:        9, // 8 chars + 1 hyphen
			wantFormat:     true,
		},
		{
			name:           "no hyphen",
			userCodeLength: 6,
			userCodeFormat: "XXXXXX",
			wantLen:        6,
			wantFormat:     false,
		},
		{
			name:           "multiple hyphens",
			userCodeLength: 10,
			userCodeFormat: "XXX-XXX-XXXX",
			wantLen:        12, // 10 chars + 2 hyphens
			wantFormat:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := NewCodeGenerator(tt.userCodeLength, tt.userCodeFormat)

			code, err := gen.GenerateUserCode()
			if err != nil {
				t.Fatalf("GenerateUserCode() failed: %v", err)
			}

			if len(code) != tt.wantLen {
				t.Errorf("GenerateUserCode() length = %d, want %d (code: %s)", len(code), tt.wantLen, code)
			}

			// Verify format if hyphen is expected
			if tt.wantFormat && !strings.Contains(code, "-") {
				t.Error("GenerateUserCode() missing expected hyphen")
			}

			// Verify all characters are from the allowed charset
			allowedChars := "BCDFGHJKLMNPQRSTVWXZ-"
			for _, ch := range code {
				if !strings.ContainsRune(allowedChars, ch) {
					t.Errorf("GenerateUserCode() contains invalid character: %c", ch)
				}
			}
		})
	}
}

func TestCodeGenerator_GenerateUserCode_Uniqueness(t *testing.T) {
	gen := NewCodeGenerator(8, "XXXX-XXXX")
	codes := make(map[string]bool)

	// Generate 100 codes and check for duplicates
	for range 100 {
		code, err := gen.GenerateUserCode()
		if err != nil {
			t.Fatalf("GenerateUserCode() failed: %v", err)
		}

		if codes[code] {
			t.Errorf("GenerateUserCode() generated duplicate code: %s", code)
		}

		codes[code] = true
	}
}

func TestValidateUserCodeFormat(t *testing.T) {
	tests := []struct {
		name    string
		format  string
		wantErr bool
	}{
		{
			name:    "valid format",
			format:  "XXXX-XXXX",
			wantErr: false,
		},
		{
			name:    "valid format no hyphen",
			format:  "XXXXXX",
			wantErr: false,
		},
		{
			name:    "valid format with spaces",
			format:  "XXXX",
			wantErr: false,
		},
		{
			name:    "too few X's",
			format:  "XXX-XX",
			wantErr: true,
		},
		{
			name:    "too many X's",
			format:  "XXXX-XXXX-XXXXX",
			wantErr: true,
		},
		{
			name:    "empty format",
			format:  "",
			wantErr: true,
		},
		{
			name:    "invalid characters",
			format:  "XXXX@XXXX",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUserCodeFormat(tt.format)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUserCodeFormat() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
