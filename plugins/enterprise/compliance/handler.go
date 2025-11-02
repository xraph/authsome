package compliance

import (
	"net/http"
	"time"
)

// Handler handles HTTP requests for compliance endpoints
type Handler struct {
	service      *Service
	policyEngine *PolicyEngine
}

// NewHandler creates a new compliance handler
func NewHandler(service *Service, policyEngine *PolicyEngine) *Handler {
	return &Handler{
		service:      service,
		policyEngine: policyEngine,
	}
}

// ===== Profile Handlers =====

// CreateProfile creates a new compliance profile
// POST /auth/compliance/profiles
func (h *Handler) CreateProfile(c Context) error {
	var req CreateProfileRequest
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Message: "Invalid request body",
			Error:   err.Error(),
		})
	}
	
	profile, err := h.service.CreateProfile(c.Request().Context(), &req)
	if err != nil {
		return handleError(c, err)
	}
	
	return c.JSON(http.StatusCreated, profile)
}

// CreateProfileFromTemplate creates a profile from a template
// POST /auth/compliance/profiles/from-template
func (h *Handler) CreateProfileFromTemplate(c Context) error {
	var req struct {
		OrganizationID string             `json:"organizationId" validate:"required"`
		Standard       ComplianceStandard `json:"standard" validate:"required"`
	}
	
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Message: "Invalid request body",
		})
	}
	
	profile, err := h.service.CreateProfileFromTemplate(c.Request().Context(), req.OrganizationID, req.Standard)
	if err != nil {
		return handleError(c, err)
	}
	
	return c.JSON(http.StatusCreated, profile)
}

// GetProfile retrieves a compliance profile
// GET /auth/compliance/profiles/:id
func (h *Handler) GetProfile(c Context) error {
	id := c.Param("id")
	
	profile, err := h.service.GetProfile(c.Request().Context(), id)
	if err != nil {
		return handleError(c, err)
	}
	
	return c.JSON(http.StatusOK, profile)
}

// GetOrganizationProfile retrieves the compliance profile for an organization
// GET /auth/compliance/organizations/:orgId/profile
func (h *Handler) GetOrganizationProfile(c Context) error {
	orgID := c.Param("orgId")
	
	profile, err := h.service.GetProfileByOrganization(c.Request().Context(), orgID)
	if err != nil {
		return handleError(c, err)
	}
	
	return c.JSON(http.StatusOK, profile)
}

// UpdateProfile updates a compliance profile
// PUT /auth/compliance/profiles/:id
func (h *Handler) UpdateProfile(c Context) error {
	id := c.Param("id")
	
	var req UpdateProfileRequest
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Message: "Invalid request body",
		})
	}
	
	profile, err := h.service.UpdateProfile(c.Request().Context(), id, &req)
	if err != nil {
		return handleError(c, err)
	}
	
	return c.JSON(http.StatusOK, profile)
}

// DeleteProfile deletes a compliance profile
// DELETE /auth/compliance/profiles/:id
func (h *Handler) DeleteProfile(c Context) error {
	id := c.Param("id")
	
	if err := h.service.repo.DeleteProfile(c.Request().Context(), id); err != nil {
		return handleError(c, err)
	}
	
	return c.NoContent(http.StatusNoContent)
}

// ===== Status & Dashboard Handlers =====

// GetComplianceStatus gets overall compliance status for an organization
// GET /auth/compliance/organizations/:orgId/status
func (h *Handler) GetComplianceStatus(c Context) error {
	orgID := c.Param("orgId")
	
	status, err := h.service.GetComplianceStatus(c.Request().Context(), orgID)
	if err != nil {
		return handleError(c, err)
	}
	
	return c.JSON(http.StatusOK, status)
}

// GetDashboard gets compliance dashboard data
// GET /auth/compliance/organizations/:orgId/dashboard
func (h *Handler) GetDashboard(c Context) error {
	orgID := c.Param("orgId")
	ctx := c.Request().Context()
	
	// Get profile
	profile, err := h.service.GetProfileByOrganization(ctx, orgID)
	if err != nil {
		return handleError(c, err)
	}
	
	// Get status
	status, _ := h.service.GetComplianceStatus(ctx, orgID)
	
	// Get recent checks
	checks, _ := h.service.repo.ListChecks(ctx, profile.ID, CheckFilters{
		Limit: 10,
	})
	
	// Get open violations
	violations, _ := h.service.repo.ListViolations(ctx, ViolationFilters{
		OrganizationID: orgID,
		Status:         "open",
		Limit:          10,
	})
	
	// Get recent reports
	reports, _ := h.service.repo.ListReports(ctx, ReportFilters{
		OrganizationID: orgID,
		Limit:          5,
	})
	
	dashboard := map[string]interface{}{
		"profile":    profile,
		"status":     status,
		"checks":     checks,
		"violations": violations,
		"reports":    reports,
	}
	
	return c.JSON(http.StatusOK, dashboard)
}

