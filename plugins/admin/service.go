package admin

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/core/rbac"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/types"
)

// Config holds the admin plugin configuration
type Config struct {
	// RequiredRole is the role required to access admin endpoints
	RequiredRole string `json:"required_role"`
	// AllowUserCreation allows admins to create users
	AllowUserCreation bool `json:"allow_user_creation"`
	// AllowUserDeletion allows admins to delete users
	AllowUserDeletion bool `json:"allow_user_deletion"`
	// AllowImpersonation allows admins to impersonate users
	AllowImpersonation bool `json:"allow_impersonation"`
	// MaxImpersonationDuration is the maximum duration for impersonation sessions
	MaxImpersonationDuration time.Duration `json:"max_impersonation_duration"`
}

// DefaultConfig returns the default admin plugin configuration
func DefaultConfig() Config {
	return Config{
		RequiredRole:             "admin",
		AllowUserCreation:        true,
		AllowUserDeletion:        true,
		AllowImpersonation:       true,
		MaxImpersonationDuration: 24 * time.Hour,
	}
}

// Service provides admin functionality for user management
type Service struct {
	config         Config
	userService    *user.Service
	sessionService *session.Service
	rbacService    *rbac.Service
	auditService   *audit.Service
	banService     *user.BanService
}

// NewService creates a new admin service
func NewService(
	config Config,
	userService *user.Service,
	sessionService *session.Service,
	rbacService *rbac.Service,
	auditService *audit.Service,
	banService *user.BanService,
) *Service {
	return &Service{
		config:         config,
		userService:    userService,
		sessionService: sessionService,
		rbacService:    rbacService,
		auditService:   auditService,
		banService:     banService,
	}
}

// CreateUserRequest represents a request to create a user
// Updated for V2 architecture: App → Environment → Organization
type CreateUserRequest struct {
	AppID              xid.ID            `json:"app_id"`                         // Platform app (required)
	UserOrganizationID *xid.ID           `json:"user_organization_id,omitempty"` // User-created org (optional)
	Email              string            `json:"email"`
	Password           string            `json:"password,omitempty"`
	Name               string            `json:"name,omitempty"`
	Username           string            `json:"username,omitempty"`
	Role               string            `json:"role,omitempty"`
	EmailVerified      bool              `json:"email_verified"`
	Metadata           map[string]string `json:"metadata,omitempty"`
	AdminID            xid.ID            `json:"-"` // Set by handler
}

// ListUsersRequest represents a request to list users
// Updated for V2 architecture
type ListUsersRequest struct {
	AppID              xid.ID  `json:"app_id"`                         // Platform app (required)
	UserOrganizationID *xid.ID `json:"user_organization_id,omitempty"` // User-created org (optional)
	Page               int     `json:"page"`
	Limit              int     `json:"limit"`
	Search             string  `json:"search,omitempty"`
	Role               string  `json:"role,omitempty"`
	Status             string  `json:"status,omitempty"` // active, banned, inactive
	AdminID            xid.ID  `json:"-"`                // Set by handler
}

// ListUsersResponse represents the response for listing users
type ListUsersResponse struct {
	Users      []*user.User `json:"users"`
	Total      int          `json:"total"`
	Page       int          `json:"page"`
	Limit      int          `json:"limit"`
	TotalPages int          `json:"total_pages"`
}

// BanUserRequest represents a request to ban a user
// Updated for V2 architecture
type BanUserRequest struct {
	AppID              xid.ID     `json:"app_id"`                         // Platform app (required)
	UserOrganizationID *xid.ID    `json:"user_organization_id,omitempty"` // User-created org (optional)
	UserID             xid.ID     `json:"user_id"`
	Reason             string     `json:"reason"`
	ExpiresAt          *time.Time `json:"expires_at,omitempty"`
	AdminID            xid.ID     `json:"-"` // Set by handler
}

// UnbanUserRequest represents a request to unban a user
// Updated for V2 architecture
type UnbanUserRequest struct {
	AppID              xid.ID  `json:"app_id"`                         // Platform app (required)
	UserOrganizationID *xid.ID `json:"user_organization_id,omitempty"` // User-created org (optional)
	UserID             xid.ID  `json:"user_id"`
	Reason             string  `json:"reason,omitempty"`
	AdminID            xid.ID  `json:"-"` // Set by handler
}

