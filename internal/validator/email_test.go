package validator

import "testing"

func TestValidateEmail(t *testing.T) {
	valid := []string{
		"user@example.com",
		"USER+tag@Example.co",
		"a.b_c-d+1@sub.example.org",
	}
	invalid := []string{
		"",
		"user@",
		"user@domain",
		"user@.com",
		"userdomain.com",
	}
	for _, e := range valid {
		if !ValidateEmail(e) {
			t.Errorf("expected valid email: %s", e)
		}
	}
	for _, e := range invalid {
		if ValidateEmail(e) {
			t.Errorf("expected invalid email: %s", e)
		}
	}
}
