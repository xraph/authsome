package compliance

import (
	"time"

	"github.com/xraph/authsome/core/pagination"
)

// ListProfilesFilter defines filters for listing compliance profiles with pagination.
type ListProfilesFilter struct {
	pagination.PaginationParams

	AppID    *string             `json:"appId,omitempty"    query:"app_id"`
	Status   *string             `json:"status,omitempty"   query:"status"`
	Standard *ComplianceStandard `json:"standard,omitempty" query:"standard"`
}

// ListChecksFilter defines filters for listing compliance checks with pagination.
type ListChecksFilter struct {
	pagination.PaginationParams

	ProfileID   *string    `json:"profileId,omitempty"   query:"profile_id"`
	AppID       *string    `json:"appId,omitempty"       query:"app_id"`
	CheckType   *string    `json:"checkType,omitempty"   query:"check_type"`
	Status      *string    `json:"status,omitempty"      query:"status"`
	SinceBefore *time.Time `json:"sinceBefore,omitempty" query:"since_before"`
}

// ListViolationsFilter defines filters for listing compliance violations with pagination.
type ListViolationsFilter struct {
	pagination.PaginationParams

	AppID         *string `json:"appId,omitempty"         query:"app_id"`
	ProfileID     *string `json:"profileId,omitempty"     query:"profile_id"`
	UserID        *string `json:"userId,omitempty"        query:"user_id"`
	ViolationType *string `json:"violationType,omitempty" query:"violation_type"`
	Severity      *string `json:"severity,omitempty"      query:"severity"`
	Status        *string `json:"status,omitempty"        query:"status"`
}

// ListReportsFilter defines filters for listing compliance reports with pagination.
type ListReportsFilter struct {
	pagination.PaginationParams

	AppID      *string             `json:"appId,omitempty"      query:"app_id"`
	ProfileID  *string             `json:"profileId,omitempty"  query:"profile_id"`
	ReportType *string             `json:"reportType,omitempty" query:"report_type"`
	Standard   *ComplianceStandard `json:"standard,omitempty"   query:"standard"`
	Status     *string             `json:"status,omitempty"     query:"status"`
	Format     *string             `json:"format,omitempty"     query:"format"`
}

// ListEvidenceFilter defines filters for listing compliance evidence with pagination.
type ListEvidenceFilter struct {
	pagination.PaginationParams

	AppID        *string             `json:"appId,omitempty"        query:"app_id"`
	ProfileID    *string             `json:"profileId,omitempty"    query:"profile_id"`
	EvidenceType *string             `json:"evidenceType,omitempty" query:"evidence_type"`
	Standard     *ComplianceStandard `json:"standard,omitempty"     query:"standard"`
	ControlID    *string             `json:"controlId,omitempty"    query:"control_id"`
}

// ListPoliciesFilter defines filters for listing compliance policies with pagination.
type ListPoliciesFilter struct {
	pagination.PaginationParams

	AppID      *string             `json:"appId,omitempty"      query:"app_id"`
	ProfileID  *string             `json:"profileId,omitempty"  query:"profile_id"`
	PolicyType *string             `json:"policyType,omitempty" query:"policy_type"`
	Standard   *ComplianceStandard `json:"standard,omitempty"   query:"standard"`
	Status     *string             `json:"status,omitempty"     query:"status"`
}

// ListTrainingFilter defines filters for listing compliance training with pagination.
type ListTrainingFilter struct {
	pagination.PaginationParams

	AppID        *string             `json:"appId,omitempty"        query:"app_id"`
	ProfileID    *string             `json:"profileId,omitempty"    query:"profile_id"`
	UserID       *string             `json:"userId,omitempty"       query:"user_id"`
	TrainingType *string             `json:"trainingType,omitempty" query:"training_type"`
	Standard     *ComplianceStandard `json:"standard,omitempty"     query:"standard"`
	Status       *string             `json:"status,omitempty"       query:"status"`
}
