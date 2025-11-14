package compliance

import (
	"net/http"

	"github.com/xraph/authsome/internal/errs"
)

// Error codes for compliance operations
const (
	// Profile errors
	CodeProfileNotFound  = "COMPLIANCE_PROFILE_NOT_FOUND"
	CodeProfileExists    = "COMPLIANCE_PROFILE_EXISTS"
	CodeInvalidProfile   = "COMPLIANCE_INVALID_PROFILE"
	CodeTemplateNotFound = "COMPLIANCE_TEMPLATE_NOT_FOUND"

	// Check errors
	CodeCheckNotFound    = "COMPLIANCE_CHECK_NOT_FOUND"
	CodeCheckFailed      = "COMPLIANCE_CHECK_FAILED"
	CodeInvalidCheckType = "COMPLIANCE_INVALID_CHECK_TYPE"

	// Violation errors
	CodeViolationNotFound = "COMPLIANCE_VIOLATION_NOT_FOUND"
	CodeViolationExists   = "COMPLIANCE_VIOLATION_EXISTS"
	CodeCannotResolve     = "COMPLIANCE_CANNOT_RESOLVE"

	// Report errors
	CodeReportNotFound    = "COMPLIANCE_REPORT_NOT_FOUND"
	CodeReportGenerating  = "COMPLIANCE_REPORT_GENERATING"
	CodeReportFailed      = "COMPLIANCE_REPORT_FAILED"
	CodeInvalidReportType = "COMPLIANCE_INVALID_REPORT_TYPE"
	CodeInvalidFormat     = "COMPLIANCE_INVALID_FORMAT"

	// Evidence errors
	CodeEvidenceNotFound = "COMPLIANCE_EVIDENCE_NOT_FOUND"
	CodeInvalidEvidence  = "COMPLIANCE_INVALID_EVIDENCE"
	CodeEvidenceExpired  = "COMPLIANCE_EVIDENCE_EXPIRED"

	// Policy errors
	CodePolicyNotFound    = "COMPLIANCE_POLICY_NOT_FOUND"
	CodePolicyExists      = "COMPLIANCE_POLICY_EXISTS"
	CodeInvalidPolicy     = "COMPLIANCE_INVALID_POLICY"
	CodePolicyNotApproved = "COMPLIANCE_POLICY_NOT_APPROVED"

	// Training errors
	CodeTrainingNotFound   = "COMPLIANCE_TRAINING_NOT_FOUND"
	CodeTrainingIncomplete = "COMPLIANCE_TRAINING_INCOMPLETE"
	CodeTrainingExpired    = "COMPLIANCE_TRAINING_EXPIRED"

	// Policy enforcement errors
	CodeMFARequired      = "COMPLIANCE_MFA_REQUIRED"
	CodeWeakPassword     = "COMPLIANCE_WEAK_PASSWORD"
	CodeSessionExpired   = "COMPLIANCE_SESSION_EXPIRED"
	CodeAccessDenied     = "COMPLIANCE_ACCESS_DENIED"
	CodeTrainingRequired = "COMPLIANCE_TRAINING_REQUIRED"

	// General errors
	CodeNotAuthorized     = "COMPLIANCE_NOT_AUTHORIZED"
	CodeInvalidInput      = "COMPLIANCE_INVALID_INPUT"
	CodeInternalError     = "COMPLIANCE_INTERNAL_ERROR"
	CodeInvalidPagination = "COMPLIANCE_INVALID_PAGINATION"
	CodeQueryFailed       = "COMPLIANCE_QUERY_FAILED"
	CodeOperationFailed   = "COMPLIANCE_OPERATION_FAILED"
)

// Profile error constructors

func ProfileNotFound(id string) *errs.AuthsomeError {
	return errs.New(CodeProfileNotFound, "Compliance profile not found", http.StatusNotFound).
		WithContext("profile_id", id)
}

func ProfileExists(appID string) *errs.AuthsomeError {
	return errs.New(CodeProfileExists, "Compliance profile already exists", http.StatusConflict).
		WithContext("app_id", appID)
}

func InvalidProfile(reason string) *errs.AuthsomeError {
	return errs.New(CodeInvalidProfile, "Invalid compliance profile", http.StatusBadRequest).
		WithContext("reason", reason)
}

func TemplateNotFound(standard string) *errs.AuthsomeError {
	return errs.New(CodeTemplateNotFound, "Compliance template not found", http.StatusNotFound).
		WithContext("standard", standard)
}

// Check error constructors

func CheckNotFound(id string) *errs.AuthsomeError {
	return errs.New(CodeCheckNotFound, "Compliance check not found", http.StatusNotFound).
		WithContext("check_id", id)
}

func CheckFailed(checkType string, reason string) *errs.AuthsomeError {
	return errs.New(CodeCheckFailed, "Compliance check failed", http.StatusUnprocessableEntity).
		WithContext("check_type", checkType).
		WithContext("reason", reason)
}

func InvalidCheckType(checkType string) *errs.AuthsomeError {
	return errs.New(CodeInvalidCheckType, "Invalid check type", http.StatusBadRequest).
		WithContext("check_type", checkType)
}

// Violation error constructors

func ViolationNotFound(id string) *errs.AuthsomeError {
	return errs.New(CodeViolationNotFound, "Compliance violation not found", http.StatusNotFound).
		WithContext("violation_id", id)
}

func ViolationExists(violationType string, userID string) *errs.AuthsomeError {
	return errs.New(CodeViolationExists, "Violation already exists", http.StatusConflict).
		WithContext("violation_type", violationType).
		WithContext("user_id", userID)
}

