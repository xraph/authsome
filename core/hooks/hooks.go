package hooks

import (
	"context"
	"log"
	"sync"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/auth"
	"github.com/xraph/authsome/core/responses"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
)

// =============================================================================
// Hook Context Access Pattern
// =============================================================================
//
// All hooks receive a context.Context that contains AuthContext with complete
// authentication state. Hooks should extract it using:
//
//   authCtx, ok := contexts.GetAuthContext(ctx)
//   if !ok || authCtx == nil {
//       // Handle missing context - typically log warning and return nil
//       return nil
//   }
//
// AuthContext provides:
//   - authCtx.User / Session          - Current user and session
//   - authCtx.AppID / EnvironmentID   - App context
//   - authCtx.OrganizationID          - Organization scope
//   - authCtx.IPAddress / UserAgent   - Security metadata
//   - authCtx.APIKey / Scopes         - API key auth (if present)
//   - authCtx.UserRoles / Permissions - RBAC data
//
// For after-auth hooks (AfterSignIn, AfterSignUp), AuthContext is freshly
// populated with the newly created session/user immediately before hook execution.
//
// Example:
//
//   hookRegistry.RegisterAfterSignIn(func(ctx context.Context, response *responses.AuthResponse) error {
//       authCtx, ok := contexts.GetAuthContext(ctx)
//       if !ok || authCtx == nil {
//           return nil // Context not available
//       }
//
//       // Access complete auth state
//       appID := authCtx.AppID
//       ipAddress := authCtx.IPAddress
//       userAgent := authCtx.UserAgent
//
//       // Your hook logic here...
//       return nil
//   })
//
// =============================================================================

// HookRegistry manages all hooks for the authentication system.
// It is thread-safe and timing-independent - hooks can be registered
// at any point in the application lifecycle and will always be executed.
type HookRegistry struct {
	mu sync.RWMutex // Protects all hook slices for thread-safety

	// Debug mode for verbose logging
	debug bool

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
	afterOrganizationUpdate  []AfterOrganizationUpdateHook
	afterOrganizationDelete  []AfterOrganizationDeleteHook
	beforeMemberAdd          []BeforeMemberAddHook
	afterMemberAdd           []AfterMemberAddHook
	afterMemberRemove        []AfterMemberRemoveHook
	afterMemberRoleChange    []AfterMemberRoleChangeHook

	// App hooks (for multi-app support)
	beforeAppCreate []BeforeAppCreateHook
	afterAppCreate  []AfterAppCreateHook

	// Permission hooks (for permissions plugin)
	beforePermissionEvaluate []BeforePermissionEvaluateHook
	afterPermissionEvaluate  []AfterPermissionEvaluateHook
	onPolicyChange           []OnPolicyChangeHook
	onCacheInvalidate        []OnCacheInvalidateHook

	// Device/Session security hooks
	onNewDeviceDetected       []OnNewDeviceDetectedHook
	onNewLocationDetected     []OnNewLocationDetectedHook
	onSuspiciousLoginDetected []OnSuspiciousLoginDetectedHook
	onDeviceRemoved           []OnDeviceRemovedHook
	onAllSessionsRevoked      []OnAllSessionsRevokedHook

	// Account lifecycle hooks
	onEmailChangeRequest []OnEmailChangeRequestHook
	onEmailChanged       []OnEmailChangedHook
	onPasswordChanged    []OnPasswordChangedHook
	onUsernameChanged    []OnUsernameChangedHook
	onAccountDeleted     []OnAccountDeletedHook
	onAccountSuspended   []OnAccountSuspendedHook
	onAccountReactivated []OnAccountReactivatedHook
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
type AfterOrganizationUpdateHook func(ctx context.Context, org interface{}) error
type AfterOrganizationDeleteHook func(ctx context.Context, orgID xid.ID, orgName string) error
type BeforeMemberAddHook func(ctx context.Context, orgID string, userID xid.ID) error
type AfterMemberAddHook func(ctx context.Context, member interface{}) error
type AfterMemberRemoveHook func(ctx context.Context, orgID xid.ID, userID xid.ID, memberName string) error
type AfterMemberRoleChangeHook func(ctx context.Context, orgID xid.ID, userID xid.ID, oldRole string, newRole string) error

// App hooks (for multi-app support)
type BeforeAppCreateHook func(ctx context.Context, req interface{}) error
type AfterAppCreateHook func(ctx context.Context, app interface{}) error

// Permission hooks (for permissions plugin)
// PermissionEvaluateRequest is passed to before/after permission evaluate hooks
type PermissionEvaluateRequest struct {
	UserID       xid.ID
	ResourceType string
	ResourceID   string
	Action       string
	Context      map[string]interface{}
}

// PermissionDecision is passed to after permission evaluate hook
type PermissionDecision struct {
	Allowed          bool
	MatchedPolicies  []string
	EvaluationTimeMs float64
	CacheHit         bool
	Error            string
}

type BeforePermissionEvaluateHook func(ctx context.Context, req *PermissionEvaluateRequest) error
type AfterPermissionEvaluateHook func(ctx context.Context, req *PermissionEvaluateRequest, decision *PermissionDecision) error
type OnPolicyChangeHook func(ctx context.Context, policyID xid.ID, action string) error
type OnCacheInvalidateHook func(ctx context.Context, scope string, id xid.ID) error

// Device/Session security hooks
type OnNewDeviceDetectedHook func(ctx context.Context, userID xid.ID, deviceName, location, ipAddress string) error
type OnNewLocationDetectedHook func(ctx context.Context, userID xid.ID, location, ipAddress string) error
type OnSuspiciousLoginDetectedHook func(ctx context.Context, userID xid.ID, reason, location, ipAddress string) error
type OnDeviceRemovedHook func(ctx context.Context, userID xid.ID, deviceName string) error
type OnAllSessionsRevokedHook func(ctx context.Context, userID xid.ID) error

// Account lifecycle hooks
type OnEmailChangeRequestHook func(ctx context.Context, userID xid.ID, oldEmail, newEmail, confirmationUrl string) error
type OnEmailChangedHook func(ctx context.Context, userID xid.ID, oldEmail, newEmail string) error
type OnPasswordChangedHook func(ctx context.Context, userID xid.ID) error
type OnUsernameChangedHook func(ctx context.Context, userID xid.ID, oldUsername, newUsername string) error
type OnAccountDeletedHook func(ctx context.Context, userID xid.ID) error
type OnAccountSuspendedHook func(ctx context.Context, userID xid.ID, reason string) error
type OnAccountReactivatedHook func(ctx context.Context, userID xid.ID) error

// NewHookRegistry creates a new hook registry
func NewHookRegistry() *HookRegistry {
	return &HookRegistry{
		debug: false, // Can be enabled via EnableDebug()
	}
}

// EnableDebug enables verbose debug logging for hook registration and execution
func (h *HookRegistry) EnableDebug() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.debug = true
	log.Println("[HookRegistry] Debug mode enabled")
}

// DisableDebug disables debug logging
func (h *HookRegistry) DisableDebug() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.debug = false
}

