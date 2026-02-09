package compliance

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/core/responses"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/forge"
)

// Handler handles HTTP requests for compliance endpoints.
type Handler struct {
	service      *Service
	policyEngine *PolicyEngine
}

// Response types - use shared responses from core.
type ErrorResponse = responses.ErrorResponse
type MessageResponse = responses.MessageResponse
type StatusResponse = responses.StatusResponse
type SuccessResponse = responses.SuccessResponse

// NewHandler creates a new compliance handler.
func NewHandler(service *Service, policyEngine *PolicyEngine) *Handler {
	return &Handler{
		service:      service,
		policyEngine: policyEngine,
	}
}

// ===== Profile Handlers =====

// CreateProfile creates a new compliance profile
// POST /auth/compliance/profiles.
func (h *Handler) CreateProfile(c forge.Context) error {
	ctx := c.Request().Context()

	// Get appID from context
	appID, ok := contexts.GetAppID(ctx)
	if !ok || appID.IsNil() {
		return c.JSON(http.StatusForbidden, errs.New(
			"APP_CONTEXT_REQUIRED",
			"App context required",
			http.StatusForbidden,
		))
	}

	var req CreateProfileRequest
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("Invalid request body"))
	}

	// Set appID from context
	req.AppID = appID.String()

	profile, err := h.service.CreateProfile(ctx, &req)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(http.StatusCreated, profile)
}

// CreateProfileFromTemplate creates a profile from a template
// POST /auth/compliance/profiles/from-template.
func (h *Handler) CreateProfileFromTemplate(c forge.Context) error {
	ctx := c.Request().Context()

	// Get appID from context
	appID, ok := contexts.GetAppID(ctx)
	if !ok || appID.IsNil() {
		return c.JSON(http.StatusForbidden, errs.New(
			"APP_CONTEXT_REQUIRED",
			"App context required",
			http.StatusForbidden,
		))
	}

	var req struct {
		Standard ComplianceStandard `json:"standard" validate:"required"`
	}

	if err := c.BindJSON(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("Invalid request body"))
	}

	profile, err := h.service.CreateProfileFromTemplate(ctx, appID.String(), req.Standard)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(http.StatusCreated, profile)
}

// GetProfile retrieves a compliance profile
// GET /auth/compliance/profiles/:id.
func (h *Handler) GetProfile(c forge.Context) error {
	id := c.Param("id")

	profile, err := h.service.GetProfile(c.Request().Context(), id)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(http.StatusOK, profile)
}

// GetAppProfile retrieves the compliance profile for an app
// GET /auth/compliance/apps/:appId/profile.
func (h *Handler) GetAppProfile(c forge.Context) error {
	ctx := c.Request().Context()

	// Get appID from context
	appID, ok := contexts.GetAppID(ctx)
	if !ok || appID.IsNil() {
		return c.JSON(http.StatusForbidden, errs.New(
			"APP_CONTEXT_REQUIRED",
			"App context required",
			http.StatusForbidden,
		))
	}

	profile, err := h.service.GetProfileByApp(ctx, appID.String())
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(http.StatusOK, profile)
}

// UpdateProfile updates a compliance profile
// PUT /auth/compliance/profiles/:id.
func (h *Handler) UpdateProfile(c forge.Context) error {
	id := c.Param("id")

	var req UpdateProfileRequest
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("Invalid request body"))
	}

	profile, err := h.service.UpdateProfile(c.Request().Context(), id, &req)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(http.StatusOK, profile)
}

// DeleteProfile deletes a compliance profile
// DELETE /auth/compliance/profiles/:id.
func (h *Handler) DeleteProfile(c forge.Context) error {
	id := c.Param("id")

	if err := h.service.repo.DeleteProfile(c.Request().Context(), id); err != nil {
		return handleError(c, err)
	}

	return c.NoContent(http.StatusNoContent)
}

// ===== Status & Dashboard Handlers =====

// GetComplianceStatus gets overall compliance status for an app
// GET /auth/compliance/apps/:appId/status.
func (h *Handler) GetComplianceStatus(c forge.Context) error {
	ctx := c.Request().Context()

	// Get appID from context
	appID, ok := contexts.GetAppID(ctx)
	if !ok || appID.IsNil() {
		return c.JSON(http.StatusForbidden, errs.New(
			"APP_CONTEXT_REQUIRED",
			"App context required",
			http.StatusForbidden,
		))
	}

	status, err := h.service.GetComplianceStatus(ctx, appID.String())
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(http.StatusOK, status)
}

