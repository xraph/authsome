package validator

import "testing"

func TestValidatePassword_Default(t *testing.T) {
	req := DefaultPasswordRequirements()
	if ok, _ := ValidatePassword("short", req); ok {
		t.Errorf("expected 'short' to fail default requirements")
	}
	if ok, msg := ValidatePassword("abcd1234", req); !ok {
		t.Errorf("expected 'abcd1234' to pass default requirements, got: %s", msg)
	}
}

func TestValidatePassword_Strict(t *testing.T) {
	req := PasswordRequirements{
		MinLength:      8,
		RequireUpper:   true,
		RequireLower:   true,
		RequireNumber:  true,
		RequireSpecial: true,
	}
	if ok, msg := ValidatePassword("Aa1!aaaa", req); !ok {
		t.Errorf("expected strict password to pass, got: %s", msg)
	}
	if ok, _ := ValidatePassword("Aaaaaaaa", req); ok {
		t.Errorf("expected missing number/special to fail in strict requirements")
	}
}