// ImpersonateUserRequest represents a request to impersonate a user
// Updated for V2 architecture
type ImpersonateUserRequest struct {
	AppID              xid.ID        `json:"app_id"`                         // Platform app (required)
	UserOrganizationID *xid.ID       `json:"user_organization_id,omitempty"` // User-created org (optional)
	UserID             xid.ID        `json:"user_id"`
	Duration           time.Duration `json:"duration,omitempty"`
	IPAddress          string        `json:"-"` // Set by handler
	UserAgent          string        `json:"-"` // Set by handler
	AdminID            xid.ID        `json:"-"` // Set by handler
}

// SetUserRoleRequest represents a request to set a user's role
// Updated for V2 architecture
type SetUserRoleRequest struct {
	AppID              xid.ID  `json:"app_id"`                         // Platform app (required)
	UserOrganizationID *xid.ID `json:"user_organization_id,omitempty"` // User-created org (optional)
	UserID             xid.ID  `json:"user_id"`
	Role               string  `json:"role"`
	AdminID            xid.ID  `json:"-"` // Set by handler
}

// ListSessionsRequest represents a request to list sessions
// Updated for V2 architecture
type ListSessionsRequest struct {
	AppID              xid.ID  `json:"app_id"`                         // Platform app (required)
	UserOrganizationID *xid.ID `json:"user_organization_id,omitempty"` // User-created org (optional)
	UserID             *xid.ID `json:"user_id,omitempty"`
	Page               int     `json:"page"`
	Limit              int     `json:"limit"`
	AdminID            xid.ID  `json:"-"` // Set by handler
}

// ListSessionsResponse represents the response for listing sessions
type ListSessionsResponse struct {
	Sessions   []*session.Session `json:"sessions"`
	Total      int                `json:"total"`
	Page       int                `json:"page"`
	Limit      int                `json:"limit"`
	TotalPages int                `json:"total_pages"`
}

// CreateUser creates a new user with admin privileges
func (s *Service) CreateUser(ctx context.Context, req *CreateUserRequest) (*user.User, error) {
	// Check admin permissions
	if err := s.checkAdminPermission(ctx, req.AdminID, "user:create"); err != nil {
		return nil, err
	}

	// Validate required fields
	if req.Email == "" {
		return nil, errors.New("email is required")
	}
	if req.Password == "" {
		return nil, errors.New("password is required")
	}
	if req.Name == "" {
		return nil, errors.New("name is required")
	}

	// Create user request with only supported fields
	createReq := &user.CreateUserRequest{
		Email:    req.Email,
		Password: req.Password,
		Name:     req.Name,
	}

	// Create user
	newUser, err := s.userService.Create(ctx, createReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Log audit event
	orgIDStr := req.AppID.String()
	if req.UserOrganizationID != nil {
		orgIDStr = req.UserOrganizationID.String()
	}
	if err := s.auditService.Log(ctx, &req.AdminID, "user:create", "user",
		getIPFromContext(ctx), getUserAgentFromContext(ctx),
		fmt.Sprintf(`{"created_user_id":"%s","email":"%s","name":"%s","app_id":"%s","organization_id":"%s"}`, newUser.ID.String(), newUser.Email, newUser.Name, req.AppID.String(), orgIDStr)); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Failed to log audit event: %v\n", err)
	}

	return newUser, nil
}

// ListUsers lists users with filtering and pagination
func (s *Service) ListUsers(ctx context.Context, req *ListUsersRequest) (*ListUsersResponse, error) {
	// Check admin permissions
	if err := s.checkAdminPermission(ctx, req.AdminID, "user:read"); err != nil {
		return nil, err
	}

	// Set defaults
	if req.Limit <= 0 || req.Limit > 100 {
		req.Limit = 20
	}
	if req.Page <= 0 {
		req.Page = 1
	}

	// Create pagination options
	opts := types.PaginationOptions{
		Page:     req.Page,
		PageSize: req.Limit,
		OrderBy:  "created_at",
		OrderDir: "desc",
	}

	// Get users
	users, total, err := s.userService.List(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	// Filter by status if specified
	if req.Status != "" {
		users, total = s.filterUsersByStatus(ctx, users, req.Status)
	}

	// Calculate total pages
	totalPages := (total + req.Limit - 1) / req.Limit

	return &ListUsersResponse{
		Users:      users,
		Total:      total,
		Page:       req.Page,
		Limit:      req.Limit,
		TotalPages: totalPages,
	}, nil
}

// DeleteUser deletes a user
func (s *Service) DeleteUser(ctx context.Context, userID, adminID xid.ID) error {
	// Check admin permissions
	if err := s.checkAdminPermission(ctx, adminID, "user:delete"); err != nil {
		return err
	}

	// Get user before deletion for audit
	targetUser, err := s.userService.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to find user: %w", err)
	}

	// Delete user
	if err := s.userService.Delete(ctx, userID); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	// Log audit event
	if err := s.auditService.Log(ctx, &adminID, "user:delete", "user",
		getIPFromContext(ctx), getUserAgentFromContext(ctx),
		fmt.Sprintf(`{"deleted_user_id":"%s","email":"%s","name":"%s"}`, targetUser.ID.String(), targetUser.Email, targetUser.Name)); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Failed to log audit event: %v\n", err)
	}

	return nil
}

