package dpop_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/xraph/authsome/dpop"
)

func TestParseMode(t *testing.T) {
	cases := map[string]dpop.Mode{
		"":         dpop.ModeOff,
		"off":      dpop.ModeOff,
		"optional": dpop.ModeOptional,
		"required": dpop.ModeRequired,
		"REQUIRED": dpop.ModeRequired,
		"nonsense": dpop.ModeOff,
	}
	for in, want := range cases {
		assert.Equal(t, want, dpop.ParseMode(in), "input %q", in)
	}
}

// TestMaxMode: a per-client setting can never weaken an app-level mandate. If
// an app is required, no client value brings it back down.
func TestMaxMode(t *testing.T) {
	cases := []struct {
		app, client, want dpop.Mode
	}{
		{dpop.ModeOff, dpop.ModeOff, dpop.ModeOff},
		{dpop.ModeOff, dpop.ModeOptional, dpop.ModeOptional},
		{dpop.ModeOff, dpop.ModeRequired, dpop.ModeRequired},
		{dpop.ModeOptional, dpop.ModeOff, dpop.ModeOptional},
		{dpop.ModeOptional, dpop.ModeOptional, dpop.ModeOptional},
		// The client-strengthens-above-app case: the direction an operator is
		// most likely to actually use, since a per-client override that raises
		// the bar is the whole point of the setting.
		{dpop.ModeOptional, dpop.ModeRequired, dpop.ModeRequired},
		{dpop.ModeRequired, dpop.ModeOff, dpop.ModeRequired},
		{dpop.ModeRequired, dpop.ModeOptional, dpop.ModeRequired},
		{dpop.ModeRequired, dpop.ModeRequired, dpop.ModeRequired},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, dpop.MaxMode(tc.app, tc.client),
			"app=%s client=%s", tc.app, tc.client)
	}
}
