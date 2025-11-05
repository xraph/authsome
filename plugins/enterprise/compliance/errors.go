package compliance

import "errors"

var (
	// Profile errors
	ErrProfileNotFound  = errors.New("compliance profile not found")
	ErrProfileExists    = errors.New("compliance profile already exists for organization")
	ErrInvalidProfile   = errors.New("invalid compliance profile")
	ErrTemplateNotFound = errors.New("compliance template not found")

	// Check errors
	ErrCheckNotFound    = errors.New("compliance check not found")
	ErrCheckFailed      = errors.New("compliance check failed")
	ErrInvalidCheckType = errors.New("invalid check type")

	// Violation errors
	ErrViolationNotFound = errors.New("compliance violation not found")
	ErrViolationExists   = errors.New("violation already exists")
	ErrCannotResolve     = errors.New("cannot resolve violation")

	// Report errors
	ErrReportNotFound    = errors.New("compliance report not found")
	ErrReportGenerating  = errors.New("report is still generating")
	ErrReportFailed      = errors.New("report generation failed")
	ErrInvalidReportType = errors.New("invalid report type")
	ErrInvalidFormat     = errors.New("invalid report format")

	// Evidence errors
	ErrEvidenceNotFound = errors.New("compliance evidence not found")
	ErrInvalidEvidence  = errors.New("invalid evidence")
	ErrEvidenceExpired  = errors.New("evidence has expired")

	// Policy errors
	ErrPolicyNotFound    = errors.New("compliance policy not found")
	ErrPolicyExists      = errors.New("policy already exists")
	ErrInvalidPolicy     = errors.New("invalid policy")
	ErrPolicyNotApproved = errors.New("policy not approved")

	// Training errors
	ErrTrainingNotFound   = errors.New("compliance training not found")
	ErrTrainingIncomplete = errors.New("required training not completed")
	ErrTrainingExpired    = errors.New("training has expired")

	// Policy enforcement errors
	ErrMFARequired      = errors.New("MFA is required for this organization")
	ErrWeakPassword     = errors.New("password does not meet compliance requirements")
	ErrSessionExpired   = errors.New("session expired due to compliance policy")
	ErrAccessDenied     = errors.New("access denied due to compliance policy")
	ErrTrainingRequired = errors.New("compliance training required")

	// General errors
	ErrNotAuthorized = errors.New("not authorized to perform this action")
	ErrInvalidInput  = errors.New("invalid input")
	ErrInternalError = errors.New("internal compliance error")
)
