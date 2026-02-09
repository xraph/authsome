package user

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/internal/crypto"
	"github.com/xraph/authsome/schema"
)

// BanRepository defines the interface for user ban operations.
type BanRepository interface {
	// Create a new ban record
	CreateBan(ctx context.Context, ban *schema.UserBan) error

	// Find active ban for a user
	FindActiveBan(ctx context.Context, userID string) (*schema.UserBan, error)

	// Find all bans for a user (including inactive)
	FindBansByUser(ctx context.Context, userID string) ([]*schema.UserBan, error)

	// Update ban record (for unbanning)
	UpdateBan(ctx context.Context, ban *schema.UserBan) error

	// Find ban by ID
	FindBanByID(ctx context.Context, banID string) (*schema.UserBan, error)
}

// Ban represents a user ban with business logic.
type Ban struct {
	ID           string
	UserID       xid.ID     `json:"userID"`
	BannedByID   xid.ID     `json:"bannedByID"`
	UnbannedByID *xid.ID    `json:"unbannedByID,omitempty"`
	Reason       string     `json:"reason"`
	IsActive     bool       `json:"isActive"`
	ExpiresAt    *time.Time `json:"expiresAt,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
	UnbannedAt   *time.Time `json:"unbannedAt,omitempty"`
}

// BanRequest represents a request to ban a user.
type BanRequest struct {
	UserID    xid.ID `json:"userID"`
	Reason    string `json:"reason"`
	BannedBy  xid.ID `json:"bannedBy"`
	ExpiresAt *time.Time
}

// UnbanRequest represents a request to unban a user.
type UnbanRequest struct {
	UserID     xid.ID `json:"userID"`
	UnbannedBy xid.ID `json:"unbannedBy"`
	Reason     string
}

// BanService handles user banning operations.
type BanService struct {
	banRepo      BanRepository
	hookRegistry any // Hook registry for lifecycle events
}

// NewBanService creates a new ban service.
func NewBanService(banRepo BanRepository) *BanService {
	return &BanService{
		banRepo: banRepo,
	}
}

// SetHookRegistry sets the hook registry for executing lifecycle hooks.
func (s *BanService) SetHookRegistry(registry any) {
	s.hookRegistry = registry
}

// BanUser bans a user with the given reason and optional expiration.
func (s *BanService) BanUser(ctx context.Context, req *BanRequest) (*Ban, error) {
	if req.UserID.IsNil() {
		return nil, errors.New("user ID is required")
	}

	if req.Reason == "" {
		return nil, errors.New("ban reason is required")
	}

	if req.BannedBy.IsNil() {
		return nil, errors.New("banned by user ID is required")
	}

	// Check if user is already banned
	existingBan, err := s.banRepo.FindActiveBan(ctx, req.UserID.String())
	if err != nil {
		return nil, err
	}

	if existingBan != nil && existingBan.IsCurrentlyActive() {
		return nil, errors.New("user is already banned")
	}

	// Create new ban record
	ban := &schema.UserBan{
		UserID:     req.UserID,
		BannedByID: req.BannedBy,
		Reason:     req.Reason,
		IsActive:   true,
		ExpiresAt:  req.ExpiresAt,
		AuditableModel: schema.AuditableModel{
			ID:        xid.New(),
			CreatedBy: req.BannedBy,
			UpdatedBy: req.BannedBy,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	if err := s.banRepo.CreateBan(ctx, ban); err != nil {
		return nil, err
	}

	// Execute account suspended lifecycle hook
	if s.hookRegistry != nil {
		if registry, ok := s.hookRegistry.(interface {
			ExecuteOnAccountSuspended(context.Context, xid.ID, string) error
		}); ok {
			_ = registry.ExecuteOnAccountSuspended(ctx, req.UserID, req.Reason)
		}
	}

	return s.schemaToBan(ban), nil
}

// UnbanUser removes an active ban from a user.
func (s *BanService) UnbanUser(ctx context.Context, req *UnbanRequest) error {
	if req.UserID.IsNil() {
		return errors.New("user ID is required")
	}

	if req.UnbannedBy.IsNil() {
		return errors.New("unbanned by user ID is required")
	}

	// Find active ban
	ban, err := s.banRepo.FindActiveBan(ctx, req.UserID.String())
	if err != nil {
		return err
	}

	if ban == nil {
		return errors.New("no active ban found for user")
	}

	// Update ban record
	now := time.Now()
	ban.IsActive = false
	ban.UnbannedByID = &req.UnbannedBy
	ban.UnbannedAt = &now
	ban.UpdatedAt = now
	ban.UpdatedBy = req.UnbannedBy

	if err := s.banRepo.UpdateBan(ctx, ban); err != nil {
		return err
	}

	// Execute account reactivated lifecycle hook
	if s.hookRegistry != nil {
		if registry, ok := s.hookRegistry.(interface {
			ExecuteOnAccountReactivated(context.Context, xid.ID) error
		}); ok {
			_ = registry.ExecuteOnAccountReactivated(ctx, req.UserID)
		}
	}

	return nil
}

// CheckBan checks if a user is currently banned.
func (s *BanService) CheckBan(ctx context.Context, userID string) (*Ban, error) {
	if userID == "" {
		return nil, errors.New("user ID is required")
	}

	ban, err := s.banRepo.FindActiveBan(ctx, userID)
	if err != nil {
		return nil, err
	}

	if ban == nil {
		return nil, nil // No ban found
	}

	// Check if ban is still active
	if !ban.IsCurrentlyActive() {
		return nil, nil // Ban has expired or been deactivated
	}

	return s.schemaToBan(ban), nil
}

// GetUserBans returns all bans for a user (including inactive).
func (s *BanService) GetUserBans(ctx context.Context, userID string) ([]*Ban, error) {
	if userID == "" {
		return nil, errors.New("user ID is required")
	}

	schemaBans, err := s.banRepo.FindBansByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	bans := make([]*Ban, len(schemaBans))
	for i, schemaBan := range schemaBans {
		bans[i] = s.schemaToBan(schemaBan)
	}

	return bans, nil
}

// IsUserBanned checks if a user is currently banned (convenience method).
func (s *BanService) IsUserBanned(ctx context.Context, userID string) (bool, error) {
	ban, err := s.CheckBan(ctx, userID)
	if err != nil {
		return false, err
	}

	return ban != nil, nil
}

// schemaToBan converts a schema.UserBan to a Ban.
func (s *BanService) schemaToBan(schemaBan *schema.UserBan) *Ban {
	return &Ban{
		ID:           schemaBan.ID.String(),
		UserID:       schemaBan.UserID,
		BannedByID:   schemaBan.BannedByID,
		UnbannedByID: schemaBan.UnbannedByID,
		Reason:       schemaBan.Reason,
		IsActive:     schemaBan.IsActive,
		ExpiresAt:    schemaBan.ExpiresAt,
		CreatedAt:    schemaBan.CreatedAt,
		UpdatedAt:    schemaBan.UpdatedAt,
		UnbannedAt:   schemaBan.UnbannedAt,
	}
}

// generateBanID generates a unique ban ID.
func generateBanID() (string, error) {
	// Use the same ID generation logic as other entities
	id, err := crypto.GenerateID()
	if err != nil {
		return "", fmt.Errorf("failed to generate ban ID: %w", err)
	}

	return "ban_" + id, nil
}