// GetHookCounts returns the count of registered hooks for diagnostics
func (h *HookRegistry) GetHookCounts() map[string]int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return map[string]int{
		"beforeUserCreate":          len(h.beforeUserCreate),
		"afterUserCreate":           len(h.afterUserCreate),
		"beforeUserUpdate":          len(h.beforeUserUpdate),
		"afterUserUpdate":           len(h.afterUserUpdate),
		"beforeUserDelete":          len(h.beforeUserDelete),
		"afterUserDelete":           len(h.afterUserDelete),
		"beforeSessionCreate":       len(h.beforeSessionCreate),
		"afterSessionCreate":        len(h.afterSessionCreate),
		"beforeSessionRevoke":       len(h.beforeSessionRevoke),
		"afterSessionRevoke":        len(h.afterSessionRevoke),
		"beforeSignUp":              len(h.beforeSignUp),
		"afterSignUp":               len(h.afterSignUp),
		"beforeSignIn":              len(h.beforeSignIn),
		"afterSignIn":               len(h.afterSignIn),
		"beforeSignOut":             len(h.beforeSignOut),
		"afterSignOut":              len(h.afterSignOut),
		"beforeOrganizationCreate":  len(h.beforeOrganizationCreate),
		"afterOrganizationCreate":   len(h.afterOrganizationCreate),
		"afterOrganizationUpdate":   len(h.afterOrganizationUpdate),
		"afterOrganizationDelete":   len(h.afterOrganizationDelete),
		"beforeMemberAdd":           len(h.beforeMemberAdd),
		"afterMemberAdd":            len(h.afterMemberAdd),
		"afterMemberRemove":         len(h.afterMemberRemove),
		"afterMemberRoleChange":     len(h.afterMemberRoleChange),
		"beforeAppCreate":           len(h.beforeAppCreate),
		"afterAppCreate":            len(h.afterAppCreate),
		"beforePermissionEvaluate":  len(h.beforePermissionEvaluate),
		"afterPermissionEvaluate":   len(h.afterPermissionEvaluate),
		"onPolicyChange":            len(h.onPolicyChange),
		"onCacheInvalidate":         len(h.onCacheInvalidate),
		"onNewDeviceDetected":       len(h.onNewDeviceDetected),
		"onNewLocationDetected":     len(h.onNewLocationDetected),
		"onSuspiciousLoginDetected": len(h.onSuspiciousLoginDetected),
		"onDeviceRemoved":           len(h.onDeviceRemoved),
		"onAllSessionsRevoked":      len(h.onAllSessionsRevoked),
		"onEmailChangeRequest":      len(h.onEmailChangeRequest),
		"onEmailChanged":            len(h.onEmailChanged),
		"onPasswordChanged":         len(h.onPasswordChanged),
		"onUsernameChanged":         len(h.onUsernameChanged),
		"onAccountDeleted":          len(h.onAccountDeleted),
		"onAccountSuspended":        len(h.onAccountSuspended),
		"onAccountReactivated":      len(h.onAccountReactivated),
	}
}

// User hook registration methods
func (h *HookRegistry) RegisterBeforeUserCreate(hook BeforeUserCreateHook) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.beforeUserCreate = append(h.beforeUserCreate, hook)
	if h.debug {
		log.Printf("[HookRegistry] Registered BeforeUserCreate hook (total: %d)", len(h.beforeUserCreate))
	}
}

func (h *HookRegistry) RegisterAfterUserCreate(hook AfterUserCreateHook) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.afterUserCreate = append(h.afterUserCreate, hook)
	if h.debug {
		log.Printf("[HookRegistry] Registered AfterUserCreate hook (total: %d)", len(h.afterUserCreate))
	}
}

func (h *HookRegistry) RegisterBeforeUserUpdate(hook BeforeUserUpdateHook) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.beforeUserUpdate = append(h.beforeUserUpdate, hook)
	if h.debug {
		log.Printf("[HookRegistry] Registered BeforeUserUpdate hook (total: %d)", len(h.beforeUserUpdate))
	}
}

func (h *HookRegistry) RegisterAfterUserUpdate(hook AfterUserUpdateHook) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.afterUserUpdate = append(h.afterUserUpdate, hook)
	if h.debug {
		log.Printf("[HookRegistry] Registered AfterUserUpdate hook (total: %d)", len(h.afterUserUpdate))
	}
}

func (h *HookRegistry) RegisterBeforeUserDelete(hook BeforeUserDeleteHook) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.beforeUserDelete = append(h.beforeUserDelete, hook)
	if h.debug {
		log.Printf("[HookRegistry] Registered BeforeUserDelete hook (total: %d)", len(h.beforeUserDelete))
	}
}

func (h *HookRegistry) RegisterAfterUserDelete(hook AfterUserDeleteHook) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.afterUserDelete = append(h.afterUserDelete, hook)
	if h.debug {
		log.Printf("[HookRegistry] Registered AfterUserDelete hook (total: %d)", len(h.afterUserDelete))
	}
}

// Session hook registration methods
func (h *HookRegistry) RegisterBeforeSessionCreate(hook BeforeSessionCreateHook) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.beforeSessionCreate = append(h.beforeSessionCreate, hook)
	if h.debug {
		log.Printf("[HookRegistry] Registered BeforeSessionCreate hook (total: %d)", len(h.beforeSessionCreate))
	}
}

func (h *HookRegistry) RegisterAfterSessionCreate(hook AfterSessionCreateHook) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.afterSessionCreate = append(h.afterSessionCreate, hook)
	if h.debug {
		log.Printf("[HookRegistry] Registered AfterSessionCreate hook (total: %d)", len(h.afterSessionCreate))
	}
}

func (h *HookRegistry) RegisterBeforeSessionRevoke(hook BeforeSessionRevokeHook) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.beforeSessionRevoke = append(h.beforeSessionRevoke, hook)
	if h.debug {
		log.Printf("[HookRegistry] Registered BeforeSessionRevoke hook (total: %d)", len(h.beforeSessionRevoke))
	}
}

func (h *HookRegistry) RegisterAfterSessionRevoke(hook AfterSessionRevokeHook) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.afterSessionRevoke = append(h.afterSessionRevoke, hook)
	if h.debug {
		log.Printf("[HookRegistry] Registered AfterSessionRevoke hook (total: %d)", len(h.afterSessionRevoke))
	}
}

// Auth hook registration methods
func (h *HookRegistry) RegisterBeforeSignUp(hook BeforeSignUpHook) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.beforeSignUp = append(h.beforeSignUp, hook)
	if h.debug {
		log.Printf("[HookRegistry] Registered BeforeSignUp hook (total: %d)", len(h.beforeSignUp))
	}
}

func (h *HookRegistry) RegisterAfterSignUp(hook AfterSignUpHook) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.afterSignUp = append(h.afterSignUp, hook)
	if h.debug {
		log.Printf("[HookRegistry] Registered AfterSignUp hook (total: %d)", len(h.afterSignUp))
	}
}

func (h *HookRegistry) RegisterBeforeSignIn(hook BeforeSignInHook) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.beforeSignIn = append(h.beforeSignIn, hook)
	if h.debug {
		log.Printf("[HookRegistry] Registered BeforeSignIn hook (total: %d)", len(h.beforeSignIn))
	}
}

