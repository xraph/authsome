package contract

import (
	"context"
	"errors"
	"testing"

	"github.com/xraph/forge/extensions/dashboard/contract"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/session"
)

// The dashboard's own auth check turns an agent token away, but the create
// handler also refuses an agent session on its context, so a dashboard wired
// to some other auth checker still mints no key for it.
func TestAPIKeysCreate_RefusesAgentSession(t *testing.T) {
	ctx := middleware.WithSession(context.Background(), &session.Session{
		ID: id.NewSessionID(), UserID: id.NewUserID(),
		PrincipalKind: session.PrincipalKindAgent, AgentID: id.NewAgentID(), GrantID: id.NewAgentGrantID(),
	})
	h := apikeysCreateHandler(Deps{Engine: nil})
	_, err := h(ctx, CreateAPIKeyInput{Name: "k", UserID: id.NewUserID().String()}, contract.Principal{})

	var ce *contract.Error
	if !errors.As(err, &ce) || ce.Code != contract.CodePermissionDenied {
		t.Fatalf("err = %v, want PERMISSION_DENIED", err)
	}
}
