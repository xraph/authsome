package compliance

import (
	"context"
	"database/sql"
	"errors"

	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core/pagination"
)

// RegisterModels registers compliance models with Bun.
func RegisterModels(db *bun.DB) {
	db.RegisterModel(
		(*ComplianceProfile)(nil),
		(*ComplianceCheck)(nil),
		(*ComplianceViolation)(nil),
		(*ComplianceReport)(nil),
		(*ComplianceEvidence)(nil),
		(*CompliancePolicy)(nil),
		(*ComplianceTraining)(nil),
	)
}

// CreateTables creates all compliance tables.
func CreateTables(ctx context.Context, db *bun.DB) error {
	models := []any{
		(*ComplianceProfile)(nil),
		(*ComplianceCheck)(nil),
		(*ComplianceViolation)(nil),
		(*ComplianceReport)(nil),
		(*ComplianceEvidence)(nil),
		(*CompliancePolicy)(nil),
		(*ComplianceTraining)(nil),
	}

	for _, model := range models {
		_, err := db.NewCreateTable().
			Model(model).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}
	}

	// Create indexes
	return createIndexes(ctx, db)
}

// createIndexes creates database indexes for optimal performance.
func createIndexes(ctx context.Context, db *bun.DB) error {
	indexes := []string{
		// Compliance Profiles
		`CREATE INDEX IF NOT EXISTS idx_compliance_profiles_app_id ON compliance_profiles(app_id)`,
		`CREATE INDEX IF NOT EXISTS idx_compliance_profiles_status ON compliance_profiles(status)`,

		// Compliance Checks
		`CREATE INDEX IF NOT EXISTS idx_compliance_checks_profile_id ON compliance_checks(profile_id)`,
		`CREATE INDEX IF NOT EXISTS idx_compliance_checks_app_id ON compliance_checks(app_id)`,
		`CREATE INDEX IF NOT EXISTS idx_compliance_checks_status ON compliance_checks(status)`,
		`CREATE INDEX IF NOT EXISTS idx_compliance_checks_check_type ON compliance_checks(check_type)`,
		`CREATE INDEX IF NOT EXISTS idx_compliance_checks_next_check_at ON compliance_checks(next_check_at)`,

		// Compliance Violations
		`CREATE INDEX IF NOT EXISTS idx_compliance_violations_profile_id ON compliance_violations(profile_id)`,
		`CREATE INDEX IF NOT EXISTS idx_compliance_violations_app_id ON compliance_violations(app_id)`,
		`CREATE INDEX IF NOT EXISTS idx_compliance_violations_user_id ON compliance_violations(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_compliance_violations_status ON compliance_violations(status)`,
		`CREATE INDEX IF NOT EXISTS idx_compliance_violations_severity ON compliance_violations(severity)`,
		`CREATE INDEX IF NOT EXISTS idx_compliance_violations_violation_type ON compliance_violations(violation_type)`,

		// Compliance Reports
		`CREATE INDEX IF NOT EXISTS idx_compliance_reports_profile_id ON compliance_reports(profile_id)`,
		`CREATE INDEX IF NOT EXISTS idx_compliance_reports_app_id ON compliance_reports(app_id)`,
		`CREATE INDEX IF NOT EXISTS idx_compliance_reports_status ON compliance_reports(status)`,
		`CREATE INDEX IF NOT EXISTS idx_compliance_reports_report_type ON compliance_reports(report_type)`,
		`CREATE INDEX IF NOT EXISTS idx_compliance_reports_period ON compliance_reports(period)`,

		// Compliance Evidence
		`CREATE INDEX IF NOT EXISTS idx_compliance_evidence_profile_id ON compliance_evidence(profile_id)`,
		`CREATE INDEX IF NOT EXISTS idx_compliance_evidence_app_id ON compliance_evidence(app_id)`,
		`CREATE INDEX IF NOT EXISTS idx_compliance_evidence_evidence_type ON compliance_evidence(evidence_type)`,
		`CREATE INDEX IF NOT EXISTS idx_compliance_evidence_standard ON compliance_evidence(standard)`,
		`CREATE INDEX IF NOT EXISTS idx_compliance_evidence_control_id ON compliance_evidence(control_id)`,

		// Compliance Policies
		`CREATE INDEX IF NOT EXISTS idx_compliance_policies_profile_id ON compliance_policies(profile_id)`,
		`CREATE INDEX IF NOT EXISTS idx_compliance_policies_app_id ON compliance_policies(app_id)`,
		`CREATE INDEX IF NOT EXISTS idx_compliance_policies_policy_type ON compliance_policies(policy_type)`,
		`CREATE INDEX IF NOT EXISTS idx_compliance_policies_status ON compliance_policies(status)`,
		`CREATE INDEX IF NOT EXISTS idx_compliance_policies_standard ON compliance_policies(standard)`,

		// Compliance Training
		`CREATE INDEX IF NOT EXISTS idx_compliance_training_profile_id ON compliance_training(profile_id)`,
		`CREATE INDEX IF NOT EXISTS idx_compliance_training_app_id ON compliance_training(app_id)`,
		`CREATE INDEX IF NOT EXISTS idx_compliance_training_user_id ON compliance_training(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_compliance_training_status ON compliance_training(status)`,
		`CREATE INDEX IF NOT EXISTS idx_compliance_training_training_type ON compliance_training(training_type)`,
		`CREATE INDEX IF NOT EXISTS idx_compliance_training_expires_at ON compliance_training(expires_at)`,
	}

	for _, query := range indexes {
		_, err := db.ExecContext(ctx, query)
		if err != nil {
			return err
		}
	}

	return nil
}

