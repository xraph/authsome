package twofa

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/registry"
	repo "github.com/xraph/authsome/repository"
	"github.com/xraph/authsome/schema"
	"github.com/xraph/forge"
)

// Plugin implements the plugins.Plugin interface for Two-Factor Authentication
type Plugin struct {
	service *Service
	db      *bun.DB
}

func NewPlugin() *Plugin { return &Plugin{} }

func (p *Plugin) ID() string { return "twofa" }

func (p *Plugin) Init(dep interface{}) error {
	type authInstance interface {
		GetDB() *bun.DB
	}

	authInst, ok := dep.(authInstance)
	if !ok {
		return fmt.Errorf("twofa plugin requires auth instance with GetDB method")
	}

	db := authInst.GetDB()
	if db == nil {
		return fmt.Errorf("database not available for twofa plugin")
	}

	p.db = db
	// Wire repository-backed service
	p.service = NewService(repo.NewTwoFARepository(db))
	return nil
}

// RegisterRoutes registers 2FA endpoints under the auth base
func (p *Plugin) RegisterRoutes(router forge.Router) error {
	// Router is already scoped to the correct basePath by the auth mount
	h := NewHandler(p.service)
	router.POST("/2fa/enable", h.Enable,
		forge.WithName("twofa.enable"),
		forge.WithSummary("Enable 2FA"),
		forge.WithDescription("Enables two-factor authentication for a user. Returns TOTP URI for QR code generation"),
		forge.WithResponseSchema(200, "2FA enabled", TwoFAEnableResponse{}),
		forge.WithResponseSchema(400, "Invalid request", TwoFAErrorResponse{}),
		forge.WithTags("2FA", "TOTP"),
		forge.WithValidation(true),
	)
	router.POST("/2fa/verify", h.Verify,
		forge.WithName("twofa.verify"),
		forge.WithSummary("Verify 2FA code"),
		forge.WithDescription("Verifies a 2FA code (TOTP or backup code) for authentication. Optionally marks device as trusted"),
		forge.WithResponseSchema(200, "Code verified", TwoFAStatusResponse{}),
		forge.WithResponseSchema(400, "Invalid request", TwoFAErrorResponse{}),
		forge.WithResponseSchema(401, "Invalid code", TwoFAErrorResponse{}),
		forge.WithTags("2FA", "Verification"),
		forge.WithValidation(true),
	)
	router.POST("/2fa/disable", h.Disable,
		forge.WithName("twofa.disable"),
		forge.WithSummary("Disable 2FA"),
		forge.WithDescription("Disables two-factor authentication for a user"),
		forge.WithResponseSchema(200, "2FA disabled", TwoFAStatusResponse{}),
		forge.WithResponseSchema(400, "Invalid request", TwoFAErrorResponse{}),
		forge.WithTags("2FA", "Management"),
		forge.WithValidation(true),
	)
	router.POST("/2fa/generate-backup-codes", h.GenerateBackupCodes,
		forge.WithName("twofa.backupcodes.generate"),
		forge.WithSummary("Generate backup codes"),
		forge.WithDescription("Generates new backup codes for 2FA recovery. Previous codes are invalidated"),
		forge.WithResponseSchema(200, "Backup codes generated", TwoFABackupCodesResponse{}),
		forge.WithResponseSchema(400, "Invalid request", TwoFAErrorResponse{}),
		forge.WithTags("2FA", "BackupCodes"),
		forge.WithValidation(true),
	)
	router.POST("/2fa/send-otp", h.SendOTP,
		forge.WithName("twofa.sendotp"),
		forge.WithSummary("Send OTP code"),
		forge.WithDescription("Sends a one-time password via email or SMS for 2FA. Returns code in dev mode"),
		forge.WithResponseSchema(200, "OTP sent", TwoFASendOTPResponse{}),
		forge.WithResponseSchema(400, "Invalid request", TwoFAErrorResponse{}),
		forge.WithTags("2FA", "OTP"),
		forge.WithValidation(true),
	)
	router.POST("/2fa/status", h.Status,
		forge.WithName("twofa.status"),
		forge.WithSummary("Get 2FA status"),
		forge.WithDescription("Retrieves 2FA status for a user including enabled state, method, and trusted device status"),
		forge.WithResponseSchema(200, "Status retrieved", TwoFAStatusDetailResponse{}),
		forge.WithResponseSchema(400, "Invalid request", TwoFAErrorResponse{}),
		forge.WithTags("2FA", "Status"),
		forge.WithValidation(true),
	)
	return nil
}

func (p *Plugin) RegisterHooks(_ *hooks.HookRegistry) error { return nil }

func (p *Plugin) RegisterServiceDecorators(_ *registry.ServiceRegistry) error { return nil }
func (p *Plugin) Migrate() error {
	if p.db == nil {
		return nil
	}
	ctx := context.Background()
	// Create required tables if not exist
	if _, err := p.db.NewCreateTable().Model((*schema.TwoFASecret)(nil)).IfNotExists().Exec(ctx); err != nil {
		return err
	}
	if _, err := p.db.NewCreateTable().Model((*schema.BackupCode)(nil)).IfNotExists().Exec(ctx); err != nil {
		return err
	}
	if _, err := p.db.NewCreateTable().Model((*schema.TrustedDevice)(nil)).IfNotExists().Exec(ctx); err != nil {
		return err
	}
	if _, err := p.db.NewCreateTable().Model((*schema.OTPCode)(nil)).IfNotExists().Exec(ctx); err != nil {
		return err
	}
	return nil
}

// Response types for 2FA routes
type TwoFAErrorResponse struct {
	Error string `json:"error" example:"Error message"`
}

type TwoFAStatusResponse struct {
	Status string `json:"status" example:"verified"`
}

type TwoFAEnableResponse struct {
	Status  string `json:"status" example:"2fa_enabled"`
	TOTPURI string `json:"totp_uri,omitempty" example:"otpauth://totp/AuthSome:user@example.com?secret=JBSWY3DPEHPK3PXP&issuer=AuthSome"`
}

type TwoFABackupCodesResponse struct {
	Codes []string `json:"codes" example:"12345678,87654321"`
}

type TwoFASendOTPResponse struct {
	Status string `json:"status" example:"otp_sent"`
	Code   string `json:"code,omitempty" example:"123456"`
}

type TwoFAStatusDetailResponse struct {
	Enabled bool   `json:"enabled" example:"true"`
	Method  string `json:"method,omitempty" example:"totp"`
	Trusted bool   `json:"trusted,omitempty" example:"false"`
}