// GetDashboard gets compliance dashboard data
// GET /auth/compliance/apps/:appId/dashboard.
func (h *Handler) GetDashboard(c forge.Context) error {
	ctx := c.Request().Context()

	// Get appID from context
	appID, ok := contexts.GetAppID(ctx)
	if !ok || appID.IsNil() {
		return c.JSON(http.StatusForbidden, errs.New(
			"APP_CONTEXT_REQUIRED",
			"App context required",
			http.StatusForbidden,
		))
	}

	appIDStr := appID.String()

	// Get profile
	profile, err := h.service.GetProfileByApp(ctx, appIDStr)
	if err != nil {
		return handleError(c, err)
	}

	// Get status
	status, _ := h.service.GetComplianceStatus(ctx, appIDStr)

	// Get recent checks
	profileIDFilter := profile.ID
	statusOpen := "open"
	checksFilter := &ListChecksFilter{
		PaginationParams: pagination.PaginationParams{Limit: 10, Offset: 0},
		ProfileID:        &profileIDFilter,
	}
	checksResp, _ := h.service.ListChecks(ctx, checksFilter)
	checks := checksResp.Data

	// Get open violations
	violationsFilter := &ListViolationsFilter{
		PaginationParams: pagination.PaginationParams{Limit: 10, Offset: 0},
		AppID:            &appIDStr,
		Status:           &statusOpen,
	}
	violationsResp, _ := h.service.ListViolations(ctx, violationsFilter)
	violations := violationsResp.Data

	// Get recent reports
	reportsFilter := &ListReportsFilter{
		PaginationParams: pagination.PaginationParams{Limit: 5, Offset: 0},
		AppID:            &appIDStr,
	}
	reportsResp, _ := h.service.ListReports(ctx, reportsFilter)
	reports := reportsResp.Data

	dashboard := map[string]any{
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
// POST /auth/compliance/profiles/:profileId/checks.
func (h *Handler) RunCheck(c forge.Context) error {
	profileID := c.Param("profileId")

	var req struct {
		CheckType string `json:"checkType" validate:"required"`
	}

	if err := c.BindJSON(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("Invalid request body"))
	}

	check, err := h.service.RunCheck(c.Request().Context(), profileID, req.CheckType)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(http.StatusCreated, check)
}

// ListChecks lists compliance checks
// GET /auth/compliance/profiles/:profileId/checks.
func (h *Handler) ListChecks(c forge.Context) error {
	ctx := c.Request().Context()
	q := c.Request().URL.Query()

	// Parse pagination
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit == 0 {
		limit = 50
	}

	offset, _ := strconv.Atoi(q.Get("offset"))

	// Build filter
	filter := &ListChecksFilter{
		PaginationParams: pagination.PaginationParams{
			Limit:  limit,
			Offset: offset,
		},
	}

	// Parse optional filters
	if profileID := c.Param("profileId"); profileID != "" {
		filter.ProfileID = &profileID
	}

	if appID := q.Get("appId"); appID != "" {
		filter.AppID = &appID
	}

	if checkType := q.Get("checkType"); checkType != "" {
		filter.CheckType = &checkType
	}

	if status := q.Get("status"); status != "" {
		filter.Status = &status
	}

	resp, err := h.service.ListChecks(ctx, filter)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(http.StatusOK, resp)
}

// GetCheck retrieves a compliance check
// GET /auth/compliance/checks/:id.
func (h *Handler) GetCheck(c forge.Context) error {
	id := c.Param("id")

	check, err := h.service.repo.GetCheck(c.Request().Context(), id)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(http.StatusOK, check)
}

// ===== Violation Handlers =====

// ListViolations lists compliance violations
// GET /auth/compliance/apps/:appId/violations.
func (h *Handler) ListViolations(c forge.Context) error {
	ctx := c.Request().Context()

	// Get appID from context
	appID, ok := contexts.GetAppID(ctx)
	if !ok || appID.IsNil() {
		return c.JSON(http.StatusForbidden, errs.New(
			"APP_CONTEXT_REQUIRED",
			"App context required",
			http.StatusForbidden,
		))
	}

	appIDStr := appID.String()

	q := c.Request().URL.Query()

	// Parse pagination
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit == 0 {
		limit = 50
	}

	offset, _ := strconv.Atoi(q.Get("offset"))

	// Build filter
	filter := &ListViolationsFilter{
		PaginationParams: pagination.PaginationParams{
			Limit:  limit,
			Offset: offset,
		},
		AppID: &appIDStr,
	}

	// Parse optional filters
	if profileID := q.Get("profileId"); profileID != "" {
		filter.ProfileID = &profileID
	}

	if userID := q.Get("userId"); userID != "" {
		filter.UserID = &userID
	}

	if violationType := q.Get("violationType"); violationType != "" {
		filter.ViolationType = &violationType
	}

	if severity := q.Get("severity"); severity != "" {
		filter.Severity = &severity
	}

	if status := q.Get("status"); status != "" {
		filter.Status = &status
	}

	resp, err := h.service.ListViolations(ctx, filter)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(http.StatusOK, resp)
}

// GetViolation retrieves a compliance violation
// GET /auth/compliance/violations/:id.
func (h *Handler) GetViolation(c forge.Context) error {
	id := c.Param("id")

	violation, err := h.service.repo.GetViolation(c.Request().Context(), id)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(http.StatusOK, violation)
}

// ResolveViolation resolves a compliance violation
// PUT /auth/compliance/violations/:id/resolve.
func (h *Handler) ResolveViolation(c forge.Context) error {
	ctx := c.Request().Context()
	id := c.Param("id")

	// Get userID from context
	userID, ok := contexts.GetUserID(ctx)
	if !ok || userID.IsNil() {
		return c.JSON(http.StatusForbidden, errs.New(
			"USER_CONTEXT_REQUIRED",
			"User context required",
			http.StatusForbidden,
		))
	}

	if err := h.service.repo.ResolveViolation(ctx, id, userID.String()); err != nil {
		return handleError(c, err)
	}

	violation, _ := h.service.repo.GetViolation(ctx, id)

	return c.JSON(http.StatusOK, violation)
}

// ===== Report Handlers =====

// GenerateReport generates a compliance report
// POST /auth/compliance/apps/:appId/reports.
func (h *Handler) GenerateReport(c forge.Context) error {
	ctx := c.Request().Context()

	// Get appID from context
	appID, ok := contexts.GetAppID(ctx)
	if !ok || appID.IsNil() {
		return c.JSON(http.StatusForbidden, errs.New(
			"APP_CONTEXT_REQUIRED",
			"App context required",
			http.StatusForbidden,
		))
	}

	var req struct {
		ReportType string             `json:"reportType" validate:"required"`
		Standard   ComplianceStandard `json:"standard"`
		Period     string             `json:"period"     validate:"required"`
		Format     string             `json:"format"     validate:"required"`
	}

	if err := c.BindJSON(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("Invalid request body"))
	}

	// Get userID from context
	userID, _ := contexts.GetUserID(ctx)

	// Create report record
	report := &ComplianceReport{
		AppID:       appID.String(),
		ReportType:  req.ReportType,
		Standard:    req.Standard,
		Period:      req.Period,
		Format:      req.Format,
		Status:      "generating",
		GeneratedBy: userID.String(),
	}

	if err := h.service.repo.CreateReport(ctx, report); err != nil {
		return handleError(c, err)
	}

	// Generate report asynchronously
	go h.generateReportAsync(report)

	return c.JSON(http.StatusAccepted, report)
}