func (h *HookRegistry) RegisterAfterSignIn(hook AfterSignInHook) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.afterSignIn = append(h.afterSignIn, hook)
	if h.debug {
		log.Printf("[HookRegistry] Registered AfterSignIn hook (total: %d)", len(h.afterSignIn))
	}
}

func (h *HookRegistry) RegisterBeforeSignOut(hook BeforeSignOutHook) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.beforeSignOut = append(h.beforeSignOut, hook)
	if h.debug {
		log.Printf("[HookRegistry] Registered BeforeSignOut hook (total: %d)", len(h.beforeSignOut))
	}
}

func (h *HookRegistry) RegisterAfterSignOut(hook AfterSignOutHook) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.afterSignOut = append(h.afterSignOut, hook)
	if h.debug {
		log.Printf("[HookRegistry] Registered AfterSignOut hook (total: %d)", len(h.afterSignOut))
	}
}

// Organization hook registration methods (for multi-tenancy plugin)
func (h *HookRegistry) RegisterBeforeOrganizationCreate(hook BeforeOrganizationCreateHook) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.beforeOrganizationCreate = append(h.beforeOrganizationCreate, hook)
	if h.debug {
		log.Printf("[HookRegistry] Registered BeforeOrganizationCreate hook (total: %d)", len(h.beforeOrganizationCreate))
	}
}

func (h *HookRegistry) RegisterAfterOrganizationCreate(hook AfterOrganizationCreateHook) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.afterOrganizationCreate = append(h.afterOrganizationCreate, hook)
	if h.debug {
		log.Printf("[HookRegistry] Registered AfterOrganizationCreate hook (total: %d)", len(h.afterOrganizationCreate))
	}
}

func (h *HookRegistry) RegisterBeforeMemberAdd(hook BeforeMemberAddHook) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.beforeMemberAdd = append(h.beforeMemberAdd, hook)
	if h.debug {
		log.Printf("[HookRegistry] Registered BeforeMemberAdd hook (total: %d)", len(h.beforeMemberAdd))
	}
}

func (h *HookRegistry) RegisterAfterMemberAdd(hook AfterMemberAddHook) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.afterMemberAdd = append(h.afterMemberAdd, hook)
	if h.debug {
		log.Printf("[HookRegistry] Registered AfterMemberAdd hook (total: %d)", len(h.afterMemberAdd))
	}
}

func (h *HookRegistry) RegisterAfterOrganizationUpdate(hook AfterOrganizationUpdateHook) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.afterOrganizationUpdate = append(h.afterOrganizationUpdate, hook)
	if h.debug {
		log.Printf("[HookRegistry] Registered AfterOrganizationUpdate hook (total: %d)", len(h.afterOrganizationUpdate))
	}
}

func (h *HookRegistry) RegisterAfterOrganizationDelete(hook AfterOrganizationDeleteHook) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.afterOrganizationDelete = append(h.afterOrganizationDelete, hook)
	if h.debug {
		log.Printf("[HookRegistry] Registered AfterOrganizationDelete hook (total: %d)", len(h.afterOrganizationDelete))
	}
}

func (h *HookRegistry) RegisterAfterMemberRemove(hook AfterMemberRemoveHook) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.afterMemberRemove = append(h.afterMemberRemove, hook)
	if h.debug {
		log.Printf("[HookRegistry] Registered AfterMemberRemove hook (total: %d)", len(h.afterMemberRemove))
	}
}

func (h *HookRegistry) RegisterAfterMemberRoleChange(hook AfterMemberRoleChangeHook) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.afterMemberRoleChange = append(h.afterMemberRoleChange, hook)
	if h.debug {
		log.Printf("[HookRegistry] Registered AfterMemberRoleChange hook (total: %d)", len(h.afterMemberRoleChange))
	}
}

// App hook registration methods (for multi-app support)
func (h *HookRegistry) RegisterBeforeAppCreate(hook BeforeAppCreateHook) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.beforeAppCreate = append(h.beforeAppCreate, hook)
	if h.debug {
		log.Printf("[HookRegistry] Registered BeforeAppCreate hook (total: %d)", len(h.beforeAppCreate))
	}
}

func (h *HookRegistry) RegisterAfterAppCreate(hook AfterAppCreateHook) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.afterAppCreate = append(h.afterAppCreate, hook)
	if h.debug {
		log.Printf("[HookRegistry] Registered AfterAppCreate hook (total: %d)", len(h.afterAppCreate))
	}
}

// Hook execution methods

// ExecuteBeforeUserCreate executes all before user create hooks
func (h *HookRegistry) ExecuteBeforeUserCreate(ctx context.Context, req *user.CreateUserRequest) error {
	h.mu.RLock()
	hooks := make([]BeforeUserCreateHook, len(h.beforeUserCreate))
	copy(hooks, h.beforeUserCreate)
	debug := h.debug
	h.mu.RUnlock()

	if debug {
		log.Printf("[HookRegistry] Executing %d BeforeUserCreate hooks", len(hooks))
	}

	for i, hook := range hooks {
		if err := hook(ctx, req); err != nil {
			if debug {
				log.Printf("[HookRegistry] BeforeUserCreate hook #%d failed: %v", i, err)
			}
			return err
		}
	}
	return nil
}

// ExecuteAfterUserCreate executes all after user create hooks
func (h *HookRegistry) ExecuteAfterUserCreate(ctx context.Context, user *user.User) error {
	h.mu.RLock()
	hooks := make([]AfterUserCreateHook, len(h.afterUserCreate))
	copy(hooks, h.afterUserCreate)
	debug := h.debug
	h.mu.RUnlock()

	if debug {
		log.Printf("[HookRegistry] Executing %d AfterUserCreate hooks", len(hooks))
	}

	for i, hook := range hooks {
		if err := hook(ctx, user); err != nil {
			if debug {
				log.Printf("[HookRegistry] AfterUserCreate hook #%d failed: %v", i, err)
			}
			return err
		}
	}
	return nil
}

// ExecuteBeforeUserUpdate executes all before user update hooks
func (h *HookRegistry) ExecuteBeforeUserUpdate(ctx context.Context, userID xid.ID, req *user.UpdateUserRequest) error {
	h.mu.RLock()
	hooks := make([]BeforeUserUpdateHook, len(h.beforeUserUpdate))
	copy(hooks, h.beforeUserUpdate)
	debug := h.debug
	h.mu.RUnlock()

	if debug {
		log.Printf("[HookRegistry] Executing %d BeforeUserUpdate hooks", len(hooks))
	}

	for i, hook := range hooks {
		if err := hook(ctx, userID, req); err != nil {
			if debug {
				log.Printf("[HookRegistry] BeforeUserUpdate hook #%d failed: %v", i, err)
			}
			return err
		}
	}
	return nil
}

// ExecuteAfterUserUpdate executes all after user update hooks
func (h *HookRegistry) ExecuteAfterUserUpdate(ctx context.Context, user *user.User) error {
	h.mu.RLock()
	hooks := make([]AfterUserUpdateHook, len(h.afterUserUpdate))
	copy(hooks, h.afterUserUpdate)
	debug := h.debug
	h.mu.RUnlock()

	if debug {
		log.Printf("[HookRegistry] Executing %d AfterUserUpdate hooks", len(hooks))
	}

	for i, hook := range hooks {
		if err := hook(ctx, user); err != nil {
			if debug {
				log.Printf("[HookRegistry] AfterUserUpdate hook #%d failed: %v", i, err)
			}
			return err
		}
	}
	return nil
}

