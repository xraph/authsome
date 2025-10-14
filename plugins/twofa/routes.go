package twofa

import "github.com/xraph/forge"

// Register registers 2FA routes under basePath
func Register(app *forge.App, basePath string, h *Handler) {
    grp := app.Group(basePath)
    grp.POST("/2fa/enable", h.Enable)
    grp.POST("/2fa/verify", h.Verify)
    grp.POST("/2fa/disable", h.Disable)
    grp.POST("/2fa/generate-backup-codes", h.GenerateBackupCodes)
    grp.POST("/2fa/send-otp", h.SendOTP)
    grp.POST("/2fa/status", h.Status)
}