// ListReports lists compliance reports
// GET /auth/compliance/apps/:appId/reports.
func (h *Handler) ListReports(c forge.Context) error {
	ctx := c.Request().Context()

	// Get appID from context
	appID, ok := contexts.GetAppID(ctx)
	if !ok || appID.IsNil() {
		return c.JSON(http.StatusForbidden, errs.New(
			"APP_CONTEXT_REQUIRED",
			"App context required",
			http.StatusForbidden,
		))
	}

	appIDStr := appID.String()

	q := c.Request().URL.Query()

	// Parse pagination
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit == 0 {
		limit = 50
	}

	offset, _ := strconv.Atoi(q.Get("offset"))

	// Build filter
	filter := &ListReportsFilter{
		PaginationParams: pagination.PaginationParams{
			Limit:  limit,
			Offset: offset,
		},
		AppID: &appIDStr,
	}

	// Parse optional filters
	if profileID := q.Get("profileId"); profileID != "" {
		filter.ProfileID = &profileID
	}

	if reportType := q.Get("reportType"); reportType != "" {
		filter.ReportType = &reportType
	}

	if status := q.Get("status"); status != "" {
		filter.Status = &status
	}

	if format := q.Get("format"); format != "" {
		filter.Format = &format
	}

	resp, err := h.service.ListReports(ctx, filter)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(http.StatusOK, resp)
}

