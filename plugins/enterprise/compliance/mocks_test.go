package compliance

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/xraph/authsome/core/pagination"
)

// MockRepository implements Repository interface for testing
type MockRepository struct {
	mu         sync.RWMutex
	profiles   map[string]*ComplianceProfile
	checks     map[string]*ComplianceCheck
	violations map[string]*ComplianceViolation
	reports    map[string]*ComplianceReport
	evidence   map[string]*ComplianceEvidence
	policies   map[string]*CompliancePolicy
	training   map[string]*ComplianceTraining
}

// NewMockRepository creates a new mock repository
func NewMockRepository() *MockRepository {
	return &MockRepository{
		profiles:   make(map[string]*ComplianceProfile),
		checks:     make(map[string]*ComplianceCheck),
		violations: make(map[string]*ComplianceViolation),
		reports:    make(map[string]*ComplianceReport),
		evidence:   make(map[string]*ComplianceEvidence),
		policies:   make(map[string]*CompliancePolicy),
		training:   make(map[string]*ComplianceTraining),
	}
}

// Profile methods

func (m *MockRepository) CreateProfile(ctx context.Context, profile *ComplianceProfile) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if profile.ID == "" {
		profile.ID = uuid.New().String()
	}
	m.profiles[profile.ID] = profile
	return nil
}

func (m *MockRepository) GetProfile(ctx context.Context, id string) (*ComplianceProfile, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	profile, ok := m.profiles[id]
	if !ok {
		return nil, ProfileNotFound(id)
	}
	return profile, nil
}

func (m *MockRepository) UpdateProfile(ctx context.Context, profile *ComplianceProfile) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.profiles[profile.ID]; !ok {
		return ProfileNotFound(profile.ID)
	}
	m.profiles[profile.ID] = profile
	return nil
}

func (m *MockRepository) DeleteProfile(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.profiles[id]; !ok {
		return ProfileNotFound(id)
	}
	delete(m.profiles, id)
	return nil
}

func (m *MockRepository) GetProfileByApp(ctx context.Context, appID string) (*ComplianceProfile, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, profile := range m.profiles {
		if profile.AppID == appID {
			return profile, nil
		}
	}
	return nil, ProfileNotFound(appID)
}

func (m *MockRepository) ListProfiles(ctx context.Context, filter *ListProfilesFilter) (*pagination.PageResponse[*ComplianceProfile], error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var profiles []*ComplianceProfile
	for _, profile := range m.profiles {
		if filter.AppID != nil && profile.AppID != *filter.AppID {
			continue
		}
		profiles = append(profiles, profile)
	}

	return &pagination.PageResponse[*ComplianceProfile]{
		Data: profiles,
	}, nil
}

// Check methods

func (m *MockRepository) CreateCheck(ctx context.Context, check *ComplianceCheck) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if check.ID == "" {
		check.ID = uuid.New().String()
	}
	m.checks[check.ID] = check
	return nil
}

func (m *MockRepository) GetCheck(ctx context.Context, id string) (*ComplianceCheck, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	check, ok := m.checks[id]
	if !ok {
		return nil, CheckNotFound(id)
	}
	return check, nil
}

func (m *MockRepository) UpdateCheck(ctx context.Context, check *ComplianceCheck) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.checks[check.ID] = check
	return nil
}

func (m *MockRepository) ListChecks(ctx context.Context, filter *ListChecksFilter) (*pagination.PageResponse[*ComplianceCheck], error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var checks []*ComplianceCheck
	for _, check := range m.checks {
		checks = append(checks, check)
	}

	return &pagination.PageResponse[*ComplianceCheck]{
		Data: checks,
	}, nil
}

func (m *MockRepository) GetDueChecks(ctx context.Context) ([]*ComplianceCheck, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var checks []*ComplianceCheck
	for _, check := range m.checks {
		checks = append(checks, check)
	}
	return checks, nil
}

// Violation methods

func (m *MockRepository) CreateViolation(ctx context.Context, violation *ComplianceViolation) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if violation.ID == "" {
		violation.ID = uuid.New().String()
	}
	m.violations[violation.ID] = violation
	return nil
}

func (m *MockRepository) GetViolation(ctx context.Context, id string) (*ComplianceViolation, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	violation, ok := m.violations[id]
	if !ok {
		return nil, ViolationNotFound(id)
	}
	return violation, nil
}

func (m *MockRepository) ListViolations(ctx context.Context, filter *ListViolationsFilter) (*pagination.PageResponse[*ComplianceViolation], error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var violations []*ComplianceViolation
	for _, violation := range m.violations {
		violations = append(violations, violation)
	}

	return &pagination.PageResponse[*ComplianceViolation]{
		Data: violations,
	}, nil
}

func (m *MockRepository) UpdateViolation(ctx context.Context, violation *ComplianceViolation) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.violations[violation.ID] = violation
	return nil
}

func (m *MockRepository) ResolveViolation(ctx context.Context, id string, resolvedBy string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	violation, ok := m.violations[id]
	if !ok {
		return ViolationNotFound(id)
	}
	violation.Status = "resolved"
	violation.ResolvedBy = resolvedBy
	return nil
}

func (m *MockRepository) CountViolations(ctx context.Context, appID string, status string) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, violation := range m.violations {
		if violation.AppID == appID && (status == "" || violation.Status == status) {
			count++
		}
	}
	return count, nil
}

// Report methods

func (m *MockRepository) CreateReport(ctx context.Context, report *ComplianceReport) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if report.ID == "" {
		report.ID = uuid.New().String()
	}
	m.reports[report.ID] = report
	return nil
}

