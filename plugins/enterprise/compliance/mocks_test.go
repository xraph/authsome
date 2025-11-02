package compliance

import (
	"context"
	"sync"
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
	m.profiles[profile.ID] = profile
	return nil
}

func (m *MockRepository) GetProfile(ctx context.Context, id string) (*ComplianceProfile, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	profile, ok := m.profiles[id]
	if !ok {
		return nil, ErrProfileNotFound
	}
	return profile, nil
}

func (m *MockRepository) UpdateProfile(ctx context.Context, profile *ComplianceProfile) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.profiles[profile.ID]; !ok {
		return ErrProfileNotFound
	}
	m.profiles[profile.ID] = profile
	return nil
}

func (m *MockRepository) DeleteProfile(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.profiles[id]; !ok {
		return ErrProfileNotFound
	}
	delete(m.profiles, id)
	return nil
}

func (m *MockRepository) GetProfileByOrganization(ctx context.Context, orgID string) (*ComplianceProfile, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, profile := range m.profiles {
		if profile.OrganizationID == orgID {
			return profile, nil
		}
	}
	return nil, ErrProfileNotFound
}

// Check methods

func (m *MockRepository) CreateCheck(ctx context.Context, check *ComplianceCheck) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.checks[check.ID] = check
	return nil
}

func (m *MockRepository) GetCheck(ctx context.Context, id string) (*ComplianceCheck, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	check, ok := m.checks[id]
	if !ok {
		return nil, ErrCheckNotFound
	}
	return check, nil
}

func (m *MockRepository) ListChecks(ctx context.Context, filters CheckFilters) ([]*ComplianceCheck, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*ComplianceCheck
	for _, check := range m.checks {
		if filters.ProfileID != "" && check.ProfileID != filters.ProfileID {
			continue
		}
		if filters.CheckType != "" && check.CheckType != filters.CheckType {
			continue
		}
		result = append(result, check)
	}
	return result, nil
}

// Violation methods

func (m *MockRepository) CreateViolation(ctx context.Context, violation *ComplianceViolation) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.violations[violation.ID] = violation
	return nil
}

func (m *MockRepository) GetViolation(ctx context.Context, id string) (*ComplianceViolation, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	violation, ok := m.violations[id]
	if !ok {
		return nil, ErrViolationNotFound
	}
	return violation, nil
}

func (m *MockRepository) ListViolations(ctx context.Context, filters ViolationFilters) ([]*ComplianceViolation, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*ComplianceViolation
	for _, violation := range m.violations {
		if filters.OrganizationID != "" && violation.OrganizationID != filters.OrganizationID {
			continue
		}
		if filters.ViolationType != "" && violation.ViolationType != filters.ViolationType {
			continue
		}
		if filters.Status != "" && violation.Status != filters.Status {
			continue
		}
		result = append(result, violation)
	}
	return result, nil
}

func (m *MockRepository) ResolveViolation(ctx context.Context, id, resolvedBy string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	violation, ok := m.violations[id]
	if !ok {
		return ErrViolationNotFound
	}
	violation.Status = "resolved"
	violation.ResolvedBy = resolvedBy
	return nil
}

// Report methods

func (m *MockRepository) CreateReport(ctx context.Context, report *ComplianceReport) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.reports[report.ID] = report
	return nil
}

func (m *MockRepository) GetReport(ctx context.Context, id string) (*ComplianceReport, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	report, ok := m.reports[id]
	if !ok {
		return nil, ErrReportNotFound
	}
	return report, nil
}

func (m *MockRepository) ListReports(ctx context.Context, filters ReportFilters) ([]*ComplianceReport, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*ComplianceReport
	for _, report := range m.reports {
		if filters.OrganizationID != "" && report.OrganizationID != filters.OrganizationID {
			continue
		}
		if filters.Standard != "" && report.Standard != filters.Standard {
			continue
		}
		result = append(result, report)
	}
	return result, nil
}

// Evidence methods

func (m *MockRepository) CreateEvidence(ctx context.Context, evidence *ComplianceEvidence) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.evidence[evidence.ID] = evidence
	return nil
}

func (m *MockRepository) GetEvidence(ctx context.Context, id string) (*ComplianceEvidence, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	evidence, ok := m.evidence[id]
	if !ok {
		return nil, ErrEvidenceNotFound
	}
	return evidence, nil
}

func (m *MockRepository) ListEvidence(ctx context.Context, filters EvidenceFilters) ([]*ComplianceEvidence, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*ComplianceEvidence
	for _, evidence := range m.evidence {
		if filters.ProfileID != "" && evidence.ProfileID != filters.ProfileID {
			continue
		}
		if filters.Category != "" && evidence.Category != filters.Category {
			continue
		}
		result = append(result, evidence)
	}
	return result, nil
}

