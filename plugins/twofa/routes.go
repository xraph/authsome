package twofa

import "github.com/xraph/forge"

// Register registers 2FA routes under basePath.
func Register(router forge.Router, basePath string, h *Handler) error {
	grp := router.Group(basePath)
	if err := grp.POST("/2fa/enable", h.Enable); err != nil { return err }
	if err := grp.POST("/2fa/verify", h.Verify); err != nil { return err }
	if err := grp.POST("/2fa/disable", h.Disable); err != nil { return err }
	if err := grp.POST("/2fa/generate-backup-codes", h.GenerateBackupCodes); err != nil { return err }
	if err := grp.POST("/2fa/send-otp", h.SendOTP); err != nil { return err }
	if err := grp.POST("/2fa/status", h.Status); err != nil { return err }
	return nil
}
