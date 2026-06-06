package account

import (
	"testing"
	"time"

	"github.com/xraph/authsome/id"
)

func TestGenerateOTPCode(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 200; i++ {
		code, err := GenerateOTPCode()
		if err != nil {
			t.Fatalf("GenerateOTPCode error: %v", err)
		}
		if len(code) != 6 {
			t.Fatalf("expected a 6-digit code, got %q (len %d)", code, len(code))
		}
		for _, r := range code {
			if r < '0' || r > '9' {
				t.Fatalf("code %q contains a non-digit", code)
			}
		}
		seen[code] = true
	}
	// 200 draws from 1e6 should almost never collide enough to drop below 100 uniques.
	if len(seen) < 100 {
		t.Fatalf("codes not sufficiently random: %d unique of 200", len(seen))
	}
}

func TestNewEmailVerificationCode(t *testing.T) {
	appID := id.NewAppID()
	userID := id.NewUserID()

	v, err := NewEmailVerificationCode(appID, userID, 15*time.Minute)
	if err != nil {
		t.Fatalf("NewEmailVerificationCode error: %v", err)
	}
	if v.Type != VerificationEmail {
		t.Fatalf("expected type %q, got %q", VerificationEmail, v.Type)
	}
	if v.AppID != appID || v.UserID != userID {
		t.Fatalf("app/user id mismatch")
	}
	if len(v.Token) != 6 {
		t.Fatalf("expected a 6-digit code in Token, got %q", v.Token)
	}
	if v.Consumed {
		t.Fatalf("new verification should not be consumed")
	}
	if v.Attempts != 0 {
		t.Fatalf("new verification should have 0 attempts, got %d", v.Attempts)
	}
	if !v.ExpiresAt.After(time.Now()) {
		t.Fatalf("verification should expire in the future")
	}
	if v.ID.IsNil() {
		t.Fatalf("verification should have an ID")
	}
}
