package authsome

import (
	"context"
	"errors"
	"testing"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/principal"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/store"
)

// stubSessionStore records what actually reached persistence.
//
// It embeds a nil store.Store so only the two methods the decorator overrides
// need implementing; anything else this test provoked would panic loudly
// rather than silently pass through, which is the behaviour we want from a
// stub standing in for a 200-method interface.
type stubSessionStore struct {
	store.Store

	created  *session.Session
	rotated  *session.Session
	rotateOK bool
	rotateNo error
}

func (s *stubSessionStore) CreateSession(_ context.Context, sess *session.Session) error {
	s.created = sess

	return nil
}

func (s *stubSessionStore) RotateSession(
	_ context.Context, sess *session.Session, _ string,
) (bool, error) {
	s.rotated = sess

	return s.rotateOK, s.rotateNo
}

func testSession() *session.Session {
	return &session.Session{
		ID:     id.NewSessionID(),
		AppID:  id.NewAppID(),
		UserID: id.NewUserID(),
	}
}

func stampingStore(t *testing.T, stamp roleStamper) (store.Store, *stubSessionStore) {
	t.Helper()

	inner := &stubSessionStore{rotateOK: true}

	return newRoleStampingStore(inner, stamp, nil), inner
}

func TestCreateSessionStampsRoles(t *testing.T) {
	s, inner := stampingStore(t, func(context.Context, id.AppID, id.UserID) ([]string, error) {
		return []string{"admin", "owner"}, nil
	})

	sess := testSession()
	if err := s.CreateSession(context.Background(), sess); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	if got := inner.created.Roles; len(got) != 2 || got[0] != "admin" || got[1] != "owner" {
		t.Errorf("persisted roles = %v, want [admin owner]", got)
	}
}

// TestCreateSessionFailsWhenRolesCannotBeResolved pins the policy chosen in
// CreateSession: a sign-in that cannot resolve roles fails rather than
// producing a session with none.
//
// The stamp is persisted, so the alternative is not a momentary degradation.
// It is a session that authenticates for its whole lifetime and can reach no
// route declaring a role. The assertion that nothing was persisted is the
// important half: failing after the insert would leave exactly the session
// this policy exists to prevent.
func TestCreateSessionFailsWhenRolesCannotBeResolved(t *testing.T) {
	wantErr := errors.New("rbac unavailable")

	s, inner := stampingStore(t, func(context.Context, id.AppID, id.UserID) ([]string, error) {
		return nil, wantErr
	})

	err := s.CreateSession(context.Background(), testSession())
	if !errors.Is(err, wantErr) {
		t.Fatalf("CreateSession error = %v, want it to wrap %v", err, wantErr)
	}

	if inner.created != nil {
		t.Error("a session was persisted despite the role lookup failing")
	}
}

// TestCreateSessionLeavesCallerSuppliedRoles covers the escape hatch: a caller
// that resolved roles itself keeps control of what it wrote.
func TestCreateSessionLeavesCallerSuppliedRoles(t *testing.T) {
	s, inner := stampingStore(t, func(context.Context, id.AppID, id.UserID) ([]string, error) {
		t.Error("stamper ran for a session that already carried roles")

		return []string{"resolved"}, nil
	})

	sess := testSession()
	sess.Roles = []string{"impersonator"}

	if err := s.CreateSession(context.Background(), sess); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	if got := inner.created.Roles; len(got) != 1 || got[0] != "impersonator" {
		t.Errorf("persisted roles = %v, want the caller's [impersonator]", got)
	}
}

// TestCreateSessionSkipsServiceAccounts guards the one principal kind with no
// roles to look up: its UserID is the zero value, so a lookup would be both
// meaningless and, under the policy above, fatal to every service-account
// sign-in.
func TestCreateSessionSkipsServiceAccounts(t *testing.T) {
	s, inner := stampingStore(t, func(context.Context, id.AppID, id.UserID) ([]string, error) {
		t.Error("stamper ran for a service-account session")

		return nil, errors.New("should not be called")
	})

	sess := testSession()
	sess.PrincipalKind = principal.KindService
	sess.UserID = id.UserID{}

	if err := s.CreateSession(context.Background(), sess); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	if len(inner.created.Roles) != 0 {
		t.Errorf("service-account session carried roles: %v", inner.created.Roles)
	}
}

// TestRotateSessionReStampsRoles is why rotation is decorated at all: without
// it, a role granted after sign-in waits for the user to sign out and back in.
func TestRotateSessionReStampsRoles(t *testing.T) {
	s, inner := stampingStore(t, func(context.Context, id.AppID, id.UserID) ([]string, error) {
		return []string{"admin", "auditor"}, nil
	})

	sess := testSession()
	sess.Roles = []string{"admin"} // what it was issued with

	if _, err := s.RotateSession(context.Background(), sess, "old-token"); err != nil {
		t.Fatalf("RotateSession: %v", err)
	}

	if got := inner.rotated.Roles; len(got) != 2 || got[1] != "auditor" {
		t.Errorf("rotated roles = %v, want the newly granted [admin auditor]", got)
	}
}

// TestRotateSessionKeepsRolesWhenRefreshFails pins the other half of the
// policy pair. Rotation has a good fallback that CreateSession does not: roles
// that resolved successfully at issue time. Refusing the rotation would sign
// the user out over a transient lookup failure, so it continues with those.
func TestRotateSessionKeepsRolesWhenRefreshFails(t *testing.T) {
	s, inner := stampingStore(t, func(context.Context, id.AppID, id.UserID) ([]string, error) {
		return nil, errors.New("rbac unavailable")
	})

	sess := testSession()
	sess.Roles = []string{"admin"}

	ok, err := s.RotateSession(context.Background(), sess, "old-token")
	if err != nil {
		t.Fatalf("RotateSession returned %v, want the rotation to proceed", err)
	}

	if !ok {
		t.Error("rotation reported failure when only the role refresh failed")
	}

	if got := inner.rotated.Roles; len(got) != 1 || got[0] != "admin" {
		t.Errorf("rotated roles = %v, want the previously stamped [admin]", got)
	}
}