// ExecuteBeforeUserDelete executes all before user delete hooks
func (h *HookRegistry) ExecuteBeforeUserDelete(ctx context.Context, userID xid.ID) error {
	h.mu.RLock()
	hooks := make([]BeforeUserDeleteHook, len(h.beforeUserDelete))
	copy(hooks, h.beforeUserDelete)
	debug := h.debug
	h.mu.RUnlock()

	if debug {
		log.Printf("[HookRegistry] Executing %d BeforeUserDelete hooks", len(hooks))
	}

	for i, hook := range hooks {
		if err := hook(ctx, userID); err != nil {
			if debug {
				log.Printf("[HookRegistry] BeforeUserDelete hook #%d failed: %v", i, err)
			}
			return err
		}
	}
	return nil
}

// ExecuteAfterUserDelete executes all after user delete hooks
func (h *HookRegistry) ExecuteAfterUserDelete(ctx context.Context, userID xid.ID) error {
	h.mu.RLock()
	hooks := make([]AfterUserDeleteHook, len(h.afterUserDelete))
	copy(hooks, h.afterUserDelete)
	debug := h.debug
	h.mu.RUnlock()

	if debug {
		log.Printf("[HookRegistry] Executing %d AfterUserDelete hooks", len(hooks))
	}

	for i, hook := range hooks {
		if err := hook(ctx, userID); err != nil {
			if debug {
				log.Printf("[HookRegistry] AfterUserDelete hook #%d failed: %v", i, err)
			}
			return err
		}
	}
	return nil
}

// ExecuteBeforeSessionCreate executes all before session create hooks
func (h *HookRegistry) ExecuteBeforeSessionCreate(ctx context.Context, req *session.CreateSessionRequest) error {
	h.mu.RLock()
	hooks := make([]BeforeSessionCreateHook, len(h.beforeSessionCreate))
	copy(hooks, h.beforeSessionCreate)
	debug := h.debug
	h.mu.RUnlock()

	if debug {
		log.Printf("[HookRegistry] Executing %d BeforeSessionCreate hooks", len(hooks))
	}

	for i, hook := range hooks {
		if err := hook(ctx, req); err != nil {
			if debug {
				log.Printf("[HookRegistry] BeforeSessionCreate hook #%d failed: %v", i, err)
			}
			return err
		}
	}
	return nil
}

// ExecuteAfterSessionCreate executes all after session create hooks
func (h *HookRegistry) ExecuteAfterSessionCreate(ctx context.Context, session *session.Session) error {
	h.mu.RLock()
	hooks := make([]AfterSessionCreateHook, len(h.afterSessionCreate))
	copy(hooks, h.afterSessionCreate)
	debug := h.debug
	h.mu.RUnlock()

	if debug {
		log.Printf("[HookRegistry] Executing %d AfterSessionCreate hooks", len(hooks))
	}

	for i, hook := range hooks {
		if err := hook(ctx, session); err != nil {
			if debug {
				log.Printf("[HookRegistry] AfterSessionCreate hook #%d failed: %v", i, err)
			}
			return err
		}
	}
	return nil
}

// ExecuteBeforeSessionRevoke executes all before session revoke hooks
func (h *HookRegistry) ExecuteBeforeSessionRevoke(ctx context.Context, token string) error {
	h.mu.RLock()
	hooks := make([]BeforeSessionRevokeHook, len(h.beforeSessionRevoke))
	copy(hooks, h.beforeSessionRevoke)
	debug := h.debug
	h.mu.RUnlock()

	if debug {
		log.Printf("[HookRegistry] Executing %d BeforeSessionRevoke hooks", len(hooks))
	}

	for i, hook := range hooks {
		if err := hook(ctx, token); err != nil {
			if debug {
				log.Printf("[HookRegistry] BeforeSessionRevoke hook #%d failed: %v", i, err)
			}
			return err
		}
	}
	return nil
}

// ExecuteAfterSessionRevoke executes all after session revoke hooks
func (h *HookRegistry) ExecuteAfterSessionRevoke(ctx context.Context, sessionID xid.ID) error {
	h.mu.RLock()
	hooks := make([]AfterSessionRevokeHook, len(h.afterSessionRevoke))
	copy(hooks, h.afterSessionRevoke)
	debug := h.debug
	h.mu.RUnlock()

	if debug {
		log.Printf("[HookRegistry] Executing %d AfterSessionRevoke hooks", len(hooks))
	}

	for i, hook := range hooks {
		if err := hook(ctx, sessionID); err != nil {
			if debug {
				log.Printf("[HookRegistry] AfterSessionRevoke hook #%d failed: %v", i, err)
			}
			return err
		}
	}
	return nil
}

// ExecuteBeforeSignUp executes all before sign up hooks
func (h *HookRegistry) ExecuteBeforeSignUp(ctx context.Context, req *auth.SignUpRequest) error {
	h.mu.RLock()
	hooks := make([]BeforeSignUpHook, len(h.beforeSignUp))
	copy(hooks, h.beforeSignUp)
	debug := h.debug
	h.mu.RUnlock()

	if debug {
		log.Printf("[HookRegistry] Executing %d BeforeSignUp hooks", len(hooks))
	}

	for i, hook := range hooks {
		if err := hook(ctx, req); err != nil {
			if debug {
				log.Printf("[HookRegistry] BeforeSignUp hook #%d failed: %v", i, err)
			}
			return err
		}
	}
	return nil
}

// ExecuteAfterSignUp executes all after sign up hooks
func (h *HookRegistry) ExecuteAfterSignUp(ctx context.Context, response *responses.AuthResponse) error {
	h.mu.RLock()
	hooks := make([]AfterSignUpHook, len(h.afterSignUp))
	copy(hooks, h.afterSignUp)
	debug := h.debug
	h.mu.RUnlock()

	if debug {
		log.Printf("[HookRegistry] Executing %d AfterSignUp hooks", len(hooks))
	}

	for i, hook := range hooks {
		if err := hook(ctx, response); err != nil {
			if debug {
				log.Printf("[HookRegistry] AfterSignUp hook #%d failed: %v", i, err)
			}
			return err
		}
	}
	return nil
}

// ExecuteBeforeSignIn executes all before sign in hooks
func (h *HookRegistry) ExecuteBeforeSignIn(ctx context.Context, req *auth.SignInRequest) error {
	h.mu.RLock()
	hooks := make([]BeforeSignInHook, len(h.beforeSignIn))
	copy(hooks, h.beforeSignIn)
	debug := h.debug
	h.mu.RUnlock()

	if debug {
		log.Printf("[HookRegistry] Executing %d BeforeSignIn hooks", len(hooks))
	}

	for i, hook := range hooks {
		if err := hook(ctx, req); err != nil {
			if debug {
				log.Printf("[HookRegistry] BeforeSignIn hook #%d failed: %v", i, err)
			}
			return err
		}
	}
	return nil
}