// ===== Check Handlers =====

// RunCheck executes a compliance check
// POST /auth/compliance/profiles/:profileId/checks
func (h *Handler) RunCheck(c Context) error {
	profileID := c.Param("profileId")
	
	var req struct {
		CheckType string `json:"checkType" validate:"required"`
	}
	
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Message: "Invalid request body",
		})
	}
	
	check, err := h.service.RunCheck(c.Request().Context(), profileID, req.CheckType)
	if err != nil {
		return handleError(c, err)
	}
	
	return c.JSON(http.StatusCreated, check)
}

// ListChecks lists compliance checks
// GET /auth/compliance/profiles/:profileId/checks
func (h *Handler) ListChecks(c Context) error {
	profileID := c.Param("profileId")
	
	filters := CheckFilters{
		ProfileID: profileID,
		CheckType: c.Query("checkType"),
		Status:    c.Query("status"),
		Limit:     c.QueryInt("limit", 20),
		Offset:    c.QueryInt("offset", 0),
	}
	
	checks, err := h.service.repo.ListChecks(c.Request().Context(), profileID, filters)
	if err != nil {
		return handleError(c, err)
	}
	
	return c.JSON(http.StatusOK, checks)
}

// GetCheck retrieves a compliance check
// GET /auth/compliance/checks/:id
func (h *Handler) GetCheck(c Context) error {
	id := c.Param("id")
	
	check, err := h.service.repo.GetCheck(c.Request().Context(), id)
	if err != nil {
		return handleError(c, err)
	}
	
	return c.JSON(http.StatusOK, check)
}

// ===== Violation Handlers =====

// ListViolations lists compliance violations
// GET /auth/compliance/organizations/:orgId/violations
func (h *Handler) ListViolations(c Context) error {
	orgID := c.Param("orgId")
	
	filters := ViolationFilters{
		OrganizationID: orgID,
		UserID:         c.Query("userId"),
		ViolationType:  c.Query("violationType"),
		Severity:       c.Query("severity"),
		Status:         c.Query("status"),
		Limit:          c.QueryInt("limit", 20),
		Offset:         c.QueryInt("offset", 0),
	}
	
	violations, err := h.service.repo.ListViolations(c.Request().Context(), filters)
	if err != nil {
		return handleError(c, err)
	}
	
	return c.JSON(http.StatusOK, violations)
}

// GetViolation retrieves a compliance violation
// GET /auth/compliance/violations/:id
func (h *Handler) GetViolation(c Context) error {
	id := c.Param("id")
	
	violation, err := h.service.repo.GetViolation(c.Request().Context(), id)
	if err != nil {
		return handleError(c, err)
	}
	
	return c.JSON(http.StatusOK, violation)
}

// ResolveViolation resolves a compliance violation
// PUT /auth/compliance/violations/:id/resolve
func (h *Handler) ResolveViolation(c Context) error {
	id := c.Param("id")
	resolvedBy := c.Get("user_id").(string) // From auth middleware
	
	if err := h.service.repo.ResolveViolation(c.Request().Context(), id, resolvedBy); err != nil {
		return handleError(c, err)
	}
	
	violation, _ := h.service.repo.GetViolation(c.Request().Context(), id)
	
	return c.JSON(http.StatusOK, violation)
}

// ===== Report Handlers =====

// GenerateReport generates a compliance report
// POST /auth/compliance/organizations/:orgId/reports
func (h *Handler) GenerateReport(c Context) error {
	orgID := c.Param("orgId")
	
	var req struct {
		ReportType string             `json:"reportType" validate:"required"`
		Standard   ComplianceStandard `json:"standard"`
		Period     string             `json:"period" validate:"required"`
		Format     string             `json:"format" validate:"required"`
	}
	
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Message: "Invalid request body",
		})
	}
	
	// Create report record
	report := &ComplianceReport{
		OrganizationID: orgID,
		ReportType:     req.ReportType,
		Standard:       req.Standard,
		Period:         req.Period,
		Format:         req.Format,
		Status:         "generating",
		GeneratedBy:    c.Get("user_id").(string),
	}
	
	if err := h.service.repo.CreateReport(c.Request().Context(), report); err != nil {
		return handleError(c, err)
	}
	
	// Generate report asynchronously
	go h.generateReportAsync(report)
	
	return c.JSON(http.StatusAccepted, report)
}

