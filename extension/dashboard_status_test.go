package extension

import (
	"testing"

	authsome "github.com/xraph/authsome"
	authclient "github.com/xraph/authsome/sdk/go"
)

func TestDashboardStatus(t *testing.T) {
	tests := []struct {
		name       string
		ext        *Extension
		configured bool
	}{
		{"engine mode, no engine yet", &Extension{}, false},
		{"engine mode, engine built", &Extension{engine: &authsome.Engine{}}, true},
		{"client mode, no client", &Extension{clientMode: true}, false},
		{"client mode, client built", &Extension{clientMode: true, client: &authclient.Client{}}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.ext.DashboardStatus()
			if got.Version != ExtensionVersion {
				t.Errorf("version = %q, want %q", got.Version, ExtensionVersion)
			}
			if got.Configured != tt.configured {
				t.Errorf("configured = %v, want %v", got.Configured, tt.configured)
			}
			if !got.Configured && got.Message == "" {
				t.Error("an unconfigured status should say why")
			}
		})
	}
}
