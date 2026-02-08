package deviceflow

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/internal/errs"
	repo "github.com/xraph/authsome/repository"
	"github.com/xraph/authsome/schema"
)

// Config holds device flow configuration
type Config struct {
	Enabled          bool          `json:"enabled"`
	DeviceCodeExpiry time.Duration `json:"deviceCodeExpiry"` // e.g., 10 minutes
	UserCodeLength   int           `json:"userCodeLength"`   // e.g., 8 characters
	UserCodeFormat   string        `json:"userCodeFormat"`   // e.g., "XXXX-XXXX"
	PollingInterval  int           `json:"pollingInterval"`  // e.g., 5 seconds
	VerificationURI  string        `json:"verificationUri"`  // e.g., "/device"
	AllowedClients   []string      `json:"allowedClients"`   // optional whitelist
	MaxPollAttempts  int           `json:"maxPollAttempts"`  // max polls before requiring new request
	CleanupInterval  time.Duration `json:"cleanupInterval"`  // how often to clean up expired codes
}

// DefaultConfig returns the default device flow configuration
func DefaultConfig() Config {
	return Config{
		Enabled:          true,
		DeviceCodeExpiry: 10 * time.Minute,
		UserCodeLength:   8,
		UserCodeFormat:   "XXXX-XXXX",
		PollingInterval:  5,
		VerificationURI:  "/device",
		AllowedClients:   []string{}, // empty = all clients allowed
		MaxPollAttempts:  120,        // 10 minutes / 5 seconds = 120 polls max
		CleanupInterval:  5 * time.Minute,
	}
}

// Service handles device flow business logic
type Service struct {
	repo          *repo.DeviceCodeRepository
	codeGenerator *CodeGenerator
	config        Config
}

// NewService creates a new device flow service
func NewService(repo *repo.DeviceCodeRepository, config Config) *Service {
	// Set defaults if not provided
	if config.DeviceCodeExpiry == 0 {
		config.DeviceCodeExpiry = 10 * time.Minute
	}
	if config.PollingInterval == 0 {
		config.PollingInterval = 5
	}
	if config.UserCodeLength == 0 {
		config.UserCodeLength = 8
	}
	if config.UserCodeFormat == "" {
		config.UserCodeFormat = "XXXX-XXXX"
	}
	if config.MaxPollAttempts == 0 {
		config.MaxPollAttempts = 120
	}

	return &Service{
		repo:          repo,
		codeGenerator: NewCodeGenerator(config.UserCodeLength, config.UserCodeFormat),
		config:        config,
	}
}