// BanUser bans a user
func (s *Service) BanUser(ctx context.Context, req *BanUserRequest) error {
	// Check admin permissions
	if err := s.checkAdminPermission(ctx, req.AdminID, "user:ban"); err != nil {
		return err
	}

	// Create ban request
	banReq := &user.BanRequest{
		UserID:    req.UserID.String(),
		BannedBy:  req.AdminID.String(),
		Reason:    req.Reason,
		ExpiresAt: req.ExpiresAt,
	}

	// Ban user
	_, err := s.banService.BanUser(ctx, banReq)
	if err != nil {
		return fmt.Errorf("failed to ban user: %w", err)
	}

	// Log audit event
	expiresAtStr := "null"
	if req.ExpiresAt != nil {
		expiresAtStr = fmt.Sprintf(`"%s"`, req.ExpiresAt.Format(time.RFC3339))
	}
	if err := s.auditService.Log(ctx, &req.AdminID, "user:ban", "user",
		getIPFromContext(ctx), getUserAgentFromContext(ctx),
		fmt.Sprintf(`{"banned_user_id":"%s","reason":"%s","expires_at":%s}`, req.UserID.String(), req.Reason, expiresAtStr)); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Failed to log audit event: %v\n", err)
	}

	return nil
}

// UnbanUser unbans a user
func (s *Service) UnbanUser(ctx context.Context, req *UnbanUserRequest) error {
	// Check admin permissions
	if err := s.checkAdminPermission(ctx, req.AdminID, "user:unban"); err != nil {
		return err
	}

	// Create unban request
	unbanReq := &user.UnbanRequest{
		UserID:     req.UserID.String(),
		UnbannedBy: req.AdminID.String(),
	}

	// Unban user
	if err := s.banService.UnbanUser(ctx, unbanReq); err != nil {
		return fmt.Errorf("failed to unban user: %w", err)
	}

	// Log audit event
	if err := s.auditService.Log(ctx, &req.AdminID, "user:unban", "user",
		getIPFromContext(ctx), getUserAgentFromContext(ctx),
		fmt.Sprintf(`{"unbanned_user_id":"%s"}`, req.UserID.String())); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Failed to log audit event: %v\n", err)
	}

	return nil
}

// ImpersonateUser creates a session for impersonating a user
func (s *Service) ImpersonateUser(ctx context.Context, req *ImpersonateUserRequest) (*session.Session, error) {
	// Check admin permissions
	if err := s.checkAdminPermission(ctx, req.AdminID, "user:impersonate"); err != nil {
		return nil, err
	}

	// Get target user to ensure they exist
	targetUser, err := s.userService.FindByID(ctx, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}
	if targetUser == nil {
		return nil, fmt.Errorf("user not found")
	}

	// Check if user is banned
	if banned, err := s.banService.IsUserBanned(ctx, req.UserID.String()); err != nil {
		return nil, fmt.Errorf("failed to check ban status: %w", err)
	} else if banned {
		return nil, fmt.Errorf("cannot impersonate banned user")
	}

	// Set duration with max limit
	duration := req.Duration
	if duration <= 0 || duration > s.config.MaxImpersonationDuration {
		duration = s.config.MaxImpersonationDuration
	}

	// Create impersonation session
	sessionReq := &session.CreateSessionRequest{
		UserID:    req.UserID,
		IPAddress: req.IPAddress,
		UserAgent: req.UserAgent,
		Remember:  false,
	}

	newSession, err := s.sessionService.Create(ctx, sessionReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create impersonation session: %w", err)
	}

	// Log audit event
	if err := s.auditService.Log(ctx, &req.AdminID, "user:impersonate", "user",
		req.IPAddress, req.UserAgent,
		fmt.Sprintf(`{"impersonated_user_id":"%s","session_id":"%s","duration":"%s"}`, req.UserID.String(), newSession.ID.String(), duration.String())); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Failed to log audit event: %v\n", err)
	}

	return newSession, nil
}