// GetReport retrieves a compliance report
// GET /auth/compliance/reports/:id.
func (h *Handler) GetReport(c forge.Context) error {
	id := c.Param("id")

	report, err := h.service.repo.GetReport(c.Request().Context(), id)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(http.StatusOK, report)
}

// DownloadReport downloads a compliance report file
// GET /auth/compliance/reports/:id/download.
func (h *Handler) DownloadReport(c forge.Context) error {
	id := c.Param("id")

	report, err := h.service.repo.GetReport(c.Request().Context(), id)
	if err != nil {
		return handleError(c, err)
	}

	if report.Status != "ready" {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("Report is not ready for download"))
	}

	// In real implementation, this would stream the file from storage
	return c.Redirect(http.StatusFound, report.FileURL)
}

// generateReportAsync generates report asynchronously.
func (h *Handler) generateReportAsync(report *ComplianceReport) {
	// This would be implemented with proper report generation logic
	// For now, it's a placeholder
}

// ===== Evidence Handlers =====

// CreateEvidence creates compliance evidence
// POST /auth/compliance/apps/:appId/evidence.
func (h *Handler) CreateEvidence(c forge.Context) error {
	ctx := c.Request().Context()

	// Get appID from context
	appID, ok := contexts.GetAppID(ctx)
	if !ok || appID.IsNil() {
		return c.JSON(http.StatusForbidden, errs.New(
			"APP_CONTEXT_REQUIRED",
			"App context required",
			http.StatusForbidden,
		))
	}

	var req struct {
		EvidenceType string             `json:"evidenceType" validate:"required"`
		Standard     ComplianceStandard `json:"standard"`
		ControlID    string             `json:"controlId"`
		Title        string             `json:"title"        validate:"required"`
		Description  string             `json:"description"`
		FileURL      string             `json:"fileUrl"`
	}

	if err := c.BindJSON(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("Invalid request body"))
	}

	// Get profile
	profile, err := h.service.GetProfileByApp(ctx, appID.String())
	if err != nil {
		return handleError(c, err)
	}

	// Get userID from context
	userID, _ := contexts.GetUserID(ctx)

	evidence := &ComplianceEvidence{
		ProfileID:    profile.ID,
		AppID:        appID.String(),
		EvidenceType: req.EvidenceType,
		Standard:     req.Standard,
		ControlID:    req.ControlID,
		Title:        req.Title,
		Description:  req.Description,
		FileURL:      req.FileURL,
		CollectedBy:  userID.String(),
	}

	if err := h.service.repo.CreateEvidence(ctx, evidence); err != nil {
		return handleError(c, err)
	}

	return c.JSON(http.StatusCreated, evidence)
}

// ListEvidence lists compliance evidence
// GET /auth/compliance/apps/:appId/evidence.
func (h *Handler) ListEvidence(c forge.Context) error {
	ctx := c.Request().Context()

	// Get appID from context
	appID, ok := contexts.GetAppID(ctx)
	if !ok || appID.IsNil() {
		return c.JSON(http.StatusForbidden, errs.New(
			"APP_CONTEXT_REQUIRED",
			"App context required",
			http.StatusForbidden,
		))
	}

	appIDStr := appID.String()

	q := c.Request().URL.Query()

	// Parse pagination
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit == 0 {
		limit = 50
	}

	offset, _ := strconv.Atoi(q.Get("offset"))

	// Build filter
	filter := &ListEvidenceFilter{
		PaginationParams: pagination.PaginationParams{
			Limit:  limit,
			Offset: offset,
		},
		AppID: &appIDStr,
	}

	// Parse optional filters
	if profileID := q.Get("profileId"); profileID != "" {
		filter.ProfileID = &profileID
	}

	if evidenceType := q.Get("evidenceType"); evidenceType != "" {
		filter.EvidenceType = &evidenceType
	}

	if controlID := q.Get("controlId"); controlID != "" {
		filter.ControlID = &controlID
	}

	resp, err := h.service.ListEvidence(ctx, filter)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(http.StatusOK, resp)
}

// GetEvidence retrieves compliance evidence
// GET /auth/compliance/evidence/:id.
func (h *Handler) GetEvidence(c forge.Context) error {
	id := c.Param("id")

	evidence, err := h.service.repo.GetEvidence(c.Request().Context(), id)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(http.StatusOK, evidence)
}