// InitiateDeviceAuthorization generates a new device code and user code
func (s *Service) InitiateDeviceAuthorization(ctx context.Context, clientID string, scope string, appID, envID xid.ID, orgID *xid.ID) (*schema.DeviceCode, error) {
	// Validate client is allowed (if whitelist is configured)
	if len(s.config.AllowedClients) > 0 {
		allowed := false
		for _, allowedClient := range s.config.AllowedClients {
			if allowedClient == clientID {
				allowed = true
				break
			}
		}
		if !allowed {
			return nil, errs.PermissionDenied("device_flow", "client")
		}
	}

	// Generate device code (long, secure)
	deviceCode, err := s.codeGenerator.GenerateDeviceCode()
	if err != nil {
		return nil, fmt.Errorf("failed to generate device code: %w", err)
	}

	// Generate user code (short, human-typable) with collision detection
	var userCode string
	var displayUserCode string // User-friendly display version with hyphens
	maxRetries := 5
	for i := 0; i < maxRetries; i++ {
		displayUserCode, err = s.codeGenerator.GenerateUserCode()
		if err != nil {
			return nil, fmt.Errorf("failed to generate user code: %w", err)
		}

		// Normalize the code for storage (remove hyphens and spaces, uppercase)
		userCode = normalizeUserCode(displayUserCode)

		// Check for collision using normalized code
		existing, err := s.repo.FindByUserCode(ctx, userCode)
		if err != nil {
			return nil, fmt.Errorf("failed to check user code uniqueness: %w", err)
		}
		if existing == nil {
			break // No collision, we're good
		}

		if i == maxRetries-1 {
			return nil, fmt.Errorf("failed to generate unique user code after %d attempts", maxRetries)
		}
	}

	// Create device code record
	dc := &schema.DeviceCode{
		AuditableModel: schema.AuditableModel{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		AppID:           appID,
		EnvironmentID:   envID,
		OrganizationID:  orgID,
		ClientID:        clientID,
		DeviceCode:      deviceCode,
		UserCode:        userCode, // Store normalized version (without hyphens)
		VerificationURI: s.config.VerificationURI,
		ExpiresAt:       time.Now().Add(s.config.DeviceCodeExpiry),
		Interval:        s.config.PollingInterval,
		Scope:           scope,
		Status:          schema.DeviceCodeStatusPending,
		PollCount:       0,
	}

	if err := s.repo.Create(ctx, dc); err != nil {
		return nil, fmt.Errorf("failed to store device code: %w", err)
	}

	return dc, nil
}

// GetDeviceCodeByUserCode retrieves a device code by user code
func (s *Service) GetDeviceCodeByUserCode(ctx context.Context, userCode string) (*schema.DeviceCode, error) {
	dc, err := s.repo.FindByUserCode(ctx, userCode)
	if err != nil {
		return nil, fmt.Errorf("failed to find device code: %w", err)
	}
	if dc == nil {
		return nil, errs.NotFound("device code not found")
	}
	return dc, nil
}

// GetDeviceCodeByDeviceCode retrieves a device code by device code
func (s *Service) GetDeviceCodeByDeviceCode(ctx context.Context, deviceCode string) (*schema.DeviceCode, error) {
	dc, err := s.repo.FindByDeviceCode(ctx, deviceCode)
	if err != nil {
		return nil, fmt.Errorf("failed to find device code: %w", err)
	}
	if dc == nil {
		return nil, errs.NotFound("device code not found")
	}
	return dc, nil
}

// AuthorizeDevice marks a device as authorized by a user
func (s *Service) AuthorizeDevice(ctx context.Context, userCode string, userID, sessionID xid.ID) error {
	// Get device code
	dc, err := s.GetDeviceCodeByUserCode(ctx, userCode)
	if err != nil {
		return err
	}

	// Validate device code is still pending
	if !dc.IsPending() {
		if dc.IsExpired() {
			return errs.BadRequest("device code has expired")
		}
		return errs.BadRequest("device code is not pending authorization")
	}

	// Mark as authorized
	if err := s.repo.AuthorizeDevice(ctx, userCode, userID, sessionID); err != nil {
		return fmt.Errorf("failed to authorize device: %w", err)
	}

	return nil
}

// DenyDevice marks a device authorization as denied
func (s *Service) DenyDevice(ctx context.Context, userCode string) error {
	// Get device code
	dc, err := s.GetDeviceCodeByUserCode(ctx, userCode)
	if err != nil {
		return err
	}

	// Validate device code is still pending
	if !dc.IsPending() {
		if dc.IsExpired() {
			return errs.BadRequest("device code has expired")
		}
		return errs.BadRequest("device code is not pending authorization")
	}

	// Mark as denied
	if err := s.repo.DenyDevice(ctx, userCode); err != nil {
		return fmt.Errorf("failed to deny device: %w", err)
	}

	return nil
}

// PollDeviceCode handles device polling for authorization status
// Returns: dc (*schema.DeviceCode), shouldSlowDown (bool), error
func (s *Service) PollDeviceCode(ctx context.Context, deviceCode string) (*schema.DeviceCode, bool, error) {
	// Get device code
	dc, err := s.repo.FindByDeviceCode(ctx, deviceCode)
	if err != nil {
		return nil, false, fmt.Errorf("failed to find device code: %w", err)
	}
	if dc == nil {
		return nil, false, errs.NotFound("device code not found")
	}

	// Check if expired
	if dc.IsExpired() {
		// Mark as expired
		_ = s.repo.UpdateStatus(ctx, deviceCode, schema.DeviceCodeStatusExpired)
		return dc, false, errs.BadRequest("device code has expired")
	}

	// Check if already consumed
	if dc.Status == schema.DeviceCodeStatusConsumed {
		return dc, false, errs.BadRequest("device code already used")
	}

	// Check if denied
	if dc.Status == schema.DeviceCodeStatusDenied {
		return dc, false, errs.PermissionDenied("device_authorization", "user_denied")
	}

	// Check polling rate
	shouldSlowDown := dc.ShouldSlowDown()

	// Update poll info
	if err := s.repo.UpdatePollInfo(ctx, deviceCode); err != nil {
		return dc, shouldSlowDown, fmt.Errorf("failed to update poll info: %w", err)
	}

	// Refresh device code after update
	dc, err = s.repo.FindByDeviceCode(ctx, deviceCode)
	if err != nil {
		return nil, shouldSlowDown, fmt.Errorf("failed to refresh device code: %w", err)
	}

	// Check max poll attempts
	if dc.PollCount > s.config.MaxPollAttempts {
		return dc, false, errs.BadRequest("maximum poll attempts exceeded")
	}

	return dc, shouldSlowDown, nil
}

// ConsumeDeviceCode marks a device code as consumed after token exchange
func (s *Service) ConsumeDeviceCode(ctx context.Context, deviceCode string) error {
	return s.repo.MarkAsConsumed(ctx, deviceCode)
}

// CleanupExpiredCodes removes expired device codes
func (s *Service) CleanupExpiredCodes(ctx context.Context) (int, error) {
	count, err := s.repo.DeleteExpired(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired codes: %w", err)
	}
	return count, nil
}

// CleanupOldConsumedCodes removes old consumed device codes
func (s *Service) CleanupOldConsumedCodes(ctx context.Context, olderThan time.Duration) (int, error) {
	count, err := s.repo.DeleteOldConsumedCodes(ctx, olderThan)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup old consumed codes: %w", err)
	}
	return count, nil
}

// GetConfig returns the device flow configuration
func (s *Service) GetConfig() Config {
	return s.config
}

// normalizeUserCode normalizes a user code by removing spaces, hyphens, and converting to uppercase
func normalizeUserCode(code string) string {
	// Remove spaces and hyphens
	code = strings.ReplaceAll(code, " ", "")
	code = strings.ReplaceAll(code, "-", "")
	// Convert to uppercase
	return strings.ToUpper(code)
}

// formatUserCode formats a normalized user code to display format (e.g., "BCDFGHJK" -> "BCDF-GHJK")
func formatUserCode(normalized string) string {
	// Default format is "XXXX-XXXX" (8 characters with hyphen in the middle)
	if len(normalized) == 8 {
		return normalized[:4] + "-" + normalized[4:]
	}
	// For other lengths, just return as-is
	return normalized
}