// ListReports lists compliance reports
// GET /auth/compliance/organizations/:orgId/reports
func (h *Handler) ListReports(c Context) error {
	orgID := c.Param("orgId")
	
	filters := ReportFilters{
		OrganizationID: orgID,
		ReportType:     c.Query("reportType"),
		Status:         c.Query("status"),
		Limit:          c.QueryInt("limit", 20),
		Offset:         c.QueryInt("offset", 0),
	}
	
	reports, err := h.service.repo.ListReports(c.Request().Context(), filters)
	if err != nil {
		return handleError(c, err)
	}
	
	return c.JSON(http.StatusOK, reports)
}

// GetReport retrieves a compliance report
// GET /auth/compliance/reports/:id
func (h *Handler) GetReport(c Context) error {
	id := c.Param("id")
	
	report, err := h.service.repo.GetReport(c.Request().Context(), id)
	if err != nil {
		return handleError(c, err)
	}
	
	return c.JSON(http.StatusOK, report)
}

// DownloadReport downloads a compliance report file
// GET /auth/compliance/reports/:id/download
func (h *Handler) DownloadReport(c Context) error {
	id := c.Param("id")
	
	report, err := h.service.repo.GetReport(c.Request().Context(), id)
	if err != nil {
		return handleError(c, err)
	}
	
	if report.Status != "ready" {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Message: "Report is not ready for download",
		})
	}
	
	// In real implementation, this would stream the file from storage
	return c.Redirect(http.StatusFound, report.FileURL)
}

// generateReportAsync generates report asynchronously
func (h *Handler) generateReportAsync(report *ComplianceReport) {
	// This would be implemented with proper report generation logic
	// For now, it's a placeholder
}

// ===== Evidence Handlers =====

// CreateEvidence creates compliance evidence
// POST /auth/compliance/organizations/:orgId/evidence
func (h *Handler) CreateEvidence(c Context) error {
	orgID := c.Param("orgId")
	
	var req struct {
		EvidenceType string             `json:"evidenceType" validate:"required"`
		Standard     ComplianceStandard `json:"standard"`
		ControlID    string             `json:"controlId"`
		Title        string             `json:"title" validate:"required"`
		Description  string             `json:"description"`
		FileURL      string             `json:"fileUrl"`
	}
	
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Message: "Invalid request body",
		})
	}
	
	// Get profile
	profile, err := h.service.GetProfileByOrganization(c.Request().Context(), orgID)
	if err != nil {
		return handleError(c, err)
	}
	
	evidence := &ComplianceEvidence{
		ProfileID:      profile.ID,
		OrganizationID: orgID,
		EvidenceType:   req.EvidenceType,
		Standard:       req.Standard,
		ControlID:      req.ControlID,
		Title:          req.Title,
		Description:    req.Description,
		FileURL:        req.FileURL,
		CollectedBy:    c.Get("user_id").(string),
	}
	
	if err := h.service.repo.CreateEvidence(c.Request().Context(), evidence); err != nil {
		return handleError(c, err)
	}
	
	return c.JSON(http.StatusCreated, evidence)
}

// ListEvidence lists compliance evidence
// GET /auth/compliance/organizations/:orgId/evidence
func (h *Handler) ListEvidence(c Context) error {
	orgID := c.Param("orgId")
	
	filters := EvidenceFilters{
		OrganizationID: orgID,
		EvidenceType:   c.Query("evidenceType"),
		ControlID:      c.Query("controlId"),
		Limit:          c.QueryInt("limit", 20),
		Offset:         c.QueryInt("offset", 0),
	}
	
	evidence, err := h.service.repo.ListEvidence(c.Request().Context(), filters)
	if err != nil {
		return handleError(c, err)
	}
	
	return c.JSON(http.StatusOK, evidence)
}

// GetEvidence retrieves compliance evidence
// GET /auth/compliance/evidence/:id
func (h *Handler) GetEvidence(c Context) error {
	id := c.Param("id")
	
	evidence, err := h.service.repo.GetEvidence(c.Request().Context(), id)
	if err != nil {
		return handleError(c, err)
	}
	
	return c.JSON(http.StatusOK, evidence)
}

