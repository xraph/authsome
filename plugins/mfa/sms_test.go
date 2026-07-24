package mfa

import (
	"testing"
	"time"
)

func TestValidateSMSCode(t *testing.T) {
	valid := &SMSChallenge{Code: "123456", ExpiresAt: time.Now().Add(time.Minute)}
	expired := &SMSChallenge{Code: "123456", ExpiresAt: time.Now().Add(-time.Minute)}

	cases := []struct {
		name      string
		code      string
		challenge *SMSChallenge
		want      bool
	}{
		{"correct code", "123456", valid, true},
		{"wrong code", "000000", valid, false},
		{"empty code", "", valid, false},
		{"expired challenge", "123456", expired, false},
		{"nil challenge", "123456", nil, false},
		{"length mismatch", "12345", valid, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ValidateSMSCode(tc.code, tc.challenge); got != tc.want {
				t.Fatalf("ValidateSMSCode(%q) = %v, want %v", tc.code, got, tc.want)
			}
		})
	}
}
