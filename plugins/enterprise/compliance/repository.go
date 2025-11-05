package compliance

import (
	"context"
	"time"
)

// Repository defines the data access interface for compliance
type Repository interface {
	// Compliance Profiles
	CreateProfile(ctx context.Context, profile *ComplianceProfile) error
	GetProfile(ctx context.Context, id string) (*ComplianceProfile, error)
	GetProfileByOrganization(ctx context.Context, orgID string) (*ComplianceProfile, error)
	UpdateProfile(ctx context.Context, profile *ComplianceProfile) error
	DeleteProfile(ctx context.Context, id string) error
	ListProfiles(ctx context.Context, filters ProfileFilters) ([]*ComplianceProfile, error)

	// Compliance Checks
	CreateCheck(ctx context.Context, check *ComplianceCheck) error
	GetCheck(ctx context.Context, id string) (*ComplianceCheck, error)
	ListChecks(ctx context.Context, profileID string, filters CheckFilters) ([]*ComplianceCheck, error)
	UpdateCheck(ctx context.Context, check *ComplianceCheck) error
	GetDueChecks(ctx context.Context) ([]*ComplianceCheck, error)

	// Violations
	CreateViolation(ctx context.Context, violation *ComplianceViolation) error
	GetViolation(ctx context.Context, id string) (*ComplianceViolation, error)
	ListViolations(ctx context.Context, filters ViolationFilters) ([]*ComplianceViolation, error)
	UpdateViolation(ctx context.Context, violation *ComplianceViolation) error
	ResolveViolation(ctx context.Context, id, resolvedBy string) error
	CountViolations(ctx context.Context, orgID string, status string) (int, error)

	// Reports
	CreateReport(ctx context.Context, report *ComplianceReport) error
	GetReport(ctx context.Context, id string) (*ComplianceReport, error)
	ListReports(ctx context.Context, filters ReportFilters) ([]*ComplianceReport, error)
	UpdateReport(ctx context.Context, report *ComplianceReport) error
	DeleteReport(ctx context.Context, id string) error

	// Evidence
	CreateEvidence(ctx context.Context, evidence *ComplianceEvidence) error
	GetEvidence(ctx context.Context, id string) (*ComplianceEvidence, error)
	ListEvidence(ctx context.Context, filters EvidenceFilters) ([]*ComplianceEvidence, error)
	DeleteEvidence(ctx context.Context, id string) error

	// Policies
	CreatePolicy(ctx context.Context, policy *CompliancePolicy) error
	GetPolicy(ctx context.Context, id string) (*CompliancePolicy, error)
	GetActivePolicies(ctx context.Context, orgID string) ([]*CompliancePolicy, error)
	ListPolicies(ctx context.Context, filters PolicyFilters) ([]*CompliancePolicy, error)
	UpdatePolicy(ctx context.Context, policy *CompliancePolicy) error
	DeletePolicy(ctx context.Context, id string) error

	// Training
	CreateTraining(ctx context.Context, training *ComplianceTraining) error
	GetTraining(ctx context.Context, id string) (*ComplianceTraining, error)
	ListTraining(ctx context.Context, filters TrainingFilters) ([]*ComplianceTraining, error)
	UpdateTraining(ctx context.Context, training *ComplianceTraining) error
	GetUserTrainingStatus(ctx context.Context, userID string) ([]*ComplianceTraining, error)
	GetOverdueTraining(ctx context.Context, orgID string) ([]*ComplianceTraining, error)
}

// ProfileFilters for querying profiles
type ProfileFilters struct {
	OrganizationID string
	Status         string
	Standard       ComplianceStandard
	Limit          int
	Offset         int
}

// CheckFilters for querying checks
type CheckFilters struct {
	ProfileID   string
	CheckType   string
	Status      string
	SinceBefore time.Time
	Limit       int
	Offset      int
}

// ViolationFilters for querying violations
type ViolationFilters struct {
	OrganizationID string
	ProfileID      string
	UserID         string
	ViolationType  string
	Severity       string
	Status         string
	Since          time.Time
	Limit          int
	Offset         int
}

// ReportFilters for querying reports
type ReportFilters struct {
	OrganizationID string
	ProfileID      string
	ReportType     string
	Standard       ComplianceStandard
	Status         string
	Period         string
	Limit          int
	Offset         int
}

// EvidenceFilters for querying evidence
type EvidenceFilters struct {
	OrganizationID string
	ProfileID      string
	EvidenceType   string
	Standard       ComplianceStandard
	ControlID      string
	Limit          int
	Offset         int
}

// PolicyFilters for querying policies
type PolicyFilters struct {
	OrganizationID string
	ProfileID      string
	PolicyType     string
	Standard       ComplianceStandard
	Status         string
	Limit          int
	Offset         int
}

// TrainingFilters for querying training
type TrainingFilters struct {
	OrganizationID string
	ProfileID      string
	UserID         string
	TrainingType   string
	Standard       ComplianceStandard
	Status         string
	Limit          int
	Offset         int
}