// DeleteEvidence deletes compliance evidence
// DELETE /auth/compliance/evidence/:id.
func (h *Handler) DeleteEvidence(c forge.Context) error {
	id := c.Param("id")

	if err := h.service.repo.DeleteEvidence(c.Request().Context(), id); err != nil {
		return handleError(c, err)
	}

	return c.NoContent(http.StatusNoContent)
}

// ===== Policy Handlers =====

// CreatePolicy creates a compliance policy
// POST /auth/compliance/apps/:appId/policies.
func (h *Handler) CreatePolicy(c forge.Context) error {
	ctx := c.Request().Context()

	// Get appID from context
	appID, ok := contexts.GetAppID(ctx)
	if !ok || appID.IsNil() {
		return c.JSON(http.StatusForbidden, errs.New(
			"APP_CONTEXT_REQUIRED",
			"App context required",
			http.StatusForbidden,
		))
	}

	var req struct {
		PolicyType string             `json:"policyType" validate:"required"`
		Standard   ComplianceStandard `json:"standard"`
		Title      string             `json:"title"      validate:"required"`
		Version    string             `json:"version"    validate:"required"`
		Content    string             `json:"content"    validate:"required"`
	}

	if err := c.BindJSON(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("Invalid request body"))
	}

	// Get profile
	profile, err := h.service.GetProfileByApp(ctx, appID.String())
	if err != nil {
		return handleError(c, err)
	}

	policy := &CompliancePolicy{
		ProfileID:  profile.ID,
		AppID:      appID.String(),
		PolicyType: req.PolicyType,
		Standard:   req.Standard,
		Title:      req.Title,
		Version:    req.Version,
		Content:    req.Content,
		Status:     "draft",
	}

	if err := h.service.repo.CreatePolicy(ctx, policy); err != nil {
		return handleError(c, err)
	}

	return c.JSON(http.StatusCreated, policy)
}

// ListPolicies lists compliance policies
// GET /auth/compliance/apps/:appId/policies.
func (h *Handler) ListPolicies(c forge.Context) error {
	ctx := c.Request().Context()

	// Get appID from context
	appID, ok := contexts.GetAppID(ctx)
	if !ok || appID.IsNil() {
		return c.JSON(http.StatusForbidden, errs.New(
			"APP_CONTEXT_REQUIRED",
			"App context required",
			http.StatusForbidden,
		))
	}

	appIDStr := appID.String()

	q := c.Request().URL.Query()

	// Parse pagination
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit == 0 {
		limit = 50
	}

	offset, _ := strconv.Atoi(q.Get("offset"))

	// Build filter
	filter := &ListPoliciesFilter{
		PaginationParams: pagination.PaginationParams{
			Limit:  limit,
			Offset: offset,
		},
		AppID: &appIDStr,
	}

	// Parse optional filters
	if profileID := q.Get("profileId"); profileID != "" {
		filter.ProfileID = &profileID
	}

	if policyType := q.Get("policyType"); policyType != "" {
		filter.PolicyType = &policyType
	}

	if status := q.Get("status"); status != "" {
		filter.Status = &status
	}

	resp, err := h.service.ListPolicies(ctx, filter)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(http.StatusOK, resp)
}

// GetPolicy retrieves a compliance policy
// GET /auth/compliance/policies/:id.
func (h *Handler) GetPolicy(c forge.Context) error {
	id := c.Param("id")

	policy, err := h.service.repo.GetPolicy(c.Request().Context(), id)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(http.StatusOK, policy)
}