// ExecuteAfterSignIn executes all after sign in hooks
func (h *HookRegistry) ExecuteAfterSignIn(ctx context.Context, response *responses.AuthResponse) error {
	h.mu.RLock()
	hooks := make([]AfterSignInHook, len(h.afterSignIn))
	copy(hooks, h.afterSignIn)
	debug := h.debug
	h.mu.RUnlock()

	if debug {
		log.Printf("[HookRegistry] Executing %d AfterSignIn hooks", len(hooks))
	}

	for i, hook := range hooks {
		if err := hook(ctx, response); err != nil {
			if debug {
				log.Printf("[HookRegistry] AfterSignIn hook #%d failed: %v", i, err)
			}
			return err
		}
	}

	if debug {
		log.Printf("[HookRegistry] All %d AfterSignIn hooks executed successfully", len(hooks))
	}
	return nil
}

// ExecuteBeforeSignOut executes all before sign out hooks
func (h *HookRegistry) ExecuteBeforeSignOut(ctx context.Context, token string) error {
	h.mu.RLock()
	hooks := make([]BeforeSignOutHook, len(h.beforeSignOut))
	copy(hooks, h.beforeSignOut)
	debug := h.debug
	h.mu.RUnlock()

	if debug {
		log.Printf("[HookRegistry] Executing %d BeforeSignOut hooks", len(hooks))
	}

	for i, hook := range hooks {
		if err := hook(ctx, token); err != nil {
			if debug {
				log.Printf("[HookRegistry] BeforeSignOut hook #%d failed: %v", i, err)
			}
			return err
		}
	}
	return nil
}

// ExecuteAfterSignOut executes all after sign out hooks
func (h *HookRegistry) ExecuteAfterSignOut(ctx context.Context, token string) error {
	h.mu.RLock()
	hooks := make([]AfterSignOutHook, len(h.afterSignOut))
	copy(hooks, h.afterSignOut)
	debug := h.debug
	h.mu.RUnlock()

	if debug {
		log.Printf("[HookRegistry] Executing %d AfterSignOut hooks", len(hooks))
	}

	for i, hook := range hooks {
		if err := hook(ctx, token); err != nil {
			if debug {
				log.Printf("[HookRegistry] AfterSignOut hook #%d failed: %v", i, err)
			}
			return err
		}
	}
	return nil
}

// ExecuteBeforeOrganizationCreate executes all before organization create hooks
func (h *HookRegistry) ExecuteBeforeOrganizationCreate(ctx context.Context, req interface{}) error {
	h.mu.RLock()
	hooks := make([]BeforeOrganizationCreateHook, len(h.beforeOrganizationCreate))
	copy(hooks, h.beforeOrganizationCreate)
	debug := h.debug
	h.mu.RUnlock()

	if debug {
		log.Printf("[HookRegistry] Executing %d BeforeOrganizationCreate hooks", len(hooks))
	}

	for i, hook := range hooks {
		if err := hook(ctx, req); err != nil {
			if debug {
				log.Printf("[HookRegistry] BeforeOrganizationCreate hook #%d failed: %v", i, err)
			}
			return err
		}
	}
	return nil
}

// ExecuteAfterOrganizationCreate executes all after organization create hooks
func (h *HookRegistry) ExecuteAfterOrganizationCreate(ctx context.Context, org interface{}) error {
	h.mu.RLock()
	hooks := make([]AfterOrganizationCreateHook, len(h.afterOrganizationCreate))
	copy(hooks, h.afterOrganizationCreate)
	debug := h.debug
	h.mu.RUnlock()

	if debug {
		log.Printf("[HookRegistry] Executing %d AfterOrganizationCreate hooks", len(hooks))
	}

	for i, hook := range hooks {
		if err := hook(ctx, org); err != nil {
			if debug {
				log.Printf("[HookRegistry] AfterOrganizationCreate hook #%d failed: %v", i, err)
			}
			return err
		}
	}
	return nil
}

// ExecuteBeforeMemberAdd executes all before member add hooks
func (h *HookRegistry) ExecuteBeforeMemberAdd(ctx context.Context, orgID string, userID xid.ID) error {
	h.mu.RLock()
	hooks := make([]BeforeMemberAddHook, len(h.beforeMemberAdd))
	copy(hooks, h.beforeMemberAdd)
	debug := h.debug
	h.mu.RUnlock()

	if debug {
		log.Printf("[HookRegistry] Executing %d BeforeMemberAdd hooks", len(hooks))
	}

	for i, hook := range hooks {
		if err := hook(ctx, orgID, userID); err != nil {
			if debug {
				log.Printf("[HookRegistry] BeforeMemberAdd hook #%d failed: %v", i, err)
			}
			return err
		}
	}
	return nil
}

// ExecuteAfterMemberAdd executes all after member add hooks
func (h *HookRegistry) ExecuteAfterMemberAdd(ctx context.Context, member interface{}) error {
	h.mu.RLock()
	hooks := make([]AfterMemberAddHook, len(h.afterMemberAdd))
	copy(hooks, h.afterMemberAdd)
	debug := h.debug
	h.mu.RUnlock()

	if debug {
		log.Printf("[HookRegistry] Executing %d AfterMemberAdd hooks", len(hooks))
	}

	for i, hook := range hooks {
		if err := hook(ctx, member); err != nil {
			if debug {
				log.Printf("[HookRegistry] AfterMemberAdd hook #%d failed: %v", i, err)
			}
			return err
		}
	}
	return nil
}

// ExecuteAfterOrganizationUpdate executes all after organization update hooks
func (h *HookRegistry) ExecuteAfterOrganizationUpdate(ctx context.Context, org interface{}) error {
	h.mu.RLock()
	hooks := make([]AfterOrganizationUpdateHook, len(h.afterOrganizationUpdate))
	copy(hooks, h.afterOrganizationUpdate)
	debug := h.debug
	h.mu.RUnlock()

	if debug {
		log.Printf("[HookRegistry] Executing %d AfterOrganizationUpdate hooks", len(hooks))
	}

	for i, hook := range hooks {
		if err := hook(ctx, org); err != nil {
			if debug {
				log.Printf("[HookRegistry] AfterOrganizationUpdate hook #%d failed: %v", i, err)
			}
			return err
		}
	}
	return nil
}

// ExecuteAfterOrganizationDelete executes all after organization delete hooks
func (h *HookRegistry) ExecuteAfterOrganizationDelete(ctx context.Context, orgID xid.ID, orgName string) error {
	h.mu.RLock()
	hooks := make([]AfterOrganizationDeleteHook, len(h.afterOrganizationDelete))
	copy(hooks, h.afterOrganizationDelete)
	debug := h.debug
	h.mu.RUnlock()

	if debug {
		log.Printf("[HookRegistry] Executing %d AfterOrganizationDelete hooks", len(hooks))
	}

	for i, hook := range hooks {
		if err := hook(ctx, orgID, orgName); err != nil {
			if debug {
				log.Printf("[HookRegistry] AfterOrganizationDelete hook #%d failed: %v", i, err)
			}
			return err
		}
	}
	return nil
}

