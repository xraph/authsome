package wardenseed

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const testAppID = "aapp_01jf0000000000000000000000"

// TestEmbed_Shared_Parses verifies the embedded shared/*.warden files parse
// and resolve cleanly with ${APP_ID} substitution.
func TestEmbed_Shared_Parses(t *testing.T) {
	prog, err := Load(SharedSource(), LoadOptions{AppID: testAppID})
	require.NoError(t, err)
	require.NotNil(t, prog)
	require.Equal(t, testAppID, prog.Tenant, "tenant should be substituted from ${APP_ID}")
	require.NotEmpty(t, prog.Permissions, "shared catalog should declare permissions")
	require.NotEmpty(t, prog.Roles, "shared app.warden should declare roles")
}

// TestEmbed_Platform_Parses verifies the embedded platform/*.warden files
// parse and resolve cleanly.
func TestEmbed_Platform_Parses(t *testing.T) {
	prog, err := Load(PlatformSource(), LoadOptions{AppID: testAppID})
	require.NoError(t, err)
	require.NotNil(t, prog)
	require.NotEmpty(t, prog.Roles, "platform.warden should declare roles")
}

// TestEmbed_Shared_RoleSlugs sanity-checks that the canonical role slugs
// authsome relies on are present in the shared program.
func TestEmbed_Shared_RoleSlugs(t *testing.T) {
	prog, err := Load(SharedSource(), LoadOptions{AppID: testAppID})
	require.NoError(t, err)

	want := map[string]bool{"user": false, "admin": false, "owner": false}
	for _, r := range prog.Roles {
		if _, ok := want[r.Slug]; ok {
			want[r.Slug] = true
		}
	}
	for slug, found := range want {
		require.Truef(t, found, "shared program is missing role %q", slug)
	}
}

// TestEmbed_Platform_RoleSlugs sanity-checks the platform-only roles.
func TestEmbed_Platform_RoleSlugs(t *testing.T) {
	prog, err := Load(PlatformSource(), LoadOptions{AppID: testAppID})
	require.NoError(t, err)

	want := map[string]bool{"platform-user": false, "platform-admin": false, "platform-owner": false}
	for _, r := range prog.Roles {
		if _, ok := want[r.Slug]; ok {
			want[r.Slug] = true
		}
	}
	for slug, found := range want {
		require.Truef(t, found, "platform program is missing role %q", slug)
	}
}

// TestLoad_RejectsEmptySource fails fast when no source is provided.
func TestLoad_RejectsEmptySource(t *testing.T) {
	_, err := Load(Source{}, LoadOptions{AppID: testAppID})
	require.Error(t, err)
}

// TestLoad_MissingDirReportsCleanError surfaces a readable error when an
// override directory is set but does not exist.
func TestLoad_MissingDirReportsCleanError(t *testing.T) {
	_, err := Load(Source{Dir: "/nonexistent/path/should/not/exist/__abc"}, LoadOptions{AppID: testAppID})
	require.Error(t, err)
	require.Contains(t, err.Error(), "does not exist")
}