func (m *MockRepository) GetReport(ctx context.Context, id string) (*ComplianceReport, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	report, ok := m.reports[id]
	if !ok {
		return nil, ReportNotFound(id)
	}
	return report, nil
}

func (m *MockRepository) ListReports(ctx context.Context, filter *ListReportsFilter) (*pagination.PageResponse[*ComplianceReport], error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var reports []*ComplianceReport
	for _, report := range m.reports {
		reports = append(reports, report)
	}

	return &pagination.PageResponse[*ComplianceReport]{
		Data: reports,
	}, nil
}

func (m *MockRepository) UpdateReport(ctx context.Context, report *ComplianceReport) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.reports[report.ID] = report
	return nil
}

func (m *MockRepository) DeleteReport(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.reports, id)
	return nil
}

// Evidence methods

func (m *MockRepository) CreateEvidence(ctx context.Context, evidence *ComplianceEvidence) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if evidence.ID == "" {
		evidence.ID = uuid.New().String()
	}
	m.evidence[evidence.ID] = evidence
	return nil
}

func (m *MockRepository) GetEvidence(ctx context.Context, id string) (*ComplianceEvidence, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	evidence, ok := m.evidence[id]
	if !ok {
		return nil, EvidenceNotFound(id)
	}
	return evidence, nil
}

func (m *MockRepository) DeleteEvidence(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.evidence, id)
	return nil
}

func (m *MockRepository) ListEvidence(ctx context.Context, filter *ListEvidenceFilter) (*pagination.PageResponse[*ComplianceEvidence], error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var evidenceList []*ComplianceEvidence
	for _, evidence := range m.evidence {
		evidenceList = append(evidenceList, evidence)
	}

	return &pagination.PageResponse[*ComplianceEvidence]{
		Data: evidenceList,
	}, nil
}

// Policy methods

func (m *MockRepository) CreatePolicy(ctx context.Context, policy *CompliancePolicy) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if policy.ID == "" {
		policy.ID = uuid.New().String()
	}
	m.policies[policy.ID] = policy
	return nil
}

func (m *MockRepository) GetPolicy(ctx context.Context, id string) (*CompliancePolicy, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	policy, ok := m.policies[id]
	if !ok {
		return nil, PolicyNotFound(id)
	}
	return policy, nil
}

func (m *MockRepository) UpdatePolicy(ctx context.Context, policy *CompliancePolicy) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.policies[policy.ID] = policy
	return nil
}

func (m *MockRepository) DeletePolicy(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.policies, id)
	return nil
}

func (m *MockRepository) ListPolicies(ctx context.Context, filter *ListPoliciesFilter) (*pagination.PageResponse[*CompliancePolicy], error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var policies []*CompliancePolicy
	for _, policy := range m.policies {
		policies = append(policies, policy)
	}

	return &pagination.PageResponse[*CompliancePolicy]{
		Data: policies,
	}, nil
}

func (m *MockRepository) GetActivePolicies(ctx context.Context, appID string) ([]*CompliancePolicy, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var policies []*CompliancePolicy
	for _, policy := range m.policies {
		if policy.AppID == appID && policy.Status == "active" {
			policies = append(policies, policy)
		}
	}
	return policies, nil
}

// Training methods

func (m *MockRepository) CreateTraining(ctx context.Context, training *ComplianceTraining) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if training.ID == "" {
		training.ID = uuid.New().String()
	}
	m.training[training.ID] = training
	return nil
}

func (m *MockRepository) GetTraining(ctx context.Context, id string) (*ComplianceTraining, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	training, ok := m.training[id]
	if !ok {
		return nil, TrainingNotFound(id)
	}
	return training, nil
}

func (m *MockRepository) UpdateTraining(ctx context.Context, training *ComplianceTraining) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.training[training.ID] = training
	return nil
}

func (m *MockRepository) GetUserTrainingStatus(ctx context.Context, userID string) ([]*ComplianceTraining, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*ComplianceTraining
	for _, training := range m.training {
		if training.UserID == userID {
			result = append(result, training)
		}
	}
	return result, nil
}

func (m *MockRepository) ListTraining(ctx context.Context, filter *ListTrainingFilter) (*pagination.PageResponse[*ComplianceTraining], error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var trainingList []*ComplianceTraining
	for _, training := range m.training {
		trainingList = append(trainingList, training)
	}

	return &pagination.PageResponse[*ComplianceTraining]{
		Data: trainingList,
	}, nil
}

func (m *MockRepository) GetOverdueTraining(ctx context.Context, appID string) ([]*ComplianceTraining, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var trainingList []*ComplianceTraining
	for _, training := range m.training {
		if training.AppID == appID {
			trainingList = append(trainingList, training)
		}
	}
	return trainingList, nil
}

// MockAuditService implements AuditService for testing
type MockAuditService struct{}

func (m *MockAuditService) LogEvent(ctx context.Context, event *AuditEvent) error {
	return nil
}

func (m *MockAuditService) GetOldestLog(ctx context.Context, appID string) (*AuditLog, error) {
	return nil, nil
}

// MockUserService implements UserService for testing
type MockUserService struct{}

func (m *MockUserService) ListByApp(ctx context.Context, appID string) ([]*User, error) {
	return []*User{}, nil
}

// MockAppService implements AppService for testing
type MockAppService struct{}

func (m *MockAppService) Get(ctx context.Context, id string) (*App, error) {
	return &App{}, nil
}

// MockEmailService implements EmailService for testing
type MockEmailService struct{}

func (m *MockEmailService) SendEmail(ctx context.Context, email *Email) error {
	return nil
}