// DeleteEvidence deletes compliance evidence
// DELETE /auth/compliance/evidence/:id
func (h *Handler) DeleteEvidence(c Context) error {
	id := c.Param("id")
	
	if err := h.service.repo.DeleteEvidence(c.Request().Context(), id); err != nil {
		return handleError(c, err)
	}
	
	return c.NoContent(http.StatusNoContent)
}

// ===== Policy Handlers =====

// CreatePolicy creates a compliance policy
// POST /auth/compliance/organizations/:orgId/policies
func (h *Handler) CreatePolicy(c Context) error {
	orgID := c.Param("orgId")
	
	var req struct {
		PolicyType string             `json:"policyType" validate:"required"`
		Standard   ComplianceStandard `json:"standard"`
		Title      string             `json:"title" validate:"required"`
		Version    string             `json:"version" validate:"required"`
		Content    string             `json:"content" validate:"required"`
	}
	
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Message: "Invalid request body",
		})
	}
	
	// Get profile
	profile, err := h.service.GetProfileByOrganization(c.Request().Context(), orgID)
	if err != nil {
		return handleError(c, err)
	}
	
	policy := &CompliancePolicy{
		ProfileID:      profile.ID,
		OrganizationID: orgID,
		PolicyType:     req.PolicyType,
		Standard:       req.Standard,
		Title:          req.Title,
		Version:        req.Version,
		Content:        req.Content,
		Status:         "draft",
	}
	
	if err := h.service.repo.CreatePolicy(c.Request().Context(), policy); err != nil {
		return handleError(c, err)
	}
	
	return c.JSON(http.StatusCreated, policy)
}

// ListPolicies lists compliance policies
// GET /auth/compliance/organizations/:orgId/policies
func (h *Handler) ListPolicies(c Context) error {
	orgID := c.Param("orgId")
	
	filters := PolicyFilters{
		OrganizationID: orgID,
		PolicyType:     c.Query("policyType"),
		Status:         c.Query("status"),
		Limit:          c.QueryInt("limit", 20),
		Offset:         c.QueryInt("offset", 0),
	}
	
	policies, err := h.service.repo.ListPolicies(c.Request().Context(), filters)
	if err != nil {
		return handleError(c, err)
	}
	
	return c.JSON(http.StatusOK, policies)
}

// GetPolicy retrieves a compliance policy
// GET /auth/compliance/policies/:id
func (h *Handler) GetPolicy(c Context) error {
	id := c.Param("id")
	
	policy, err := h.service.repo.GetPolicy(c.Request().Context(), id)
	if err != nil {
		return handleError(c, err)
	}
	
	return c.JSON(http.StatusOK, policy)
}

// UpdatePolicy updates a compliance policy
// PUT /auth/compliance/policies/:id
func (h *Handler) UpdatePolicy(c Context) error {
	id := c.Param("id")
	
	var req struct {
		Title   *string `json:"title"`
		Version *string `json:"version"`
		Content *string `json:"content"`
		Status  *string `json:"status"`
	}
	
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Message: "Invalid request body",
		})
	}
	
	policy, err := h.service.repo.GetPolicy(c.Request().Context(), id)
	if err != nil {
		return handleError(c, err)
	}
	
	// Apply updates
	if req.Title != nil {
		policy.Title = *req.Title
	}
	if req.Version != nil {
		policy.Version = *req.Version
	}
	if req.Content != nil {
		policy.Content = *req.Content
	}
	if req.Status != nil {
		policy.Status = *req.Status
	}
	
	if err := h.service.repo.UpdatePolicy(c.Request().Context(), policy); err != nil {
		return handleError(c, err)
	}
	
	return c.JSON(http.StatusOK, policy)
}

// DeletePolicy deletes a compliance policy
// DELETE /auth/compliance/policies/:id
func (h *Handler) DeletePolicy(c Context) error {
	id := c.Param("id")
	
	if err := h.service.repo.DeletePolicy(c.Request().Context(), id); err != nil {
		return handleError(c, err)
	}
	
	return c.NoContent(http.StatusNoContent)
}

// ===== Training Handlers =====

// CreateTraining creates a training record
// POST /auth/compliance/organizations/:orgId/training
func (h *Handler) CreateTraining(c Context) error {
	orgID := c.Param("orgId")
	
	var req struct {
		UserID       string             `json:"userId" validate:"required"`
		TrainingType string             `json:"trainingType" validate:"required"`
		Standard     ComplianceStandard `json:"standard"`
	}
	
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Message: "Invalid request body",
		})
	}
	
	// Get profile
	profile, err := h.service.GetProfileByOrganization(c.Request().Context(), orgID)
	if err != nil {
		return handleError(c, err)
	}
	
	training := &ComplianceTraining{
		ProfileID:      profile.ID,
		OrganizationID: orgID,
		UserID:         req.UserID,
		TrainingType:   req.TrainingType,
		Standard:       req.Standard,
		Status:         "required",
	}
	
	if err := h.service.repo.CreateTraining(c.Request().Context(), training); err != nil {
		return handleError(c, err)
	}
	
	return c.JSON(http.StatusCreated, training)
}