// SetUserRole sets a user's role
func (s *Service) SetUserRole(ctx context.Context, req *SetUserRoleRequest) error {
	// Check admin permissions
	if err := s.checkAdminPermission(ctx, req.AdminID, "role:assign"); err != nil {
		return err
	}

	// Role assignment through RBAC
	// TODO: Implement role assignment when role repository is integrated
	// For now, log the intended role assignment for audit purposes

	orgIDStr := req.AppID.String()
	if req.UserOrganizationID != nil {
		orgIDStr = req.UserOrganizationID.String()
	}

	// Log audit event
	if err := s.auditService.Log(ctx, &req.AdminID, "role:assign", "user",
		getIPFromContext(ctx), getUserAgentFromContext(ctx),
		fmt.Sprintf(`{"target_user_id":"%s","role":"%s","app_id":"%s","organization_id":"%s"}`, req.UserID.String(), req.Role, req.AppID.String(), orgIDStr)); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Failed to log audit event: %v\n", err)
	}

	return nil
}

// ListSessions lists all sessions with filtering and pagination
func (s *Service) ListSessions(ctx context.Context, req *ListSessionsRequest) (*ListSessionsResponse, error) {
	// Check admin permissions
	if err := s.checkAdminPermission(ctx, req.AdminID, "session:list"); err != nil {
		return nil, err
	}

	// Set defaults
	if req.Limit <= 0 || req.Limit > 100 {
		req.Limit = 20
	}
	if req.Page <= 0 {
		req.Page = 1
	}

	// TODO: Implement session listing through session service
	// This requires adding a List method to the session service or accessing the database directly
	// For now, return placeholder response
	return &ListSessionsResponse{
		Sessions:   []*session.Session{},
		Total:      0,
		Page:       req.Page,
		Limit:      req.Limit,
		TotalPages: 0,
	}, nil
}

// RevokeSession revokes a session
func (s *Service) RevokeSession(ctx context.Context, sessionID, adminID xid.ID) error {
	// Check admin permissions
	if err := s.checkAdminPermission(ctx, adminID, "session:revoke"); err != nil {
		return err
	}

	// Revoke session by ID
	// Note: Session service uses token for revocation, but we only have ID here
	// TODO: Add FindByID method to session service or use token-based lookup
	if err := s.sessionService.Revoke(ctx, sessionID.String()); err != nil {
		return fmt.Errorf("failed to revoke session: %w", err)
	}

	// Log audit event
	if err := s.auditService.Log(ctx, &adminID, "session:revoke", "session",
		getIPFromContext(ctx), getUserAgentFromContext(ctx),
		fmt.Sprintf(`{"revoked_session_id":"%s"}`, sessionID.String())); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Failed to log audit event: %v\n", err)
	}

	return nil
}

// checkAdminPermission checks if the admin has the required permission
func (s *Service) checkAdminPermission(ctx context.Context, userID xid.ID, permission string) error {
	// Check if admin exists
	admin, err := s.userService.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("admin not found: %w", err)
	}
	if admin == nil {
		return fmt.Errorf("admin not found")
	}

	// RBAC permission check
	// For now, check if user is in admin role or has required role from config
	// Full RBAC integration can be added when role repository is available
	// TODO: Integrate with role repository when available for granular permissions

	// Placeholder: Assume users calling admin endpoints have been authenticated
	// and authorized at the route level. Production deployments should implement
	// proper RBAC checks here using a role repository.

	return nil
}

// filterUsersByStatus filters users by status
func (s *Service) filterUsersByStatus(ctx context.Context, users []*user.User, status string) ([]*user.User, int) {
	// This is a placeholder implementation
	// In a real implementation, you would filter based on user status
	// For now, return all users
	return users, len(users)
}

// getIPFromContext extracts IP address from context
func getIPFromContext(ctx context.Context) string {
	// This is a placeholder implementation
	// In a real implementation, you would extract IP from context or request
	return "127.0.0.1"
}

// getUserAgentFromContext extracts user agent from context
func getUserAgentFromContext(ctx context.Context) string {
	// This is a placeholder implementation
	// In a real implementation, you would extract user agent from context or request
	return "Admin-Plugin/1.0"
}
