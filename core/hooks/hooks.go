package hooks

import (
	"context"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/auth"
	"github.com/xraph/authsome/core/responses"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
)

// HookRegistry manages all hooks for the authentication system
type HookRegistry struct {
	// User hooks
	beforeUserCreate []BeforeUserCreateHook
	afterUserCreate  []AfterUserCreateHook
	beforeUserUpdate []BeforeUserUpdateHook
	afterUserUpdate  []AfterUserUpdateHook
	beforeUserDelete []BeforeUserDeleteHook
	afterUserDelete  []AfterUserDeleteHook

	// Session hooks
	beforeSessionCreate []BeforeSessionCreateHook
	afterSessionCreate  []AfterSessionCreateHook
	beforeSessionRevoke []BeforeSessionRevokeHook
	afterSessionRevoke  []AfterSessionRevokeHook

	// Auth hooks
	beforeSignUp  []BeforeSignUpHook
	afterSignUp   []AfterSignUpHook
	beforeSignIn  []BeforeSignInHook
	afterSignIn   []AfterSignInHook
	beforeSignOut []BeforeSignOutHook
	afterSignOut  []AfterSignOutHook

	// Organization hooks (for multi-tenancy plugin)
	beforeOrganizationCreate []BeforeOrganizationCreateHook
	afterOrganizationCreate  []AfterOrganizationCreateHook
	beforeMemberAdd          []BeforeMemberAddHook
	afterMemberAdd           []AfterMemberAddHook

	// App hooks (for multi-app support)
	beforeAppCreate []BeforeAppCreateHook
	afterAppCreate  []AfterAppCreateHook
}

// Hook function types

// User hooks
type BeforeUserCreateHook func(ctx context.Context, req *user.CreateUserRequest) error
type AfterUserCreateHook func(ctx context.Context, user *user.User) error
type BeforeUserUpdateHook func(ctx context.Context, userID xid.ID, req *user.UpdateUserRequest) error
type AfterUserUpdateHook func(ctx context.Context, user *user.User) error
type BeforeUserDeleteHook func(ctx context.Context, userID xid.ID) error
type AfterUserDeleteHook func(ctx context.Context, userID xid.ID) error

// Session hooks
type BeforeSessionCreateHook func(ctx context.Context, req *session.CreateSessionRequest) error
type AfterSessionCreateHook func(ctx context.Context, session *session.Session) error
type BeforeSessionRevokeHook func(ctx context.Context, token string) error
type AfterSessionRevokeHook func(ctx context.Context, sessionID xid.ID) error

// Auth hooks
type BeforeSignUpHook func(ctx context.Context, req *auth.SignUpRequest) error
type AfterSignUpHook func(ctx context.Context, response *responses.AuthResponse) error
type BeforeSignInHook func(ctx context.Context, req *auth.SignInRequest) error
type AfterSignInHook func(ctx context.Context, response *responses.AuthResponse) error
type BeforeSignOutHook func(ctx context.Context, token string) error
type AfterSignOutHook func(ctx context.Context, token string) error

// Organization hooks (for multi-tenancy plugin)
type BeforeOrganizationCreateHook func(ctx context.Context, req interface{}) error
type AfterOrganizationCreateHook func(ctx context.Context, org interface{}) error
type BeforeMemberAddHook func(ctx context.Context, orgID string, userID xid.ID) error
type AfterMemberAddHook func(ctx context.Context, member interface{}) error

// App hooks (for multi-app support)
type BeforeAppCreateHook func(ctx context.Context, req interface{}) error
type AfterAppCreateHook func(ctx context.Context, app interface{}) error

// NewHookRegistry creates a new hook registry
func NewHookRegistry() *HookRegistry {
	return &HookRegistry{}
}

// User hook registration methods
func (h *HookRegistry) RegisterBeforeUserCreate(hook BeforeUserCreateHook) {
	h.beforeUserCreate = append(h.beforeUserCreate, hook)
}

func (h *HookRegistry) RegisterAfterUserCreate(hook AfterUserCreateHook) {
	h.afterUserCreate = append(h.afterUserCreate, hook)
}

func (h *HookRegistry) RegisterBeforeUserUpdate(hook BeforeUserUpdateHook) {
	h.beforeUserUpdate = append(h.beforeUserUpdate, hook)
}

func (h *HookRegistry) RegisterAfterUserUpdate(hook AfterUserUpdateHook) {
	h.afterUserUpdate = append(h.afterUserUpdate, hook)
}

func (h *HookRegistry) RegisterBeforeUserDelete(hook BeforeUserDeleteHook) {
	h.beforeUserDelete = append(h.beforeUserDelete, hook)
}

func (h *HookRegistry) RegisterAfterUserDelete(hook AfterUserDeleteHook) {
	h.afterUserDelete = append(h.afterUserDelete, hook)
}

// Session hook registration methods
func (h *HookRegistry) RegisterBeforeSessionCreate(hook BeforeSessionCreateHook) {
	h.beforeSessionCreate = append(h.beforeSessionCreate, hook)
}