func (m *MockRepository) DeleteEvidence(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.evidence[id]; !ok {
		return ErrEvidenceNotFound
	}
	delete(m.evidence, id)
	return nil
}

// Policy methods

func (m *MockRepository) CreatePolicy(ctx context.Context, policy *CompliancePolicy) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.policies[policy.ID] = policy
	return nil
}

func (m *MockRepository) GetPolicy(ctx context.Context, id string) (*CompliancePolicy, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	policy, ok := m.policies[id]
	if !ok {
		return nil, ErrPolicyNotFound
	}
	return policy, nil
}

func (m *MockRepository) ListPolicies(ctx context.Context, filters PolicyFilters) ([]*CompliancePolicy, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*CompliancePolicy
	for _, policy := range m.policies {
		if filters.OrganizationID != "" && policy.OrganizationID != filters.OrganizationID {
			continue
		}
		if filters.PolicyType != "" && policy.PolicyType != filters.PolicyType {
			continue
		}
		result = append(result, policy)
	}
	return result, nil
}

func (m *MockRepository) UpdatePolicy(ctx context.Context, policy *CompliancePolicy) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.policies[policy.ID]; !ok {
		return ErrPolicyNotFound
	}
	m.policies[policy.ID] = policy
	return nil
}

func (m *MockRepository) DeletePolicy(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.policies[id]; !ok {
		return ErrPolicyNotFound
	}
	delete(m.policies, id)
	return nil
}

// Training methods

func (m *MockRepository) CreateTraining(ctx context.Context, training *ComplianceTraining) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.training[training.ID] = training
	return nil
}

func (m *MockRepository) GetTraining(ctx context.Context, id string) (*ComplianceTraining, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	training, ok := m.training[id]
	if !ok {
		return nil, ErrTrainingNotFound
	}
	return training, nil
}

func (m *MockRepository) ListTraining(ctx context.Context, filters TrainingFilters) ([]*ComplianceTraining, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*ComplianceTraining
	for _, training := range m.training {
		if filters.OrganizationID != "" && training.OrganizationID != filters.OrganizationID {
			continue
		}
		if filters.UserID != "" && training.UserID != filters.UserID {
			continue
		}
		if filters.Status != "" && training.Status != filters.Status {
			continue
		}
		result = append(result, training)
	}
	return result, nil
}

func (m *MockRepository) CompleteTraining(ctx context.Context, id, completedBy string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	training, ok := m.training[id]
	if !ok {
		return ErrTrainingNotFound
	}
	training.Status = "completed"
	return nil
}

// Mock service adapters

// MockAuditService simulates audit logging
type MockAuditService struct {
	mu     sync.Mutex
	events []*AuditEvent
}

func NewMockAuditService() *MockAuditService {
	return &MockAuditService{
		events: make([]*AuditEvent, 0),
	}
}

func (m *MockAuditService) LogEvent(ctx context.Context, event *AuditEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, event)
	return nil
}

func (m *MockAuditService) GetOldestLog(ctx context.Context, orgID string) (*AuditLog, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.events) == 0 {
		return nil, nil
	}
	return &AuditLog{}, nil
}

// MockUserService simulates user operations
type MockUserService struct {
	mu    sync.Mutex
	users []*User
}

func NewMockUserService() *MockUserService {
	return &MockUserService{
		users: make([]*User, 0),
	}
}

func (m *MockUserService) ListByOrganization(ctx context.Context, orgID string) ([]*User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.users, nil
}

// MockOrganizationService simulates org operations
type MockOrganizationService struct {
	mu   sync.Mutex
	orgs map[string]*Organization
}

func NewMockOrganizationService() *MockOrganizationService {
	return &MockOrganizationService{
		orgs: make(map[string]*Organization),
	}
}

func (m *MockOrganizationService) Get(ctx context.Context, id string) (*Organization, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	org, ok := m.orgs[id]
	if !ok {
		return nil, ErrOrganizationNotFound
	}
	return org, nil
}

// MockEmailService simulates email sending
type MockEmailService struct {
	mu   sync.Mutex
	sent []*Email
}

func NewMockEmailService() *MockEmailService {
	return &MockEmailService{
		sent: make([]*Email, 0),
	}
}

func (m *MockEmailService) SendEmail(ctx context.Context, email *Email) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sent = append(m.sent, email)
	return nil
}

func (m *MockEmailService) GetSentEmails() []*Email {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.sent
}