// ExecuteAfterMemberRemove executes all after member remove hooks
func (h *HookRegistry) ExecuteAfterMemberRemove(ctx context.Context, orgID xid.ID, userID xid.ID, memberName string) error {
	h.mu.RLock()
	hooks := make([]AfterMemberRemoveHook, len(h.afterMemberRemove))
	copy(hooks, h.afterMemberRemove)
	debug := h.debug
	h.mu.RUnlock()

	if debug {
		log.Printf("[HookRegistry] Executing %d AfterMemberRemove hooks", len(hooks))
	}

	for i, hook := range hooks {
		if err := hook(ctx, orgID, userID, memberName); err != nil {
			if debug {
				log.Printf("[HookRegistry] AfterMemberRemove hook #%d failed: %v", i, err)
			}
			return err
		}
	}
	return nil
}

// ExecuteAfterMemberRoleChange executes all after member role change hooks
func (h *HookRegistry) ExecuteAfterMemberRoleChange(ctx context.Context, orgID xid.ID, userID xid.ID, oldRole string, newRole string) error {
	h.mu.RLock()
	hooks := make([]AfterMemberRoleChangeHook, len(h.afterMemberRoleChange))
	copy(hooks, h.afterMemberRoleChange)
	debug := h.debug
	h.mu.RUnlock()

	if debug {
		log.Printf("[HookRegistry] Executing %d AfterMemberRoleChange hooks", len(hooks))
	}

	for i, hook := range hooks {
		if err := hook(ctx, orgID, userID, oldRole, newRole); err != nil {
			if debug {
				log.Printf("[HookRegistry] AfterMemberRoleChange hook #%d failed: %v", i, err)
			}
			return err
		}
	}
	return nil
}

// ExecuteBeforeAppCreate executes all before app create hooks
func (h *HookRegistry) ExecuteBeforeAppCreate(ctx context.Context, req interface{}) error {
	h.mu.RLock()
	hooks := make([]BeforeAppCreateHook, len(h.beforeAppCreate))
	copy(hooks, h.beforeAppCreate)
	debug := h.debug
	h.mu.RUnlock()

	if debug {
		log.Printf("[HookRegistry] Executing %d BeforeAppCreate hooks", len(hooks))
	}

	for i, hook := range hooks {
		if err := hook(ctx, req); err != nil {
			if debug {
				log.Printf("[HookRegistry] BeforeAppCreate hook #%d failed: %v", i, err)
			}
			return err
		}
	}
	return nil
}

// ExecuteAfterAppCreate executes all after app create hooks
func (h *HookRegistry) ExecuteAfterAppCreate(ctx context.Context, app interface{}) error {
	h.mu.RLock()
	hooks := make([]AfterAppCreateHook, len(h.afterAppCreate))
	copy(hooks, h.afterAppCreate)
	debug := h.debug
	h.mu.RUnlock()

	if debug {
		log.Printf("[HookRegistry] Executing %d AfterAppCreate hooks", len(hooks))
	}

	for i, hook := range hooks {
		if err := hook(ctx, app); err != nil {
			if debug {
				log.Printf("[HookRegistry] AfterAppCreate hook #%d failed: %v", i, err)
			}
			return err
		}
	}
	return nil
}

// =============================================================================
// PERMISSION HOOKS (for permissions plugin)
// =============================================================================

// RegisterBeforePermissionEvaluate registers a before permission evaluate hook
func (h *HookRegistry) RegisterBeforePermissionEvaluate(hook BeforePermissionEvaluateHook) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.beforePermissionEvaluate = append(h.beforePermissionEvaluate, hook)
	if h.debug {
		log.Printf("[HookRegistry] Registered BeforePermissionEvaluate hook (total: %d)", len(h.beforePermissionEvaluate))
	}
}

// RegisterAfterPermissionEvaluate registers an after permission evaluate hook
func (h *HookRegistry) RegisterAfterPermissionEvaluate(hook AfterPermissionEvaluateHook) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.afterPermissionEvaluate = append(h.afterPermissionEvaluate, hook)
	if h.debug {
		log.Printf("[HookRegistry] Registered AfterPermissionEvaluate hook (total: %d)", len(h.afterPermissionEvaluate))
	}
}

// RegisterOnPolicyChange registers an on policy change hook
func (h *HookRegistry) RegisterOnPolicyChange(hook OnPolicyChangeHook) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onPolicyChange = append(h.onPolicyChange, hook)
	if h.debug {
		log.Printf("[HookRegistry] Registered OnPolicyChange hook (total: %d)", len(h.onPolicyChange))
	}
}

// RegisterOnCacheInvalidate registers an on cache invalidate hook
func (h *HookRegistry) RegisterOnCacheInvalidate(hook OnCacheInvalidateHook) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onCacheInvalidate = append(h.onCacheInvalidate, hook)
	if h.debug {
		log.Printf("[HookRegistry] Registered OnCacheInvalidate hook (total: %d)", len(h.onCacheInvalidate))
	}
}

// Device/Session security hook registration methods
func (h *HookRegistry) RegisterOnNewDeviceDetected(hook OnNewDeviceDetectedHook) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onNewDeviceDetected = append(h.onNewDeviceDetected, hook)
	if h.debug {
		log.Printf("[HookRegistry] Registered OnNewDeviceDetected hook (total: %d)", len(h.onNewDeviceDetected))
	}
}

func (h *HookRegistry) RegisterOnNewLocationDetected(hook OnNewLocationDetectedHook) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onNewLocationDetected = append(h.onNewLocationDetected, hook)
	if h.debug {
		log.Printf("[HookRegistry] Registered OnNewLocationDetected hook (total: %d)", len(h.onNewLocationDetected))
	}
}

func (h *HookRegistry) RegisterOnSuspiciousLoginDetected(hook OnSuspiciousLoginDetectedHook) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onSuspiciousLoginDetected = append(h.onSuspiciousLoginDetected, hook)
	if h.debug {
		log.Printf("[HookRegistry] Registered OnSuspiciousLoginDetected hook (total: %d)", len(h.onSuspiciousLoginDetected))
	}
}

func (h *HookRegistry) RegisterOnDeviceRemoved(hook OnDeviceRemovedHook) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onDeviceRemoved = append(h.onDeviceRemoved, hook)
	if h.debug {
		log.Printf("[HookRegistry] Registered OnDeviceRemoved hook (total: %d)", len(h.onDeviceRemoved))
	}
}

func (h *HookRegistry) RegisterOnAllSessionsRevoked(hook OnAllSessionsRevokedHook) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onAllSessionsRevoked = append(h.onAllSessionsRevoked, hook)
	if h.debug {
		log.Printf("[HookRegistry] Registered OnAllSessionsRevoked hook (total: %d)", len(h.onAllSessionsRevoked))
	}
}

// Account lifecycle hook registration methods
func (h *HookRegistry) RegisterOnEmailChangeRequest(hook OnEmailChangeRequestHook) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onEmailChangeRequest = append(h.onEmailChangeRequest, hook)
	if h.debug {
		log.Printf("[HookRegistry] Registered OnEmailChangeRequest hook (total: %d)", len(h.onEmailChangeRequest))
	}
}