func CannotResolve(id string, reason string) *errs.AuthsomeError {
	return errs.New(CodeCannotResolve, "Cannot resolve violation", http.StatusBadRequest).
		WithContext("violation_id", id).
		WithContext("reason", reason)
}

// Report error constructors

func ReportNotFound(id string) *errs.AuthsomeError {
	return errs.New(CodeReportNotFound, "Compliance report not found", http.StatusNotFound).
		WithContext("report_id", id)
}

func ReportGenerating(id string) *errs.AuthsomeError {
	return errs.New(CodeReportGenerating, "Report is still generating", http.StatusAccepted).
		WithContext("report_id", id)
}

func ReportFailed(id string, reason string) *errs.AuthsomeError {
	return errs.New(CodeReportFailed, "Report generation failed", http.StatusInternalServerError).
		WithContext("report_id", id).
		WithContext("reason", reason)
}

func InvalidReportType(reportType string) *errs.AuthsomeError {
	return errs.New(CodeInvalidReportType, "Invalid report type", http.StatusBadRequest).
		WithContext("report_type", reportType)
}

func InvalidFormat(format string) *errs.AuthsomeError {
	return errs.New(CodeInvalidFormat, "Invalid report format", http.StatusBadRequest).
		WithContext("format", format)
}

// Evidence error constructors

func EvidenceNotFound(id string) *errs.AuthsomeError {
	return errs.New(CodeEvidenceNotFound, "Compliance evidence not found", http.StatusNotFound).
		WithContext("evidence_id", id)
}

func InvalidEvidence(reason string) *errs.AuthsomeError {
	return errs.New(CodeInvalidEvidence, "Invalid evidence", http.StatusBadRequest).
		WithContext("reason", reason)
}

func EvidenceExpired(id string) *errs.AuthsomeError {
	return errs.New(CodeEvidenceExpired, "Evidence has expired", http.StatusGone).
		WithContext("evidence_id", id)
}

// Policy error constructors

func PolicyNotFound(id string) *errs.AuthsomeError {
	return errs.New(CodePolicyNotFound, "Compliance policy not found", http.StatusNotFound).
		WithContext("policy_id", id)
}

func PolicyExists(policyType string) *errs.AuthsomeError {
	return errs.New(CodePolicyExists, "Policy already exists", http.StatusConflict).
		WithContext("policy_type", policyType)
}

func InvalidPolicy(reason string) *errs.AuthsomeError {
	return errs.New(CodeInvalidPolicy, "Invalid policy", http.StatusBadRequest).
		WithContext("reason", reason)
}

func PolicyNotApproved(id string) *errs.AuthsomeError {
	return errs.New(CodePolicyNotApproved, "Policy not approved", http.StatusForbidden).
		WithContext("policy_id", id)
}

// Training error constructors

func TrainingNotFound(id string) *errs.AuthsomeError {
	return errs.New(CodeTrainingNotFound, "Compliance training not found", http.StatusNotFound).
		WithContext("training_id", id)
}

func TrainingIncomplete(userID string, trainingType string) *errs.AuthsomeError {
	return errs.New(CodeTrainingIncomplete, "Required training not completed", http.StatusForbidden).
		WithContext("user_id", userID).
		WithContext("training_type", trainingType)
}

func TrainingExpired(userID string, trainingType string) *errs.AuthsomeError {
	return errs.New(CodeTrainingExpired, "Training has expired", http.StatusForbidden).
		WithContext("user_id", userID).
		WithContext("training_type", trainingType)
}

// Policy enforcement error constructors

func MFARequired() *errs.AuthsomeError {
	return errs.New(CodeMFARequired, "MFA is required for this app", http.StatusForbidden)
}

func WeakPassword(reason string) *errs.AuthsomeError {
	return errs.New(CodeWeakPassword, "Password does not meet compliance requirements", http.StatusBadRequest).
		WithContext("reason", reason)
}

func SessionExpired(reason string) *errs.AuthsomeError {
	return errs.New(CodeSessionExpired, "Session expired due to compliance policy", http.StatusUnauthorized).
		WithContext("reason", reason)
}

func AccessDenied(reason string) *errs.AuthsomeError {
	return errs.New(CodeAccessDenied, "Access denied due to compliance policy", http.StatusForbidden).
		WithContext("reason", reason)
}

func TrainingRequired(trainingType string) *errs.AuthsomeError {
	return errs.New(CodeTrainingRequired, "Compliance training required", http.StatusForbidden).
		WithContext("training_type", trainingType)
}

// General error constructors

func NotAuthorized() *errs.AuthsomeError {
	return errs.New(CodeNotAuthorized, "Not authorized to perform this action", http.StatusForbidden)
}

func InvalidInput(field string, reason string) *errs.AuthsomeError {
	return errs.New(CodeInvalidInput, "Invalid input", http.StatusBadRequest).
		WithContext("field", field).
		WithContext("reason", reason)
}

func InternalError(operation string, err error) *errs.AuthsomeError {
	return errs.New(CodeInternalError, "Internal compliance error", http.StatusInternalServerError).
		WithContext("operation", operation).
		WithError(err)
}

func InvalidPagination(reason string) *errs.AuthsomeError {
	return errs.New(CodeInvalidPagination, "Invalid pagination parameters", http.StatusBadRequest).
		WithContext("reason", reason)
}

func QueryFailed(operation string, err error) *errs.AuthsomeError {
	return errs.New(CodeQueryFailed, "Query operation failed", http.StatusInternalServerError).
		WithContext("operation", operation).
		WithError(err)
}

func OperationFailed(operation string, reason string) *errs.AuthsomeError {
	return errs.New(CodeOperationFailed, "Operation failed", http.StatusInternalServerError).
		WithContext("operation", operation).
		WithContext("reason", reason)
}