// ListTraining lists training records
// GET /auth/compliance/organizations/:orgId/training
func (h *Handler) ListTraining(c Context) error {
	orgID := c.Param("orgId")
	
	filters := TrainingFilters{
		OrganizationID: orgID,
		UserID:         c.Query("userId"),
		TrainingType:   c.Query("trainingType"),
		Status:         c.Query("status"),
		Limit:          c.QueryInt("limit", 20),
		Offset:         c.QueryInt("offset", 0),
	}
	
	training, err := h.service.repo.ListTraining(c.Request().Context(), filters)
	if err != nil {
		return handleError(c, err)
	}
	
	return c.JSON(http.StatusOK, training)
}

// GetUserTraining gets training status for a user
// GET /auth/compliance/users/:userId/training
func (h *Handler) GetUserTraining(c Context) error {
	userID := c.Param("userId")
	
	training, err := h.service.repo.GetUserTrainingStatus(c.Request().Context(), userID)
	if err != nil {
		return handleError(c, err)
	}
	
	return c.JSON(http.StatusOK, training)
}

// CompleteTraining marks training as completed
// PUT /auth/compliance/training/:id/complete
func (h *Handler) CompleteTraining(c Context) error {
	id := c.Param("id")
	
	var req struct {
		Score int `json:"score"`
	}
	
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Message: "Invalid request body",
		})
	}
	
	training, err := h.service.repo.GetTraining(c.Request().Context(), id)
	if err != nil {
		return handleError(c, err)
	}
	
	now := time.Now()
	training.Status = "completed"
	training.CompletedAt = &now
	training.Score = req.Score
	
	// Set expiration (1 year from completion)
	expires := now.AddDate(1, 0, 0)
	training.ExpiresAt = &expires
	
	if err := h.service.repo.UpdateTraining(c.Request().Context(), training); err != nil {
		return handleError(c, err)
	}
	
	return c.JSON(http.StatusOK, training)
}

// ===== Template Handlers =====

// ListTemplates lists available compliance templates
// GET /auth/compliance/templates
func (h *Handler) ListTemplates(c Context) error {
	templates := make([]map[string]interface{}, 0)
	
	for standard, template := range ComplianceTemplates {
		templates = append(templates, map[string]interface{}{
			"standard":    standard,
			"name":        template.Name,
			"description": template.Description,
		})
	}
	
	return c.JSON(http.StatusOK, templates)
}

// GetTemplate retrieves a compliance template
// GET /auth/compliance/templates/:standard
func (h *Handler) GetTemplate(c Context) error {
	standard := ComplianceStandard(c.Param("standard"))
	
	template, ok := GetTemplate(standard)
	if !ok {
		return c.JSON(http.StatusNotFound, ErrorResponse{
			Message: "Template not found",
		})
	}
	
	return c.JSON(http.StatusOK, template)
}

// Helper functions

func handleError(c Context, err error) error {
	switch err {
	case ErrProfileNotFound, ErrCheckNotFound, ErrViolationNotFound, 
	     ErrReportNotFound, ErrEvidenceNotFound, ErrPolicyNotFound, ErrTrainingNotFound:
		return c.JSON(http.StatusNotFound, ErrorResponse{
			Message: err.Error(),
		})
	case ErrProfileExists, ErrViolationExists, ErrPolicyExists:
		return c.JSON(http.StatusConflict, ErrorResponse{
			Message: err.Error(),
		})
	case ErrMFARequired, ErrWeakPassword, ErrSessionExpired, ErrAccessDenied, ErrTrainingRequired:
		return c.JSON(http.StatusForbidden, ErrorResponse{
			Message: err.Error(),
		})
	default:
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Message: "Internal server error",
			Error:   err.Error(),
		})
	}
}

// Context interface for HTTP handlers
type Context interface {
	Request() *http.Request
	BindJSON(v interface{}) error
	Param(key string) string
	Query(key string) string
	QueryInt(key string, defaultValue int) int
	Get(key string) interface{}
	JSON(code int, v interface{}) error
	NoContent(code int) error
	Redirect(code int, url string) error
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