func (h *HookRegistry) RegisterOnEmailChanged(hook OnEmailChangedHook) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onEmailChanged = append(h.onEmailChanged, hook)
	if h.debug {
		log.Printf("[HookRegistry] Registered OnEmailChanged hook (total: %d)", len(h.onEmailChanged))
	}
}

func (h *HookRegistry) RegisterOnPasswordChanged(hook OnPasswordChangedHook) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onPasswordChanged = append(h.onPasswordChanged, hook)
	if h.debug {
		log.Printf("[HookRegistry] Registered OnPasswordChanged hook (total: %d)", len(h.onPasswordChanged))
	}
}

func (h *HookRegistry) RegisterOnUsernameChanged(hook OnUsernameChangedHook) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onUsernameChanged = append(h.onUsernameChanged, hook)
	if h.debug {
		log.Printf("[HookRegistry] Registered OnUsernameChanged hook (total: %d)", len(h.onUsernameChanged))
	}
}

func (h *HookRegistry) RegisterOnAccountDeleted(hook OnAccountDeletedHook) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onAccountDeleted = append(h.onAccountDeleted, hook)
	if h.debug {
		log.Printf("[HookRegistry] Registered OnAccountDeleted hook (total: %d)", len(h.onAccountDeleted))
	}
}

func (h *HookRegistry) RegisterOnAccountSuspended(hook OnAccountSuspendedHook) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onAccountSuspended = append(h.onAccountSuspended, hook)
	if h.debug {
		log.Printf("[HookRegistry] Registered OnAccountSuspended hook (total: %d)", len(h.onAccountSuspended))
	}
}

func (h *HookRegistry) RegisterOnAccountReactivated(hook OnAccountReactivatedHook) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onAccountReactivated = append(h.onAccountReactivated, hook)
	if h.debug {
		log.Printf("[HookRegistry] Registered OnAccountReactivated hook (total: %d)", len(h.onAccountReactivated))
	}
}

// ExecuteBeforePermissionEvaluate executes all before permission evaluate hooks
func (h *HookRegistry) ExecuteBeforePermissionEvaluate(ctx context.Context, req *PermissionEvaluateRequest) error {
	h.mu.RLock()
	hooks := make([]BeforePermissionEvaluateHook, len(h.beforePermissionEvaluate))
	copy(hooks, h.beforePermissionEvaluate)
	debug := h.debug
	h.mu.RUnlock()

	if debug {
		log.Printf("[HookRegistry] Executing %d BeforePermissionEvaluate hooks", len(hooks))
	}

	for i, hook := range hooks {
		if err := hook(ctx, req); err != nil {
			if debug {
				log.Printf("[HookRegistry] BeforePermissionEvaluate hook #%d failed: %v", i, err)
			}
			return err
		}
	}
	return nil
}

// ExecuteAfterPermissionEvaluate executes all after permission evaluate hooks
func (h *HookRegistry) ExecuteAfterPermissionEvaluate(ctx context.Context, req *PermissionEvaluateRequest, decision *PermissionDecision) error {
	h.mu.RLock()
	hooks := make([]AfterPermissionEvaluateHook, len(h.afterPermissionEvaluate))
	copy(hooks, h.afterPermissionEvaluate)
	debug := h.debug
	h.mu.RUnlock()

	if debug {
		log.Printf("[HookRegistry] Executing %d AfterPermissionEvaluate hooks", len(hooks))
	}

	for i, hook := range hooks {
		if err := hook(ctx, req, decision); err != nil {
			if debug {
				log.Printf("[HookRegistry] AfterPermissionEvaluate hook #%d failed: %v", i, err)
			}
			return err
		}
	}
	return nil
}

// ExecuteOnPolicyChange executes all on policy change hooks
func (h *HookRegistry) ExecuteOnPolicyChange(ctx context.Context, policyID xid.ID, action string) error {
	h.mu.RLock()
	hooks := make([]OnPolicyChangeHook, len(h.onPolicyChange))
	copy(hooks, h.onPolicyChange)
	debug := h.debug
	h.mu.RUnlock()

	if debug {
		log.Printf("[HookRegistry] Executing %d OnPolicyChange hooks for policy %s action %s", len(hooks), policyID, action)
	}

	for i, hook := range hooks {
		if err := hook(ctx, policyID, action); err != nil {
			if debug {
				log.Printf("[HookRegistry] OnPolicyChange hook #%d failed: %v", i, err)
			}
			return err
		}
	}
	return nil
}

// ExecuteOnCacheInvalidate executes all on cache invalidate hooks
func (h *HookRegistry) ExecuteOnCacheInvalidate(ctx context.Context, scope string, id xid.ID) error {
	h.mu.RLock()
	hooks := make([]OnCacheInvalidateHook, len(h.onCacheInvalidate))
	copy(hooks, h.onCacheInvalidate)
	debug := h.debug
	h.mu.RUnlock()

	if debug {
		log.Printf("[HookRegistry] Executing %d OnCacheInvalidate hooks for scope %s id %s", len(hooks), scope, id)
	}

	for i, hook := range hooks {
		if err := hook(ctx, scope, id); err != nil {
			if debug {
				log.Printf("[HookRegistry] OnCacheInvalidate hook #%d failed: %v", i, err)
			}
			return err
		}
	}
	return nil
}

// Device/Session security hook execution methods
func (h *HookRegistry) ExecuteOnNewDeviceDetected(ctx context.Context, userID xid.ID, deviceName, location, ipAddress string) error {
	h.mu.RLock()
	hooks := make([]OnNewDeviceDetectedHook, len(h.onNewDeviceDetected))
	copy(hooks, h.onNewDeviceDetected)
	debug := h.debug
	h.mu.RUnlock()

	if debug {
		log.Printf("[HookRegistry] Executing %d OnNewDeviceDetected hooks", len(hooks))
	}

	for i, hook := range hooks {
		if err := hook(ctx, userID, deviceName, location, ipAddress); err != nil {
			if debug {
				log.Printf("[HookRegistry] OnNewDeviceDetected hook #%d failed: %v", i, err)
			}
			return err
		}
	}
	return nil
}

func (h *HookRegistry) ExecuteOnNewLocationDetected(ctx context.Context, userID xid.ID, location, ipAddress string) error {
	h.mu.RLock()
	hooks := make([]OnNewLocationDetectedHook, len(h.onNewLocationDetected))
	copy(hooks, h.onNewLocationDetected)
	debug := h.debug
	h.mu.RUnlock()

	if debug {
		log.Printf("[HookRegistry] Executing %d OnNewLocationDetected hooks", len(hooks))
	}

	for i, hook := range hooks {
		if err := hook(ctx, userID, location, ipAddress); err != nil {
			if debug {
				log.Printf("[HookRegistry] OnNewLocationDetected hook #%d failed: %v", i, err)
			}
			return err
		}
	}
	return nil
}

