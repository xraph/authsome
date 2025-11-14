package compliance

import (
	"context"

	"github.com/xraph/authsome/core/pagination"
)

// Repository defines the data access interface for compliance
type Repository interface {
	// Compliance Profiles
	CreateProfile(ctx context.Context, profile *ComplianceProfile) error
	GetProfile(ctx context.Context, id string) (*ComplianceProfile, error)
	GetProfileByApp(ctx context.Context, appID string) (*ComplianceProfile, error)
	UpdateProfile(ctx context.Context, profile *ComplianceProfile) error
	DeleteProfile(ctx context.Context, id string) error
	ListProfiles(ctx context.Context, filter *ListProfilesFilter) (*pagination.PageResponse[*ComplianceProfile], error)

	// Compliance Checks
	CreateCheck(ctx context.Context, check *ComplianceCheck) error
	GetCheck(ctx context.Context, id string) (*ComplianceCheck, error)
	ListChecks(ctx context.Context, filter *ListChecksFilter) (*pagination.PageResponse[*ComplianceCheck], error)
	UpdateCheck(ctx context.Context, check *ComplianceCheck) error
	GetDueChecks(ctx context.Context) ([]*ComplianceCheck, error)

	// Violations
	CreateViolation(ctx context.Context, violation *ComplianceViolation) error
	GetViolation(ctx context.Context, id string) (*ComplianceViolation, error)
	ListViolations(ctx context.Context, filter *ListViolationsFilter) (*pagination.PageResponse[*ComplianceViolation], error)
	UpdateViolation(ctx context.Context, violation *ComplianceViolation) error
	ResolveViolation(ctx context.Context, id, resolvedBy string) error
	CountViolations(ctx context.Context, appID string, status string) (int, error)

	// Reports
	CreateReport(ctx context.Context, report *ComplianceReport) error
	GetReport(ctx context.Context, id string) (*ComplianceReport, error)
	ListReports(ctx context.Context, filter *ListReportsFilter) (*pagination.PageResponse[*ComplianceReport], error)
	UpdateReport(ctx context.Context, report *ComplianceReport) error
	DeleteReport(ctx context.Context, id string) error

	// Evidence
	CreateEvidence(ctx context.Context, evidence *ComplianceEvidence) error
	GetEvidence(ctx context.Context, id string) (*ComplianceEvidence, error)
	ListEvidence(ctx context.Context, filter *ListEvidenceFilter) (*pagination.PageResponse[*ComplianceEvidence], error)
	DeleteEvidence(ctx context.Context, id string) error

	// Policies
	CreatePolicy(ctx context.Context, policy *CompliancePolicy) error
	GetPolicy(ctx context.Context, id string) (*CompliancePolicy, error)
	GetActivePolicies(ctx context.Context, appID string) ([]*CompliancePolicy, error)
	ListPolicies(ctx context.Context, filter *ListPoliciesFilter) (*pagination.PageResponse[*CompliancePolicy], error)
	UpdatePolicy(ctx context.Context, policy *CompliancePolicy) error
	DeletePolicy(ctx context.Context, id string) error

	// Training
	CreateTraining(ctx context.Context, training *ComplianceTraining) error
	GetTraining(ctx context.Context, id string) (*ComplianceTraining, error)
	ListTraining(ctx context.Context, filter *ListTrainingFilter) (*pagination.PageResponse[*ComplianceTraining], error)
	UpdateTraining(ctx context.Context, training *ComplianceTraining) error
	GetUserTrainingStatus(ctx context.Context, userID string) ([]*ComplianceTraining, error)
	GetOverdueTraining(ctx context.Context, appID string) ([]*ComplianceTraining, error)
}

// Note: Filter types have been moved to filters.go with pagination support
