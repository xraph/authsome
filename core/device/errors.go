package device

import (
	"net/http"

	"github.com/xraph/authsome/internal/errs"
)

// =============================================================================
// DEVICE-SPECIFIC ERROR CODES
// =============================================================================

const (
	CodeDeviceNotFound       = "DEVICE_NOT_FOUND"
	CodeDeviceAlreadyExists  = "DEVICE_ALREADY_EXISTS"
	CodeDeviceCreationFailed = "DEVICE_CREATION_FAILED"
	CodeDeviceUpdateFailed   = "DEVICE_UPDATE_FAILED"
	CodeDeviceDeletionFailed = "DEVICE_DELETION_FAILED"
	CodeInvalidFingerprint   = "INVALID_FINGERPRINT"
	CodeMaxDevicesReached    = "MAX_DEVICES_REACHED"
)

// =============================================================================
// ERROR CONSTRUCTORS
// =============================================================================

// Device lookup errors
func DeviceNotFound() *errs.AuthsomeError {
	return errs.New(CodeDeviceNotFound, "Device not found", http.StatusNotFound)
}

func DeviceAlreadyExists(fingerprint string) *errs.AuthsomeError {
	return errs.New(CodeDeviceAlreadyExists, "Device already exists", http.StatusConflict).
		WithContext("fingerprint", fingerprint)
}

// CRUD operation errors
func DeviceCreationFailed(err error) *errs.AuthsomeError {
	return errs.Wrap(err, CodeDeviceCreationFailed, "Failed to create device", http.StatusInternalServerError)
}

func DeviceUpdateFailed(err error) *errs.AuthsomeError {
	return errs.Wrap(err, CodeDeviceUpdateFailed, "Failed to update device", http.StatusInternalServerError)
}

func DeviceDeletionFailed(err error) *errs.AuthsomeError {
	return errs.Wrap(err, CodeDeviceDeletionFailed, "Failed to delete device", http.StatusInternalServerError)
}

// Validation errors
func InvalidFingerprint() *errs.AuthsomeError {
	return errs.New(CodeInvalidFingerprint, "Invalid device fingerprint", http.StatusBadRequest)
}

func MaxDevicesReached(limit int) *errs.AuthsomeError {
	return errs.New(CodeMaxDevicesReached, "Maximum number of devices reached", http.StatusForbidden).
		WithContext("max_devices", limit)
}

// =============================================================================
// SENTINEL ERRORS (for use with errors.Is)
// =============================================================================

var (
	ErrDeviceNotFound       = &errs.AuthsomeError{Code: CodeDeviceNotFound}
	ErrDeviceAlreadyExists  = &errs.AuthsomeError{Code: CodeDeviceAlreadyExists}
	ErrDeviceCreationFailed = &errs.AuthsomeError{Code: CodeDeviceCreationFailed}
	ErrDeviceUpdateFailed   = &errs.AuthsomeError{Code: CodeDeviceUpdateFailed}
	ErrDeviceDeletionFailed = &errs.AuthsomeError{Code: CodeDeviceDeletionFailed}
	ErrInvalidFingerprint   = &errs.AuthsomeError{Code: CodeInvalidFingerprint}
	ErrMaxDevicesReached    = &errs.AuthsomeError{Code: CodeMaxDevicesReached}
)