func (h *HookRegistry) RegisterAfterSessionCreate(hook AfterSessionCreateHook) {
	h.afterSessionCreate = append(h.afterSessionCreate, hook)
}

func (h *HookRegistry) RegisterBeforeSessionRevoke(hook BeforeSessionRevokeHook) {
	h.beforeSessionRevoke = append(h.beforeSessionRevoke, hook)
}

func (h *HookRegistry) RegisterAfterSessionRevoke(hook AfterSessionRevokeHook) {
	h.afterSessionRevoke = append(h.afterSessionRevoke, hook)
}

// Auth hook registration methods
func (h *HookRegistry) RegisterBeforeSignUp(hook BeforeSignUpHook) {
	h.beforeSignUp = append(h.beforeSignUp, hook)
}

func (h *HookRegistry) RegisterAfterSignUp(hook AfterSignUpHook) {
	h.afterSignUp = append(h.afterSignUp, hook)
}

func (h *HookRegistry) RegisterBeforeSignIn(hook BeforeSignInHook) {
	h.beforeSignIn = append(h.beforeSignIn, hook)
}

func (h *HookRegistry) RegisterAfterSignIn(hook AfterSignInHook) {
	h.afterSignIn = append(h.afterSignIn, hook)
}

func (h *HookRegistry) RegisterBeforeSignOut(hook BeforeSignOutHook) {
	h.beforeSignOut = append(h.beforeSignOut, hook)
}

func (h *HookRegistry) RegisterAfterSignOut(hook AfterSignOutHook) {
	h.afterSignOut = append(h.afterSignOut, hook)
}

// Organization hook registration methods (for multi-tenancy plugin)
func (h *HookRegistry) RegisterBeforeOrganizationCreate(hook BeforeOrganizationCreateHook) {
	h.beforeOrganizationCreate = append(h.beforeOrganizationCreate, hook)
}

func (h *HookRegistry) RegisterAfterOrganizationCreate(hook AfterOrganizationCreateHook) {
	h.afterOrganizationCreate = append(h.afterOrganizationCreate, hook)
}

func (h *HookRegistry) RegisterBeforeMemberAdd(hook BeforeMemberAddHook) {
	h.beforeMemberAdd = append(h.beforeMemberAdd, hook)
}

func (h *HookRegistry) RegisterAfterMemberAdd(hook AfterMemberAddHook) {
	h.afterMemberAdd = append(h.afterMemberAdd, hook)
}

// App hook registration methods (for multi-app support)
func (h *HookRegistry) RegisterBeforeAppCreate(hook BeforeAppCreateHook) {
	h.beforeAppCreate = append(h.beforeAppCreate, hook)
}

func (h *HookRegistry) RegisterAfterAppCreate(hook AfterAppCreateHook) {
	h.afterAppCreate = append(h.afterAppCreate, hook)
}

// Hook execution methods

// ExecuteBeforeUserCreate executes all before user create hooks
func (h *HookRegistry) ExecuteBeforeUserCreate(ctx context.Context, req *user.CreateUserRequest) error {
	for _, hook := range h.beforeUserCreate {
		if err := hook(ctx, req); err != nil {
			return err
		}
	}
	return nil
}

// ExecuteAfterUserCreate executes all after user create hooks
func (h *HookRegistry) ExecuteAfterUserCreate(ctx context.Context, user *user.User) error {
	for _, hook := range h.afterUserCreate {
		if err := hook(ctx, user); err != nil {
			return err
		}
	}
	return nil
}

// ExecuteBeforeUserUpdate executes all before user update hooks
func (h *HookRegistry) ExecuteBeforeUserUpdate(ctx context.Context, userID xid.ID, req *user.UpdateUserRequest) error {
	for _, hook := range h.beforeUserUpdate {
		if err := hook(ctx, userID, req); err != nil {
			return err
		}
	}
	return nil
}

// ExecuteAfterUserUpdate executes all after user update hooks
func (h *HookRegistry) ExecuteAfterUserUpdate(ctx context.Context, user *user.User) error {
	for _, hook := range h.afterUserUpdate {
		if err := hook(ctx, user); err != nil {
			return err
		}
	}
	return nil
}

// ExecuteBeforeUserDelete executes all before user delete hooks
func (h *HookRegistry) ExecuteBeforeUserDelete(ctx context.Context, userID xid.ID) error {
	for _, hook := range h.beforeUserDelete {
		if err := hook(ctx, userID); err != nil {
			return err
		}
	}
	return nil
}

// ExecuteAfterUserDelete executes all after user delete hooks
func (h *HookRegistry) ExecuteAfterUserDelete(ctx context.Context, userID xid.ID) error {
	for _, hook := range h.afterUserDelete {
		if err := hook(ctx, userID); err != nil {
			return err
		}
	}
	return nil
}