func (h *HookRegistry) ExecuteOnSuspiciousLoginDetected(ctx context.Context, userID xid.ID, reason, location, ipAddress string) error {
	h.mu.RLock()
	hooks := make([]OnSuspiciousLoginDetectedHook, len(h.onSuspiciousLoginDetected))
	copy(hooks, h.onSuspiciousLoginDetected)
	debug := h.debug
	h.mu.RUnlock()

	if debug {
		log.Printf("[HookRegistry] Executing %d OnSuspiciousLoginDetected hooks", len(hooks))
	}

	for i, hook := range hooks {
		if err := hook(ctx, userID, reason, location, ipAddress); err != nil {
			if debug {
				log.Printf("[HookRegistry] OnSuspiciousLoginDetected hook #%d failed: %v", i, err)
			}
			return err
		}
	}
	return nil
}

func (h *HookRegistry) ExecuteOnDeviceRemoved(ctx context.Context, userID xid.ID, deviceName string) error {
	h.mu.RLock()
	hooks := make([]OnDeviceRemovedHook, len(h.onDeviceRemoved))
	copy(hooks, h.onDeviceRemoved)
	debug := h.debug
	h.mu.RUnlock()

	if debug {
		log.Printf("[HookRegistry] Executing %d OnDeviceRemoved hooks", len(hooks))
	}

	for i, hook := range hooks {
		if err := hook(ctx, userID, deviceName); err != nil {
			if debug {
				log.Printf("[HookRegistry] OnDeviceRemoved hook #%d failed: %v", i, err)
			}
			return err
		}
	}
	return nil
}

func (h *HookRegistry) ExecuteOnAllSessionsRevoked(ctx context.Context, userID xid.ID) error {
	h.mu.RLock()
	hooks := make([]OnAllSessionsRevokedHook, len(h.onAllSessionsRevoked))
	copy(hooks, h.onAllSessionsRevoked)
	debug := h.debug
	h.mu.RUnlock()

	if debug {
		log.Printf("[HookRegistry] Executing %d OnAllSessionsRevoked hooks", len(hooks))
	}

	for i, hook := range hooks {
		if err := hook(ctx, userID); err != nil {
			if debug {
				log.Printf("[HookRegistry] OnAllSessionsRevoked hook #%d failed: %v", i, err)
			}
			return err
		}
	}
	return nil
}

// Account lifecycle hook execution methods
func (h *HookRegistry) ExecuteOnEmailChangeRequest(ctx context.Context, userID xid.ID, oldEmail, newEmail, confirmationUrl string) error {
	h.mu.RLock()
	hooks := make([]OnEmailChangeRequestHook, len(h.onEmailChangeRequest))
	copy(hooks, h.onEmailChangeRequest)
	debug := h.debug
	h.mu.RUnlock()

	if debug {
		log.Printf("[HookRegistry] Executing %d OnEmailChangeRequest hooks", len(hooks))
	}

	for i, hook := range hooks {
		if err := hook(ctx, userID, oldEmail, newEmail, confirmationUrl); err != nil {
			if debug {
				log.Printf("[HookRegistry] OnEmailChangeRequest hook #%d failed: %v", i, err)
			}
			return err
		}
	}
	return nil
}

func (h *HookRegistry) ExecuteOnEmailChanged(ctx context.Context, userID xid.ID, oldEmail, newEmail string) error {
	h.mu.RLock()
	hooks := make([]OnEmailChangedHook, len(h.onEmailChanged))
	copy(hooks, h.onEmailChanged)
	debug := h.debug
	h.mu.RUnlock()

	if debug {
		log.Printf("[HookRegistry] Executing %d OnEmailChanged hooks", len(hooks))
	}

	for i, hook := range hooks {
		if err := hook(ctx, userID, oldEmail, newEmail); err != nil {
			if debug {
				log.Printf("[HookRegistry] OnEmailChanged hook #%d failed: %v", i, err)
			}
			return err
		}
	}
	return nil
}

func (h *HookRegistry) ExecuteOnPasswordChanged(ctx context.Context, userID xid.ID) error {
	h.mu.RLock()
	hooks := make([]OnPasswordChangedHook, len(h.onPasswordChanged))
	copy(hooks, h.onPasswordChanged)
	debug := h.debug
	h.mu.RUnlock()

	if debug {
		log.Printf("[HookRegistry] Executing %d OnPasswordChanged hooks", len(hooks))
	}

	for i, hook := range hooks {
		if err := hook(ctx, userID); err != nil {
			if debug {
				log.Printf("[HookRegistry] OnPasswordChanged hook #%d failed: %v", i, err)
			}
			return err
		}
	}
	return nil
}

func (h *HookRegistry) ExecuteOnUsernameChanged(ctx context.Context, userID xid.ID, oldUsername, newUsername string) error {
	h.mu.RLock()
	hooks := make([]OnUsernameChangedHook, len(h.onUsernameChanged))
	copy(hooks, h.onUsernameChanged)
	debug := h.debug
	h.mu.RUnlock()

	if debug {
		log.Printf("[HookRegistry] Executing %d OnUsernameChanged hooks", len(hooks))
	}

	for i, hook := range hooks {
		if err := hook(ctx, userID, oldUsername, newUsername); err != nil {
			if debug {
				log.Printf("[HookRegistry] OnUsernameChanged hook #%d failed: %v", i, err)
			}
			return err
		}
	}
	return nil
}

func (h *HookRegistry) ExecuteOnAccountDeleted(ctx context.Context, userID xid.ID) error {
	h.mu.RLock()
	hooks := make([]OnAccountDeletedHook, len(h.onAccountDeleted))
	copy(hooks, h.onAccountDeleted)
	debug := h.debug
	h.mu.RUnlock()

	if debug {
		log.Printf("[HookRegistry] Executing %d OnAccountDeleted hooks", len(hooks))
	}

	for i, hook := range hooks {
		if err := hook(ctx, userID); err != nil {
			if debug {
				log.Printf("[HookRegistry] OnAccountDeleted hook #%d failed: %v", i, err)
			}
			return err
		}
	}
	return nil
}

func (h *HookRegistry) ExecuteOnAccountSuspended(ctx context.Context, userID xid.ID, reason string) error {
	h.mu.RLock()
	hooks := make([]OnAccountSuspendedHook, len(h.onAccountSuspended))
	copy(hooks, h.onAccountSuspended)
	debug := h.debug
	h.mu.RUnlock()

	if debug {
		log.Printf("[HookRegistry] Executing %d OnAccountSuspended hooks", len(hooks))
	}

	for i, hook := range hooks {
		if err := hook(ctx, userID, reason); err != nil {
			if debug {
				log.Printf("[HookRegistry] OnAccountSuspended hook #%d failed: %v", i, err)
			}
			return err
		}
	}
	return nil
}

func (h *HookRegistry) ExecuteOnAccountReactivated(ctx context.Context, userID xid.ID) error {
	h.mu.RLock()
	hooks := make([]OnAccountReactivatedHook, len(h.onAccountReactivated))
	copy(hooks, h.onAccountReactivated)
	debug := h.debug
	h.mu.RUnlock()

	if debug {
		log.Printf("[HookRegistry] Executing %d OnAccountReactivated hooks", len(hooks))
	}

	for i, hook := range hooks {
		if err := hook(ctx, userID); err != nil {
			if debug {
				log.Printf("[HookRegistry] OnAccountReactivated hook #%d failed: %v", i, err)
			}
			return err
		}
	}
	return nil
}
