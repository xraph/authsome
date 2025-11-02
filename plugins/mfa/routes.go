package mfa

import "github.com/xraph/forge"

// RegisterRoutes registers all MFA routes with OpenAPI documentation
func RegisterRoutes(router forge.Router, handler *Handler) {
	// ==================== Factor Management ====================

	// POST /mfa/factors/enroll - Enroll a new authentication factor
	router.POST("/mfa/factors/enroll", handler.EnrollFactor,
		forge.WithSummary("Enroll MFA factor"),
		forge.WithDescription("Initiates enrollment of a new multi-factor authentication method (TOTP, SMS, Email, WebAuthn, etc.)"),
		forge.WithTags("MFA", "Factor Management"),
		forge.WithRequestSchema(&EnrollFactorRequest{}),
		forge.WithResponseSchema(200, "Factor enrolled successfully", &FactorEnrollmentResponse{}),
		forge.WithResponseSchema(400, "Invalid request or enrollment failed", &ErrorResponse{}),
		forge.WithResponseSchema(401, "Unauthorized - authentication required", &ErrorResponse{}),
		forge.WithValidation(true),
	)

	// GET /mfa/factors - List all enrolled factors
	router.GET("/mfa/factors", handler.ListFactors,
		forge.WithSummary("List MFA factors"),
		forge.WithDescription("Retrieves all MFA factors enrolled by the authenticated user"),
		forge.WithTags("MFA", "Factor Management"),
		forge.WithRequestSchema(&ListFactorsRequest{}),
		forge.WithResponseSchema(200, "Factors retrieved successfully", &ListFactorsResponse{}),
		forge.WithResponseSchema(401, "Unauthorized - authentication required", &ErrorResponse{}),
		forge.WithResponseSchema(500, "Internal server error", &ErrorResponse{}),
		forge.WithValidation(true),
	)

	// GET /mfa/factors/:id - Get a specific factor
	router.GET("/mfa/factors/:id", handler.GetFactor,
		forge.WithSummary("Get MFA factor"),
		forge.WithDescription("Retrieves details of a specific enrolled MFA factor"),
		forge.WithTags("MFA", "Factor Management"),
		forge.WithRequestSchema(&GetFactorRequest{}),
		forge.WithResponseSchema(200, "Factor retrieved successfully", &Factor{}),
		forge.WithResponseSchema(400, "Invalid factor ID", &ErrorResponse{}),
		forge.WithResponseSchema(401, "Unauthorized - authentication required", &ErrorResponse{}),
		forge.WithResponseSchema(403, "Forbidden - factor belongs to another user", &ErrorResponse{}),
		forge.WithResponseSchema(404, "Factor not found", &ErrorResponse{}),
		forge.WithValidation(true),
	)

	// PUT /mfa/factors/:id - Update a factor
	router.PUT("/mfa/factors/:id", handler.UpdateFactor,
		forge.WithSummary("Update MFA factor"),
		forge.WithDescription("Updates properties of an enrolled MFA factor (name, priority, status)"),
		forge.WithTags("MFA", "Factor Management"),
		forge.WithRequestSchema(&UpdateFactorRequest{}),
		forge.WithResponseSchema(200, "Factor updated successfully", &SuccessResponse{}),
		forge.WithResponseSchema(400, "Invalid request or update failed", &ErrorResponse{}),
		forge.WithResponseSchema(401, "Unauthorized - authentication required", &ErrorResponse{}),
		forge.WithResponseSchema(403, "Forbidden - factor belongs to another user", &ErrorResponse{}),
		forge.WithResponseSchema(404, "Factor not found", &ErrorResponse{}),
		forge.WithValidation(true),
	)

	// DELETE /mfa/factors/:id - Delete a factor
	router.DELETE("/mfa/factors/:id", handler.DeleteFactor,
		forge.WithSummary("Delete MFA factor"),
		forge.WithDescription("Removes an enrolled MFA factor from the user's account"),
		forge.WithTags("MFA", "Factor Management"),
		forge.WithRequestSchema(&DeleteFactorRequest{}),
		forge.WithResponseSchema(200, "Factor deleted successfully", &SuccessResponse{}),
		forge.WithResponseSchema(400, "Invalid request or deletion failed", &ErrorResponse{}),
		forge.WithResponseSchema(401, "Unauthorized - authentication required", &ErrorResponse{}),
		forge.WithResponseSchema(403, "Forbidden - factor belongs to another user", &ErrorResponse{}),
		forge.WithResponseSchema(404, "Factor not found", &ErrorResponse{}),
		forge.WithValidation(true),
	)

	// POST /mfa/factors/:id/verify - Verify an enrolled factor
	router.POST("/mfa/factors/:id/verify", handler.VerifyFactor,
		forge.WithSummary("Verify enrolled factor"),
		forge.WithDescription("Verifies and activates a newly enrolled MFA factor by providing a valid code"),
		forge.WithTags("MFA", "Factor Management"),
		forge.WithRequestSchema(&VerifyEnrolledFactorRequest{}),
		forge.WithResponseSchema(200, "Factor verified and activated", &SuccessResponse{}),
		forge.WithResponseSchema(400, "Invalid code or verification failed", &ErrorResponse{}),
		forge.WithResponseSchema(401, "Unauthorized - authentication required", &ErrorResponse{}),
		forge.WithResponseSchema(403, "Forbidden - factor belongs to another user", &ErrorResponse{}),
		forge.WithResponseSchema(404, "Factor not found", &ErrorResponse{}),
		forge.WithValidation(true),
	)

	// ==================== Challenge & Verification ====================

	// POST /mfa/challenge - Initiate an MFA challenge
	router.POST("/mfa/challenge", handler.InitiateChallenge,
		forge.WithSummary("Initiate MFA challenge"),
		forge.WithDescription("Starts a new MFA challenge session requiring verification of one or more factors"),
		forge.WithTags("MFA", "Authentication"),
		forge.WithRequestSchema(&InitiateChallengeRequest{}),
		forge.WithResponseSchema(200, "Challenge initiated successfully", &ChallengeResponse{}),
		forge.WithResponseSchema(400, "Invalid request or no enrolled factors", &ErrorResponse{}),
		forge.WithResponseSchema(401, "Unauthorized - authentication required", &ErrorResponse{}),
		forge.WithValidation(true),
	)

	// POST /mfa/verify - Verify an MFA challenge
	router.POST("/mfa/verify", handler.VerifyChallenge,
		forge.WithSummary("Verify MFA challenge"),
		forge.WithDescription("Verifies an MFA challenge by providing the required authentication code or data"),
		forge.WithTags("MFA", "Authentication"),
		forge.WithRequestSchema(&VerifyChallengeRequest{}),
		forge.WithResponseSchema(200, "Challenge verified successfully", &VerificationResponse{}),
		forge.WithResponseSchema(400, "Invalid code or verification failed", &ErrorResponse{}),
		forge.WithResponseSchema(401, "Unauthorized - authentication required", &ErrorResponse{}),
		forge.WithResponseSchema(429, "Too many failed attempts - account locked", &ErrorResponse{}),
		forge.WithValidation(true),
	)

	// GET /mfa/challenge/:id - Get challenge status
	router.GET("/mfa/challenge/:id", handler.GetChallengeStatus,
		forge.WithSummary("Get challenge status"),
		forge.WithDescription("Retrieves the current status and details of an MFA challenge"),
		forge.WithTags("MFA", "Authentication"),
		forge.WithRequestSchema(&GetChallengeStatusRequest{}),
		forge.WithResponseSchema(200, "Challenge status retrieved", &GetChallengeStatusResponse{}),
		forge.WithResponseSchema(400, "Invalid challenge ID", &ErrorResponse{}),
		forge.WithResponseSchema(401, "Unauthorized - authentication required", &ErrorResponse{}),
		forge.WithResponseSchema(404, "Challenge not found or expired", &ErrorResponse{}),
		forge.WithResponseSchema(501, "Not implemented", &ErrorResponse{}),
		forge.WithValidation(true),
	)

	// ==================== Trusted Devices ====================

	// POST /mfa/devices/trust - Trust current device
	router.POST("/mfa/devices/trust", handler.TrustDevice,
		forge.WithSummary("Trust device"),
		forge.WithDescription("Marks the current device as trusted to skip MFA for future authentications within the trust period"),
		forge.WithTags("MFA", "Trusted Devices"),
		forge.WithRequestSchema(&TrustDeviceRequest{}),
		forge.WithResponseSchema(200, "Device trusted successfully", &SuccessResponse{}),
		forge.WithResponseSchema(400, "Invalid request or trust failed", &ErrorResponse{}),
		forge.WithResponseSchema(401, "Unauthorized - authentication required", &ErrorResponse{}),
		forge.WithValidation(true),
	)

	// GET /mfa/devices - List trusted devices
	router.GET("/mfa/devices", handler.ListTrustedDevices,
		forge.WithSummary("List trusted devices"),
		forge.WithDescription("Retrieves all devices currently trusted by the authenticated user"),
		forge.WithTags("MFA", "Trusted Devices"),
		forge.WithResponseSchema(200, "Trusted devices retrieved", &ListTrustedDevicesResponse{}),
		forge.WithResponseSchema(401, "Unauthorized - authentication required", &ErrorResponse{}),
		forge.WithResponseSchema(500, "Internal server error", &ErrorResponse{}),
		forge.WithValidation(true),
	)

	// DELETE /mfa/devices/:id - Revoke trusted device
	router.DELETE("/mfa/devices/:id", handler.RevokeTrustedDevice,
		forge.WithSummary("Revoke trusted device"),
		forge.WithDescription("Removes trust status from a device, requiring MFA for future authentications"),
		forge.WithTags("MFA", "Trusted Devices"),
		forge.WithRequestSchema(&RevokeTrustedDeviceRequest{}),
		forge.WithResponseSchema(200, "Device revoked successfully", &SuccessResponse{}),
		forge.WithResponseSchema(400, "Invalid device ID or revocation failed", &ErrorResponse{}),
		forge.WithResponseSchema(401, "Unauthorized - authentication required", &ErrorResponse{}),
		forge.WithResponseSchema(403, "Forbidden - device belongs to another user", &ErrorResponse{}),
		forge.WithResponseSchema(404, "Device not found", &ErrorResponse{}),
		forge.WithValidation(true),
	)

	// ==================== Status & Info ====================

	// GET /mfa/status - Get MFA status
	router.GET("/mfa/status", handler.GetStatus,
		forge.WithSummary("Get MFA status"),
		forge.WithDescription("Retrieves the current MFA enrollment and policy status for the authenticated user"),
		forge.WithTags("MFA", "Status"),
		forge.WithRequestSchema(&GetStatusRequest{}),
		forge.WithResponseSchema(200, "MFA status retrieved", &MFAStatus{}),
		forge.WithResponseSchema(401, "Unauthorized - authentication required", &ErrorResponse{}),
		forge.WithResponseSchema(500, "Internal server error", &ErrorResponse{}),
		forge.WithValidation(true),
	)

	// GET /mfa/policy - Get MFA policy
	router.GET("/mfa/policy", handler.GetPolicy,
		forge.WithSummary("Get MFA policy"),
		forge.WithDescription("Retrieves the organization's MFA policy configuration"),
		forge.WithTags("MFA", "Policy"),
		forge.WithResponseSchema(200, "MFA policy retrieved", &MFAPolicy{}),
		forge.WithResponseSchema(401, "Unauthorized - authentication required", &ErrorResponse{}),
		forge.WithValidation(true),
	)

	// ==================== Admin Endpoints ====================

	// PUT /mfa/policy - Update MFA policy (admin only)
	router.PUT("/mfa/policy", handler.UpdatePolicy,
		forge.WithSummary("Update MFA policy"),
		forge.WithDescription("Updates the organization's MFA policy configuration (requires admin privileges)"),
		forge.WithTags("MFA", "Policy", "Admin"),
		forge.WithRequestSchema(&UpdatePolicyRequest{}),
		forge.WithResponseSchema(200, "Policy updated successfully", &MFAPolicy{}),
		forge.WithResponseSchema(400, "Invalid request", &ErrorResponse{}),
		forge.WithResponseSchema(401, "Unauthorized - authentication required", &ErrorResponse{}),
		forge.WithResponseSchema(403, "Forbidden - admin privileges required", &ErrorResponse{}),
		forge.WithResponseSchema(501, "Not implemented", &ErrorResponse{}),
		forge.WithValidation(true),
	)

	// POST /mfa/users/:id/reset - Reset user's MFA (admin only)
	router.POST("/mfa/users/:id/reset", handler.ResetUserMFA,
		forge.WithSummary("Reset user MFA"),
		forge.WithDescription("Resets all MFA factors and trusted devices for a user (requires admin privileges)"),
		forge.WithTags("MFA", "Admin"),
		forge.WithRequestSchema(&ResetUserMFARequest{}),
		forge.WithResponseSchema(200, "User MFA reset successfully", &ResetUserMFAResponse{}),
		forge.WithResponseSchema(400, "Invalid user ID", &ErrorResponse{}),
		forge.WithResponseSchema(401, "Unauthorized - authentication required", &ErrorResponse{}),
		forge.WithResponseSchema(403, "Forbidden - admin privileges required", &ErrorResponse{}),
		forge.WithResponseSchema(404, "User not found", &ErrorResponse{}),
		forge.WithResponseSchema(501, "Not implemented", &ErrorResponse{}),
		forge.WithValidation(true),
	)
}
