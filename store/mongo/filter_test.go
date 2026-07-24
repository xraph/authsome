package mongo

import (
	"regexp"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/user"
)

// TestBuildUserListFilter_EscapesEmailRegex pins that a user-supplied email
// search value is escaped before being used as a $regex, so a caller cannot
// inject regex metacharacters (ReDoS / query-semantics injection) against the
// users collection.
func TestBuildUserListFilter_EscapesEmailRegex(t *testing.T) {
	appID := id.NewAppID()

	// A value full of regex metacharacters that would be pathological if
	// interpreted as a pattern.
	malicious := "(a+)+$"
	f := buildUserListFilter(&user.Query{AppID: appID, Email: malicious})

	emailClause, ok := f["email"].(bson.M)
	if !ok {
		t.Fatalf("email clause should be a bson.M, got %T", f["email"])
	}
	got, _ := emailClause["$regex"].(string)
	want := regexp.QuoteMeta(malicious)
	if got != want {
		t.Fatalf("email regex not escaped: got %q want %q", got, want)
	}
	// Sanity: the raw metacharacters must not survive verbatim.
	if got == malicious {
		t.Fatal("raw user input must not be used as a regex verbatim")
	}
}

// TestBuildUserListFilter_ScopesAppAndEnv confirms the base scoping fields are
// always present and env is included only when set.
func TestBuildUserListFilter_ScopesAppAndEnv(t *testing.T) {
	appID := id.NewAppID()
	envID := id.NewEnvironmentID()

	f := buildUserListFilter(&user.Query{AppID: appID, EnvID: envID})
	if f["app_id"] != appID.String() {
		t.Fatalf("app_id filter missing/incorrect: %v", f["app_id"])
	}
	if _, ok := f["deleted_at"]; !ok {
		t.Fatal("deleted_at filter must be present")
	}
	if f["env_id"] != envID.String() {
		t.Fatalf("env_id filter missing/incorrect: %v", f["env_id"])
	}

	noEnv := buildUserListFilter(&user.Query{AppID: appID})
	if _, ok := noEnv["env_id"]; ok {
		t.Fatal("env_id must be omitted when not set")
	}
	if _, ok := noEnv["email"]; ok {
		t.Fatal("email filter must be omitted when empty")
	}
}