// ExecuteBeforeSessionCreate executes all before session create hooks
func (h *HookRegistry) ExecuteBeforeSessionCreate(ctx context.Context, req *session.CreateSessionRequest) error {
	for _, hook := range h.beforeSessionCreate {
		if err := hook(ctx, req); err != nil {
			return err
		}
	}
	return nil
}

// ExecuteAfterSessionCreate executes all after session create hooks
func (h *HookRegistry) ExecuteAfterSessionCreate(ctx context.Context, session *session.Session) error {
	for _, hook := range h.afterSessionCreate {
		if err := hook(ctx, session); err != nil {
			return err
		}
	}
	return nil
}

// ExecuteBeforeSessionRevoke executes all before session revoke hooks
func (h *HookRegistry) ExecuteBeforeSessionRevoke(ctx context.Context, token string) error {
	for _, hook := range h.beforeSessionRevoke {
		if err := hook(ctx, token); err != nil {
			return err
		}
	}
	return nil
}

// ExecuteAfterSessionRevoke executes all after session revoke hooks
func (h *HookRegistry) ExecuteAfterSessionRevoke(ctx context.Context, sessionID xid.ID) error {
	for _, hook := range h.afterSessionRevoke {
		if err := hook(ctx, sessionID); err != nil {
			return err
		}
	}
	return nil
}

// ExecuteBeforeSignUp executes all before sign up hooks
func (h *HookRegistry) ExecuteBeforeSignUp(ctx context.Context, req *auth.SignUpRequest) error {
	for _, hook := range h.beforeSignUp {
		if err := hook(ctx, req); err != nil {
			return err
		}
	}
	return nil
}

// ExecuteAfterSignUp executes all after sign up hooks
func (h *HookRegistry) ExecuteAfterSignUp(ctx context.Context, response *responses.AuthResponse) error {
	for _, hook := range h.afterSignUp {
		if err := hook(ctx, response); err != nil {
			return err
		}
	}
	return nil
}

// ExecuteBeforeSignIn executes all before sign in hooks
func (h *HookRegistry) ExecuteBeforeSignIn(ctx context.Context, req *auth.SignInRequest) error {
	for _, hook := range h.beforeSignIn {
		if err := hook(ctx, req); err != nil {
			return err
		}
	}
	return nil
}

// ExecuteAfterSignIn executes all after sign in hooks
func (h *HookRegistry) ExecuteAfterSignIn(ctx context.Context, response *responses.AuthResponse) error {
	for _, hook := range h.afterSignIn {
		if err := hook(ctx, response); err != nil {
			return err
		}
	}
	return nil
}

// ExecuteBeforeSignOut executes all before sign out hooks
func (h *HookRegistry) ExecuteBeforeSignOut(ctx context.Context, token string) error {
	for _, hook := range h.beforeSignOut {
		if err := hook(ctx, token); err != nil {
			return err
		}
	}
	return nil
}

// ExecuteAfterSignOut executes all after sign out hooks
func (h *HookRegistry) ExecuteAfterSignOut(ctx context.Context, token string) error {
	for _, hook := range h.afterSignOut {
		if err := hook(ctx, token); err != nil {
			return err
		}
	}
	return nil
}

// ExecuteBeforeOrganizationCreate executes all before organization create hooks
func (h *HookRegistry) ExecuteBeforeOrganizationCreate(ctx context.Context, req interface{}) error {
	for _, hook := range h.beforeOrganizationCreate {
		if err := hook(ctx, req); err != nil {
			return err
		}
	}
	return nil
}

// ExecuteAfterOrganizationCreate executes all after organization create hooks
func (h *HookRegistry) ExecuteAfterOrganizationCreate(ctx context.Context, org interface{}) error {
	for _, hook := range h.afterOrganizationCreate {
		if err := hook(ctx, org); err != nil {
			return err
		}
	}
	return nil
}

// ExecuteBeforeMemberAdd executes all before member add hooks
func (h *HookRegistry) ExecuteBeforeMemberAdd(ctx context.Context, orgID string, userID xid.ID) error {
	for _, hook := range h.beforeMemberAdd {
		if err := hook(ctx, orgID, userID); err != nil {
			return err
		}
	}
	return nil
}

// ExecuteAfterMemberAdd executes all after member add hooks
func (h *HookRegistry) ExecuteAfterMemberAdd(ctx context.Context, member interface{}) error {
	for _, hook := range h.afterMemberAdd {
		if err := hook(ctx, member); err != nil {
			return err
		}
	}
	return nil
}

// ExecuteBeforeAppCreate executes all before app create hooks
func (h *HookRegistry) ExecuteBeforeAppCreate(ctx context.Context, req interface{}) error {
	for _, hook := range h.beforeAppCreate {
		if err := hook(ctx, req); err != nil {
			return err
		}
	}
	return nil
}

// ExecuteAfterAppCreate executes all after app create hooks
func (h *HookRegistry) ExecuteAfterAppCreate(ctx context.Context, app interface{}) error {
	for _, hook := range h.afterAppCreate {
		if err := hook(ctx, app); err != nil {
			return err
		}
	}
	return nil
}
