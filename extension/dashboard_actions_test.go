package extension

import "testing"

func TestDashboardUserDropdownActions_LocalMode(t *testing.T) {
	e := &Extension{clientMode: false}

	actions := e.DashboardUserDropdownActions("/dashboard")

	if len(actions) != 2 {
		t.Fatalf("got %d actions, want 2 (Profile, Security)", len(actions))
	}

	wantHrefs := map[string]string{
		"Profile":  "/dashboard/ext/authsome/pages/profile",
		"Security": "/dashboard/ext/authsome/pages/security",
	}

	for _, action := range actions {
		want, ok := wantHrefs[action.Label]
		if !ok {
			t.Errorf("unexpected action label %q", action.Label)
			continue
		}

		if action.Href != want {
			t.Errorf("action %q href = %q, want %q (local mode → /ext/)", action.Label, action.Href, want)
		}
	}
}

func TestDashboardUserDropdownActions_ClientMode(t *testing.T) {
	e := &Extension{clientMode: true}

	actions := e.DashboardUserDropdownActions("/dashboard")

	if len(actions) != 2 {
		t.Fatalf("got %d actions, want 2", len(actions))
	}

	wantHrefs := map[string]string{
		"Profile":  "/dashboard/remote/authsome/pages/profile",
		"Security": "/dashboard/remote/authsome/pages/security",
	}

	for _, action := range actions {
		want, ok := wantHrefs[action.Label]
		if !ok {
			t.Errorf("unexpected action label %q", action.Label)
			continue
		}

		if action.Href != want {
			t.Errorf("action %q href = %q, want %q (client mode → /remote/)", action.Label, action.Href, want)
		}
	}
}

func TestDashboardUserDropdownActions_RespectsBasePath(t *testing.T) {
	e := &Extension{clientMode: true}

	actions := e.DashboardUserDropdownActions("/admin/dashboard")

	for _, action := range actions {
		if action.Label != "Profile" {
			continue
		}

		if want := "/admin/dashboard/remote/authsome/pages/profile"; action.Href != want {
			t.Errorf("Profile href = %q, want %q", action.Href, want)
		}
	}
}
