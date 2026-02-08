package bridge

import (
	"context"
	"time"

	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/authsome/schema"
	"github.com/xraph/forge"
	"github.com/xraph/forgeui/bridge"
)

// =============================================================================
// Input/Output Types
// =============================================================================

// GetDeviceCodesInput is the input for listing device codes
type GetDeviceCodesInput struct {
	AppID    string `json:"appId"`
	Status   string `json:"status,omitempty"` // pending, authorized, denied, expired, consumed
	Page     int    `json:"page,omitempty"`
	PageSize int    `json:"pageSize,omitempty"`
}

// GetDeviceCodesOutput is the output for listing device codes
type GetDeviceCodesOutput struct {
	Data       []DeviceCodeDTO `json:"data"`
	Pagination *PaginationDTO  `json:"pagination"`
}

// DeviceCodeDTO represents a device code in API responses
type DeviceCodeDTO struct {
	ID                  string     `json:"id"`
	DeviceCode          string     `json:"deviceCode"`          // Masked for security
	UserCode            string     `json:"userCode"`
	ClientID            string     `json:"clientId"`
	ClientName          string     `json:"clientName"`
	Scope               string     `json:"scope"`
	Status              string     `json:"status"`
	VerificationURI     string     `json:"verificationUri"`
	ExpiresAt           time.Time  `json:"expiresAt"`
	CreatedAt           time.Time  `json:"createdAt"`
	AuthorizedAt        *time.Time `json:"authorizedAt,omitempty"`
	ConsumedAt          *time.Time `json:"consumedAt,omitempty"`
	PollCount           int        `json:"pollCount"`
	TimeRemaining       int64      `json:"timeRemaining"` // Seconds until expiration
}

// RevokeDeviceCodeInput is the input for revoking a device code
type RevokeDeviceCodeInput struct {
	UserCode string `json:"userCode"`
}

// RevokeDeviceCodeOutput is the output for revoking a device code
type RevokeDeviceCodeOutput struct {
	Success bool `json:"success"`
}

// CleanupExpiredDeviceCodesInput is the input for cleanup
type CleanupExpiredDeviceCodesInput struct {
	AppID string `json:"appId"`
}

// CleanupExpiredDeviceCodesOutput is the output for cleanup
type CleanupExpiredDeviceCodesOutput struct {
	Data struct {
		ExpiredCount  int `json:"expiredCount"`
		ConsumedCount int `json:"consumedCount"`
	} `json:"data"`
}

// =============================================================================
// Bridge Functions
// =============================================================================