// UpdatePolicy updates a compliance policy
// PUT /auth/compliance/policies/:id.
func (h *Handler) UpdatePolicy(c forge.Context) error {
	id := c.Param("id")

	var req struct {
		Title   *string `json:"title"`
		Version *string `json:"version"`
		Content *string `json:"content"`
		Status  *string `json:"status"`
	}

	if err := c.BindJSON(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("Invalid request body"))
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
// DELETE /auth/compliance/policies/:id.
func (h *Handler) DeletePolicy(c forge.Context) error {
	id := c.Param("id")

	if err := h.service.repo.DeletePolicy(c.Request().Context(), id); err != nil {
		return handleError(c, err)
	}

	return c.NoContent(http.StatusNoContent)
}

// ===== Training Handlers =====

// CreateTraining creates a training record
// POST /auth/compliance/apps/:appId/training.
func (h *Handler) CreateTraining(c forge.Context) error {
	ctx := c.Request().Context()

	// Get appID from context
	appID, ok := contexts.GetAppID(ctx)
	if !ok || appID.IsNil() {
		return c.JSON(http.StatusForbidden, errs.New(
			"APP_CONTEXT_REQUIRED",
			"App context required",
			http.StatusForbidden,
		))
	}

	var req struct {
		UserID       string             `json:"userId"       validate:"required"`
		TrainingType string             `json:"trainingType" validate:"required"`
		Standard     ComplianceStandard `json:"standard"`
	}

	if err := c.BindJSON(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("Invalid request body"))
	}

	// Get profile
	profile, err := h.service.GetProfileByApp(ctx, appID.String())
	if err != nil {
		return handleError(c, err)
	}

	training := &ComplianceTraining{
		ProfileID:    profile.ID,
		AppID:        appID.String(),
		UserID:       req.UserID,
		TrainingType: req.TrainingType,
		Standard:     req.Standard,
		Status:       "required",
	}

	if err := h.service.repo.CreateTraining(ctx, training); err != nil {
		return handleError(c, err)
	}

	return c.JSON(http.StatusCreated, training)
}

// ListTraining lists training records
// GET /auth/compliance/apps/:appId/training.
func (h *Handler) ListTraining(c forge.Context) error {
	ctx := c.Request().Context()

	// Get appID from context
	appID, ok := contexts.GetAppID(ctx)
	if !ok || appID.IsNil() {
		return c.JSON(http.StatusForbidden, errs.New(
			"APP_CONTEXT_REQUIRED",
			"App context required",
			http.StatusForbidden,
		))
	}

	appIDStr := appID.String()

	q := c.Request().URL.Query()

	// Parse pagination
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit == 0 {
		limit = 50
	}

	offset, _ := strconv.Atoi(q.Get("offset"))

	// Build filter
	filter := &ListTrainingFilter{
		PaginationParams: pagination.PaginationParams{
			Limit:  limit,
			Offset: offset,
		},
		AppID: &appIDStr,
	}

	// Parse optional filters
	if profileID := q.Get("profileId"); profileID != "" {
		filter.ProfileID = &profileID
	}

	if userID := q.Get("userId"); userID != "" {
		filter.UserID = &userID
	}

	if trainingType := q.Get("trainingType"); trainingType != "" {
		filter.TrainingType = &trainingType
	}

	if status := q.Get("status"); status != "" {
		filter.Status = &status
	}

	resp, err := h.service.ListTraining(ctx, filter)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(http.StatusOK, resp)
}

// GetUserTraining gets training status for a user
// GET /auth/compliance/users/:userId/training.
func (h *Handler) GetUserTraining(c forge.Context) error {
	userID := c.Param("userId")

	training, err := h.service.repo.GetUserTrainingStatus(c.Request().Context(), userID)
	if err != nil {
		return handleError(c, err)
	}

	return c.JSON(http.StatusOK, training)
}

// CompleteTraining marks training as completed
// PUT /auth/compliance/training/:id/complete.
func (h *Handler) CompleteTraining(c forge.Context) error {
	id := c.Param("id")

	var req struct {
		Score int `json:"score"`
	}

	if err := c.BindJSON(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("Invalid request body"))
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
// GET /auth/compliance/templates.
func (h *Handler) ListTemplates(c forge.Context) error {
	templates := make([]map[string]any, 0)

	for standard, template := range ComplianceTemplates {
		templates = append(templates, map[string]any{
			"standard":    standard,
			"name":        template.Name,
			"description": template.Description,
		})
	}

	return c.JSON(http.StatusOK, templates)
}

// GetTemplate retrieves a compliance template
// GET /auth/compliance/templates/:standard.
func (h *Handler) GetTemplate(c forge.Context) error {
	standard := ComplianceStandard(c.Param("standard"))

	template, ok := GetTemplate(standard)
	if !ok {
		return c.JSON(http.StatusNotFound, errs.NotFound("Template not found"))
	}

	return c.JSON(http.StatusOK, template)
}

// Helper functions

func handleError(c forge.Context, err error) error {
	// Handle structured AuthsomeError
	authErr := &errs.AuthsomeError{}
	if errors.As(err, &authErr) {
		return c.JSON(authErr.HTTPStatus, authErr)
	}

	// Fallback for unexpected errors
	return c.JSON(http.StatusInternalServerError, errs.InternalServerErrorWithMessage("Internal server error"))
}