// DropTables drops all compliance tables (for testing).
func DropTables(ctx context.Context, db *bun.DB) error {
	tables := []string{
		"compliance_training",
		"compliance_policies",
		"compliance_evidence",
		"compliance_reports",
		"compliance_violations",
		"compliance_checks",
		"compliance_profiles",
	}

	for _, table := range tables {
		_, err := db.NewDropTable().
			Table(table).
			IfExists().
			Cascade().
			Exec(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

// BunRepository implements the Repository interface using Bun ORM.
type BunRepository struct {
	db *bun.DB
}

// NewBunRepository creates a new Bun repository.
func NewBunRepository(db any) Repository {
	return &BunRepository{
		db: db.(*bun.DB),
	}
}

// Implement Repository interface methods
// Note: These are stubs - full implementation would follow AuthSome patterns

func (r *BunRepository) CreateProfile(ctx context.Context, profile *ComplianceProfile) error {
	_, err := r.db.NewInsert().Model(profile).Exec(ctx)

	return err
}

func (r *BunRepository) GetProfile(ctx context.Context, id string) (*ComplianceProfile, error) {
	profile := new(ComplianceProfile)

	err := r.db.NewSelect().Model(profile).Where("id = ?", id).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	return profile, err
}

func (r *BunRepository) GetProfileByApp(ctx context.Context, appID string) (*ComplianceProfile, error) {
	profile := new(ComplianceProfile)

	err := r.db.NewSelect().Model(profile).Where("app_id = ?", appID).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	return profile, err
}

func (r *BunRepository) UpdateProfile(ctx context.Context, profile *ComplianceProfile) error {
	_, err := r.db.NewUpdate().Model(profile).WherePK().Exec(ctx)

	return err
}

func (r *BunRepository) DeleteProfile(ctx context.Context, id string) error {
	_, err := r.db.NewDelete().Model((*ComplianceProfile)(nil)).Where("id = ?", id).Exec(ctx)

	return err
}

func (r *BunRepository) ListProfiles(ctx context.Context, filter *ListProfilesFilter) (*pagination.PageResponse[*ComplianceProfile], error) {
	baseQuery := r.db.NewSelect().Model((*ComplianceProfile)(nil))

	if filter.AppID != nil {
		baseQuery = baseQuery.Where("app_id = ?", *filter.AppID)
	}

	if filter.Status != nil {
		baseQuery = baseQuery.Where("status = ?", *filter.Status)
	}

	if filter.Standard != nil {
		baseQuery = baseQuery.Where("? = ANY(standards)", *filter.Standard)
	}

	total, err := baseQuery.Count(ctx)
	if err != nil {
		return nil, err
	}

	baseQuery = baseQuery.Limit(filter.Limit).Offset(filter.Offset)

	var profiles []*ComplianceProfile

	if err := baseQuery.Scan(ctx); err != nil {
		return nil, err
	}

	params := &pagination.PaginationParams{
		Limit:  filter.Limit,
		Offset: filter.Offset,
	}

	return pagination.NewPageResponse(profiles, int64(total), params), nil
}

func (r *BunRepository) CreateCheck(ctx context.Context, check *ComplianceCheck) error {
	_, err := r.db.NewInsert().Model(check).Exec(ctx)

	return err
}

func (r *BunRepository) GetCheck(ctx context.Context, id string) (*ComplianceCheck, error) {
	check := new(ComplianceCheck)

	err := r.db.NewSelect().Model(check).Where("id = ?", id).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	return check, err
}

func (r *BunRepository) ListChecks(ctx context.Context, filter *ListChecksFilter) (*pagination.PageResponse[*ComplianceCheck], error) {
	baseQuery := r.db.NewSelect().Model((*ComplianceCheck)(nil))

	if filter.ProfileID != nil {
		baseQuery = baseQuery.Where("profile_id = ?", *filter.ProfileID)
	}

	if filter.AppID != nil {
		baseQuery = baseQuery.Where("app_id = ?", *filter.AppID)
	}

	if filter.CheckType != nil {
		baseQuery = baseQuery.Where("check_type = ?", *filter.CheckType)
	}

	if filter.Status != nil {
		baseQuery = baseQuery.Where("status = ?", *filter.Status)
	}

	if filter.SinceBefore != nil {
		baseQuery = baseQuery.Where("last_checked_at >= ?", *filter.SinceBefore)
	}

	total, err := baseQuery.Count(ctx)
	if err != nil {
		return nil, err
	}

	baseQuery = baseQuery.Order("last_checked_at DESC").Limit(filter.Limit).Offset(filter.Offset)

	var checks []*ComplianceCheck

	if err := baseQuery.Scan(ctx); err != nil {
		return nil, err
	}

	params := &pagination.PaginationParams{
		Limit:  filter.Limit,
		Offset: filter.Offset,
	}

	return pagination.NewPageResponse(checks, int64(total), params), nil
}

func (r *BunRepository) UpdateCheck(ctx context.Context, check *ComplianceCheck) error {
	_, err := r.db.NewUpdate().Model(check).WherePK().Exec(ctx)

	return err
}

func (r *BunRepository) GetDueChecks(ctx context.Context) ([]*ComplianceCheck, error) {
	var checks []*ComplianceCheck

	err := r.db.NewSelect().Model(&checks).
		Where("next_check_at <= NOW()").
		Scan(ctx)

	return checks, err
}

func (r *BunRepository) CreateViolation(ctx context.Context, violation *ComplianceViolation) error {
	_, err := r.db.NewInsert().Model(violation).Exec(ctx)

	return err
}

func (r *BunRepository) GetViolation(ctx context.Context, id string) (*ComplianceViolation, error) {
	violation := new(ComplianceViolation)

	err := r.db.NewSelect().Model(violation).Where("id = ?", id).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	return violation, err
}

func (r *BunRepository) ListViolations(ctx context.Context, filter *ListViolationsFilter) (*pagination.PageResponse[*ComplianceViolation], error) {
	baseQuery := r.db.NewSelect().Model((*ComplianceViolation)(nil))

	if filter.AppID != nil {
		baseQuery = baseQuery.Where("app_id = ?", *filter.AppID)
	}

	if filter.ProfileID != nil {
		baseQuery = baseQuery.Where("profile_id = ?", *filter.ProfileID)
	}

	if filter.UserID != nil {
		baseQuery = baseQuery.Where("user_id = ?", *filter.UserID)
	}

	if filter.Status != nil {
		baseQuery = baseQuery.Where("status = ?", *filter.Status)
	}

	if filter.Severity != nil {
		baseQuery = baseQuery.Where("severity = ?", *filter.Severity)
	}

	if filter.ViolationType != nil {
		baseQuery = baseQuery.Where("violation_type = ?", *filter.ViolationType)
	}

	total, err := baseQuery.Count(ctx)
	if err != nil {
		return nil, err
	}

	baseQuery = baseQuery.Order("created_at DESC").Limit(filter.Limit).Offset(filter.Offset)

	var violations []*ComplianceViolation

	if err := baseQuery.Scan(ctx); err != nil {
		return nil, err
	}

	params := &pagination.PaginationParams{
		Limit:  filter.Limit,
		Offset: filter.Offset,
	}

	return pagination.NewPageResponse(violations, int64(total), params), nil
}

func (r *BunRepository) UpdateViolation(ctx context.Context, violation *ComplianceViolation) error {
	_, err := r.db.NewUpdate().Model(violation).WherePK().Exec(ctx)

	return err
}

func (r *BunRepository) ResolveViolation(ctx context.Context, id, resolvedBy string) error {
	_, err := r.db.NewUpdate().
		Model((*ComplianceViolation)(nil)).
		Set("status = ?", "resolved").
		Set("resolved_by = ?", resolvedBy).
		Set("resolved_at = NOW()").
		Where("id = ?", id).
		Exec(ctx)

	return err
}

func (r *BunRepository) CountViolations(ctx context.Context, appID string, status string) (int, error) {
	count, err := r.db.NewSelect().
		Model((*ComplianceViolation)(nil)).
		Where("app_id = ?", appID).
		Where("status = ?", status).
		Count(ctx)

	return count, err
}

// Additional methods would follow the same pattern
// For brevity, including stubs for remaining methods

func (r *BunRepository) CreateReport(ctx context.Context, report *ComplianceReport) error {
	_, err := r.db.NewInsert().Model(report).Exec(ctx)

	return err
}

func (r *BunRepository) GetReport(ctx context.Context, id string) (*ComplianceReport, error) {
	report := new(ComplianceReport)

	err := r.db.NewSelect().Model(report).Where("id = ?", id).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	return report, err
}

func (r *BunRepository) ListReports(ctx context.Context, filter *ListReportsFilter) (*pagination.PageResponse[*ComplianceReport], error) {
	baseQuery := r.db.NewSelect().Model((*ComplianceReport)(nil))

	if filter.AppID != nil {
		baseQuery = baseQuery.Where("app_id = ?", *filter.AppID)
	}

	if filter.ProfileID != nil {
		baseQuery = baseQuery.Where("profile_id = ?", *filter.ProfileID)
	}

	if filter.ReportType != nil {
		baseQuery = baseQuery.Where("report_type = ?", *filter.ReportType)
	}

	if filter.Standard != nil {
		baseQuery = baseQuery.Where("standard = ?", *filter.Standard)
	}

	if filter.Status != nil {
		baseQuery = baseQuery.Where("status = ?", *filter.Status)
	}

	if filter.Format != nil {
		baseQuery = baseQuery.Where("format = ?", *filter.Format)
	}

	total, err := baseQuery.Count(ctx)
	if err != nil {
		return nil, err
	}

	baseQuery = baseQuery.Order("created_at DESC").Limit(filter.Limit).Offset(filter.Offset)

	var reports []*ComplianceReport

	if err := baseQuery.Scan(ctx); err != nil {
		return nil, err
	}

	params := &pagination.PaginationParams{
		Limit:  filter.Limit,
		Offset: filter.Offset,
	}

	return pagination.NewPageResponse(reports, int64(total), params), nil
}

func (r *BunRepository) UpdateReport(ctx context.Context, report *ComplianceReport) error {
	_, err := r.db.NewUpdate().Model(report).WherePK().Exec(ctx)

	return err
}

func (r *BunRepository) DeleteReport(ctx context.Context, id string) error {
	_, err := r.db.NewDelete().Model((*ComplianceReport)(nil)).Where("id = ?", id).Exec(ctx)

	return err
}

func (r *BunRepository) CreateEvidence(ctx context.Context, evidence *ComplianceEvidence) error {
	_, err := r.db.NewInsert().Model(evidence).Exec(ctx)

	return err
}

func (r *BunRepository) GetEvidence(ctx context.Context, id string) (*ComplianceEvidence, error) {
	evidence := new(ComplianceEvidence)

	err := r.db.NewSelect().Model(evidence).Where("id = ?", id).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	return evidence, err
}

func (r *BunRepository) ListEvidence(ctx context.Context, filter *ListEvidenceFilter) (*pagination.PageResponse[*ComplianceEvidence], error) {
	baseQuery := r.db.NewSelect().Model((*ComplianceEvidence)(nil))

	if filter.AppID != nil {
		baseQuery = baseQuery.Where("app_id = ?", *filter.AppID)
	}

	if filter.ProfileID != nil {
		baseQuery = baseQuery.Where("profile_id = ?", *filter.ProfileID)
	}

	if filter.EvidenceType != nil {
		baseQuery = baseQuery.Where("evidence_type = ?", *filter.EvidenceType)
	}

	if filter.Standard != nil {
		baseQuery = baseQuery.Where("standard = ?", *filter.Standard)
	}

	if filter.ControlID != nil {
		baseQuery = baseQuery.Where("control_id = ?", *filter.ControlID)
	}

	total, err := baseQuery.Count(ctx)
	if err != nil {
		return nil, err
	}

	baseQuery = baseQuery.Order("created_at DESC").Limit(filter.Limit).Offset(filter.Offset)

	var evidence []*ComplianceEvidence

	if err := baseQuery.Scan(ctx); err != nil {
		return nil, err
	}

	params := &pagination.PaginationParams{
		Limit:  filter.Limit,
		Offset: filter.Offset,
	}

	return pagination.NewPageResponse(evidence, int64(total), params), nil
}

func (r *BunRepository) DeleteEvidence(ctx context.Context, id string) error {
	_, err := r.db.NewDelete().Model((*ComplianceEvidence)(nil)).Where("id = ?", id).Exec(ctx)

	return err
}

func (r *BunRepository) CreatePolicy(ctx context.Context, policy *CompliancePolicy) error {
	_, err := r.db.NewInsert().Model(policy).Exec(ctx)

	return err
}

func (r *BunRepository) GetPolicy(ctx context.Context, id string) (*CompliancePolicy, error) {
	policy := new(CompliancePolicy)

	err := r.db.NewSelect().Model(policy).Where("id = ?", id).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	return policy, err
}

func (r *BunRepository) GetActivePolicies(ctx context.Context, appID string) ([]*CompliancePolicy, error) {
	var policies []*CompliancePolicy

	err := r.db.NewSelect().Model(&policies).
		Where("app_id = ?", appID).
		Where("status = ?", "active").
		Scan(ctx)

	return policies, err
}

func (r *BunRepository) ListPolicies(ctx context.Context, filter *ListPoliciesFilter) (*pagination.PageResponse[*CompliancePolicy], error) {
	baseQuery := r.db.NewSelect().Model((*CompliancePolicy)(nil))

	if filter.AppID != nil {
		baseQuery = baseQuery.Where("app_id = ?", *filter.AppID)
	}

	if filter.ProfileID != nil {
		baseQuery = baseQuery.Where("profile_id = ?", *filter.ProfileID)
	}

	if filter.PolicyType != nil {
		baseQuery = baseQuery.Where("policy_type = ?", *filter.PolicyType)
	}

	if filter.Standard != nil {
		baseQuery = baseQuery.Where("standard = ?", *filter.Standard)
	}

	if filter.Status != nil {
		baseQuery = baseQuery.Where("status = ?", *filter.Status)
	}

	total, err := baseQuery.Count(ctx)
	if err != nil {
		return nil, err
	}

	baseQuery = baseQuery.Order("created_at DESC").Limit(filter.Limit).Offset(filter.Offset)

	var policies []*CompliancePolicy

	if err := baseQuery.Scan(ctx); err != nil {
		return nil, err
	}

	params := &pagination.PaginationParams{
		Limit:  filter.Limit,
		Offset: filter.Offset,
	}

	return pagination.NewPageResponse(policies, int64(total), params), nil
}

func (r *BunRepository) UpdatePolicy(ctx context.Context, policy *CompliancePolicy) error {
	_, err := r.db.NewUpdate().Model(policy).WherePK().Exec(ctx)

	return err
}

func (r *BunRepository) DeletePolicy(ctx context.Context, id string) error {
	_, err := r.db.NewDelete().Model((*CompliancePolicy)(nil)).Where("id = ?", id).Exec(ctx)

	return err
}

func (r *BunRepository) CreateTraining(ctx context.Context, training *ComplianceTraining) error {
	_, err := r.db.NewInsert().Model(training).Exec(ctx)

	return err
}

func (r *BunRepository) GetTraining(ctx context.Context, id string) (*ComplianceTraining, error) {
	training := new(ComplianceTraining)

	err := r.db.NewSelect().Model(training).Where("id = ?", id).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	return training, err
}

func (r *BunRepository) ListTraining(ctx context.Context, filter *ListTrainingFilter) (*pagination.PageResponse[*ComplianceTraining], error) {
	baseQuery := r.db.NewSelect().Model((*ComplianceTraining)(nil))

	if filter.AppID != nil {
		baseQuery = baseQuery.Where("app_id = ?", *filter.AppID)
	}

	if filter.ProfileID != nil {
		baseQuery = baseQuery.Where("profile_id = ?", *filter.ProfileID)
	}

	if filter.UserID != nil {
		baseQuery = baseQuery.Where("user_id = ?", *filter.UserID)
	}

	if filter.TrainingType != nil {
		baseQuery = baseQuery.Where("training_type = ?", *filter.TrainingType)
	}

	if filter.Standard != nil {
		baseQuery = baseQuery.Where("standard = ?", *filter.Standard)
	}

	if filter.Status != nil {
		baseQuery = baseQuery.Where("status = ?", *filter.Status)
	}

	total, err := baseQuery.Count(ctx)
	if err != nil {
		return nil, err
	}

	baseQuery = baseQuery.Order("created_at DESC").Limit(filter.Limit).Offset(filter.Offset)

	var training []*ComplianceTraining

	if err := baseQuery.Scan(ctx); err != nil {
		return nil, err
	}

	params := &pagination.PaginationParams{
		Limit:  filter.Limit,
		Offset: filter.Offset,
	}

	return pagination.NewPageResponse(training, int64(total), params), nil
}

func (r *BunRepository) UpdateTraining(ctx context.Context, training *ComplianceTraining) error {
	_, err := r.db.NewUpdate().Model(training).WherePK().Exec(ctx)

	return err
}

func (r *BunRepository) GetUserTrainingStatus(ctx context.Context, userID string) ([]*ComplianceTraining, error) {
	var training []*ComplianceTraining

	err := r.db.NewSelect().Model(&training).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Scan(ctx)

	return training, err
}

func (r *BunRepository) GetOverdueTraining(ctx context.Context, appID string) ([]*ComplianceTraining, error) {
	var training []*ComplianceTraining

	err := r.db.NewSelect().Model(&training).
		Where("app_id = ?", appID).
		Where("status != ?", "completed").
		Where("expires_at < NOW()").
		Scan(ctx)

	return training, err
}