// GetDeviceCodes lists device authorization codes
func (bm *BridgeManager) GetDeviceCodes(ctx bridge.Context, input GetDeviceCodesInput) (*GetDeviceCodesOutput, error) {
	goCtx, _, appID, err := bm.buildContextWithAppID(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	// Check if device flow service is enabled
	if bm.service.GetDeviceFlowService() == nil {
		return nil, errs.BadRequest("device flow is not enabled")
	}

	// Get environment ID
	envID, _ := contexts.GetEnvironmentID(goCtx)

	// Set pagination defaults
	page := input.Page
	if page < 1 {
		page = 1
	}
	pageSize := input.PageSize
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// Query device codes
	var deviceCodes []*schema.DeviceCode
	var total int64

	if input.Status != "" {
		// Filter by status
		deviceCodes, err = bm.deviceCodeRepo.ListByAppEnvAndStatus(goCtx, appID, envID, input.Status, page, pageSize)
		if err != nil {
			bm.logger.Error("failed to list device codes by status",
				forge.F("error", err.Error()),
				forge.F("appId", appID.String()),
				forge.F("status", input.Status))
			return nil, errs.InternalServerError("failed to list device codes", err)
		}
		total, _ = bm.deviceCodeRepo.CountByAppEnvAndStatus(goCtx, appID, envID, input.Status)
	} else {
		// List all
		deviceCodes, err = bm.deviceCodeRepo.ListByAppAndEnv(goCtx, appID, envID, page, pageSize)
		if err != nil {
			bm.logger.Error("failed to list device codes",
				forge.F("error", err.Error()),
				forge.F("appId", appID.String()))
			return nil, errs.InternalServerError("failed to list device codes", err)
		}
		total, _ = bm.deviceCodeRepo.CountByAppAndEnv(goCtx, appID, envID)
	}

	// Convert to DTOs
	data := make([]DeviceCodeDTO, len(deviceCodes))
	for i, dc := range deviceCodes {
		data[i] = deviceCodeToDTO(dc, bm)
	}

	return &GetDeviceCodesOutput{
		Data: data,
		Pagination: &PaginationDTO{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: (total + int64(pageSize) - 1) / int64(pageSize),
		},
	}, nil
}

// RevokeDeviceCode manually revokes a device code
func (bm *BridgeManager) RevokeDeviceCode(ctx bridge.Context, input RevokeDeviceCodeInput) (*RevokeDeviceCodeOutput, error) {
	goCtx, _, _, err := bm.buildContext(ctx)
	if err != nil {
		return nil, err
	}

	// Check if device flow service is enabled
	if bm.service.GetDeviceFlowService() == nil {
		return nil, errs.BadRequest("device flow is not enabled")
	}

	// Find device code by user code
	deviceCode, err := bm.deviceCodeRepo.FindByUserCode(goCtx, input.UserCode)
	if err != nil {
		return nil, errs.NotFound("device code not found")
	}

	// Update status to denied (effectively revoking it)
	deviceCode.Status = "denied"
	// Note: UpdatedAt is set automatically by AuditableModel

	if err := bm.deviceCodeRepo.Update(goCtx, deviceCode); err != nil {
		bm.logger.Error("failed to revoke device code",
			forge.F("error", err.Error()),
			forge.F("userCode", input.UserCode))
		return nil, errs.InternalServerError("failed to revoke device code", err)
	}

	bm.logger.Info("device code manually revoked",
		forge.F("userCode", input.UserCode))

	return &RevokeDeviceCodeOutput{
		Success: true,
	}, nil
}

// CleanupExpiredDeviceCodes triggers cleanup of expired device codes
func (bm *BridgeManager) CleanupExpiredDeviceCodes(ctx bridge.Context, input CleanupExpiredDeviceCodesInput) (*CleanupExpiredDeviceCodesOutput, error) {
	goCtx, _, _, err := bm.buildContextWithAppID(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	// Check if device flow service is enabled
	deviceFlowSvcIface := bm.service.GetDeviceFlowService()
	if deviceFlowSvcIface == nil {
		return nil, errs.BadRequest("device flow is not enabled")
	}

	// Type assert to deviceflow.Service
	type deviceFlowService interface {
		CleanupExpiredCodes(ctx context.Context) (int, error)
		CleanupOldConsumedCodes(ctx context.Context, olderThan time.Duration) (int, error)
	}
	
	deviceFlowSvc, ok := deviceFlowSvcIface.(deviceFlowService)
	if !ok {
		return nil, errs.InternalServerError("invalid device flow service", nil)
	}

	// Clean up expired codes
	expiredCount, err := deviceFlowSvc.CleanupExpiredCodes(goCtx)
	if err != nil {
		bm.logger.Error("failed to cleanup expired device codes",
			forge.F("error", err.Error()))
		return nil, errs.InternalServerError("failed to cleanup expired codes", err)
	}

	// Clean up old consumed codes (older than 7 days)
	consumedCount, err := deviceFlowSvc.CleanupOldConsumedCodes(goCtx, 7*24*time.Hour)
	if err != nil {
		bm.logger.Error("failed to cleanup old consumed device codes",
			forge.F("error", err.Error()))
		// Continue even if this fails
		consumedCount = 0
	}

	bm.logger.Info("device codes cleanup completed",
		forge.F("expired", expiredCount),
		forge.F("consumed", consumedCount))

	return &CleanupExpiredDeviceCodesOutput{
		Data: struct {
			ExpiredCount  int `json:"expiredCount"`
			ConsumedCount int `json:"consumedCount"`
		}{
			ExpiredCount:  expiredCount,
			ConsumedCount: consumedCount,
		},
	}, nil
}

// =============================================================================
// Helper Functions
// =============================================================================

// deviceCodeToDTO converts a schema.DeviceCode to DeviceCodeDTO
func deviceCodeToDTO(dc *schema.DeviceCode, bm *BridgeManager) DeviceCodeDTO {
	dto := DeviceCodeDTO{
		ID:              dc.ID.String(),
		DeviceCode:      maskDeviceCode(dc.DeviceCode),
		UserCode:        dc.FormattedUserCode(), // Display formatted version
		ClientID:        dc.ClientID,
		Scope:           dc.Scope,
		Status:          dc.Status,
		VerificationURI: dc.VerificationURI,
		ExpiresAt:       dc.ExpiresAt,
		CreatedAt:       dc.CreatedAt,
		PollCount:       dc.PollCount,
	}

	// For authorized/consumed status, use UpdatedAt as a proxy for when it was authorized/consumed
	if dc.Status == "authorized" || dc.Status == "consumed" {
		dto.AuthorizedAt = &dc.UpdatedAt
	}
	if dc.Status == "consumed" {
		dto.ConsumedAt = &dc.UpdatedAt
	}

	// Calculate time remaining
	timeRemaining := int64(time.Until(dc.ExpiresAt).Seconds())
	if timeRemaining < 0 {
		timeRemaining = 0
	}
	dto.TimeRemaining = timeRemaining

	// Try to get client name
	// Note: This requires a context, which we'd need to pass through
	// For now, just use client ID
	dto.ClientName = dc.ClientID

	return dto
}

// maskDeviceCode masks the device code for security
func maskDeviceCode(deviceCode string) string {
	if len(deviceCode) <= 8 {
		return "****"
	}
	// Show first 4 and last 4 characters
	return deviceCode[:4] + "..." + deviceCode[len(deviceCode)-4:]
}
