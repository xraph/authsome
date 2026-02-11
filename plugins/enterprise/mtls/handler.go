package mtls

import (
	"encoding/json"
	"encoding/pem"
	"strconv"
	"time"

	"github.com/xraph/authsome/core/responses"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/forge"
)

// Handler handles HTTP requests for mTLS operations.
type Handler struct {
	service *Service
}

// ErrorResponse types - use shared responses from core.
//
//nolint:errname // HTTP response DTO, not a Go error type
type ErrorResponse = responses.ErrorResponse
type MessageResponse = responses.MessageResponse
type StatusResponse = responses.StatusResponse
type SuccessResponse = responses.SuccessResponse

type CertificatesResponse struct {
	Certificates any `json:"certificates"`
	Count        int `json:"count"`
}

type CertificateResponse struct {
	Certificate any `json:"certificate"`
}

type TrustStoresResponse struct {
	TrustStores any `json:"trust_stores"`
	Count       int `json:"count"`
}

// NewHandler creates a new mTLS handler.
func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

// ===== Certificate Management Endpoints =====

// RegisterCertificate registers a new certificate
// RegisterCertificate /auth/mtls/certificates.
func (h *Handler) RegisterCertificate(c forge.Context) error {
	var req RegisterCertificateRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(400, errs.BadRequest("invalid request"))
	}

	cert, err := h.service.RegisterCertificate(c.Request().Context(), &req)
	if err != nil {
		return c.JSON(400, errs.BadRequest(err.Error()))
	}

	return c.JSON(201, cert)
}

// AuthenticateWithCertificate authenticates using client certificate
// AuthenticateWithCertificate /auth/mtls/authenticate.
func (h *Handler) AuthenticateWithCertificate(c forge.Context) error {
	// Get certificate from TLS connection
	if c.Request().TLS == nil || len(c.Request().TLS.PeerCertificates) == 0 {
		return c.JSON(400, errs.BadRequest("no client certificate provided"))
	}

	// Get organization ID from query parameter
	orgID := c.Query("organizationId")
	if orgID == "" {
		return c.JSON(400, errs.RequiredField("organization_id"))
	}

	// Get certificate PEM
	peerCert := c.Request().TLS.PeerCertificates[0]
	certPEM := certToPEM(peerCert.Raw)

	result, err := h.service.AuthenticateWithCertificate(c.Request().Context(), certPEM, orgID)
	if err != nil {
		return c.JSON(500, errs.InternalError(err))
	}

	if !result.Success {
		return c.JSON(401, map[string]any{
			"success": false,
			"errors":  result.Errors,
		})
	}

	return c.JSON(200, map[string]any{
		"success":       true,
		"userId":        result.UserID,
		"certificateId": result.CertificateID,
	})
}

// GetCertificate retrieves a certificate by ID
// GetCertificate /auth/mtls/certificates/:id.
func (h *Handler) GetCertificate(c forge.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(400, errs.RequiredField("certificate_id"))
	}

	cert, err := h.service.GetCertificate(c.Request().Context(), id)
	if err != nil {
		return c.JSON(404, errs.NotFound("certificate not found"))
	}

	return c.JSON(200, cert)
}

// ListCertificates lists certificates with filters
// ListCertificates /auth/mtls/certificates.
func (h *Handler) ListCertificates(c forge.Context) error {
	filters := CertificateFilters{
		OrganizationID: c.Query("organizationId"),
		UserID:         c.Query("userId"),
		DeviceID:       c.Query("deviceId"),
		Status:         c.Query("status"),
		CertType:       c.Query("type"),
	}

	if limit := c.Query("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil {
			filters.Limit = l
		}
	}

	if offset := c.Query("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil {
			filters.Offset = o
		}
	}

	certs, err := h.service.ListCertificates(c.Request().Context(), filters)
	if err != nil {
		return c.JSON(500, errs.InternalError(err))
	}

	return c.JSON(200, map[string]any{
		"certificates": certs,
		"total":        len(certs),
	})
}

// RevokeCertificate revokes a certificate
// RevokeCertificate /auth/mtls/certificates/:id/revoke.
func (h *Handler) RevokeCertificate(c forge.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(400, errs.RequiredField("certificate_id"))
	}

	var req struct {
		Reason string `json:"reason"`
	}
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(400, errs.BadRequest("invalid request"))
	}

	if err := h.service.RevokeCertificate(c.Request().Context(), id, req.Reason); err != nil {
		return c.JSON(500, errs.InternalError(err))
	}

	return c.JSON(200, map[string]any{
		"success": true,
		"message": "certificate revoked",
	})
}

// ===== Trust Anchor Endpoints =====

// AddTrustAnchor adds a new trust anchor
// AddTrustAnchor /auth/mtls/trust-anchors.
func (h *Handler) AddTrustAnchor(c forge.Context) error {
	var req AddTrustAnchorRequest
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(400, errs.BadRequest("invalid request"))
	}

	anchor, err := h.service.AddTrustAnchor(c.Request().Context(), &req)
	if err != nil {
		return c.JSON(400, errs.BadRequest(err.Error()))
	}

	return c.JSON(201, anchor)
}

// GetTrustAnchors lists trust anchors for an organization
// GetTrustAnchors /auth/mtls/trust-anchors.
func (h *Handler) GetTrustAnchors(c forge.Context) error {
	orgID := c.Query("organizationId")
	if orgID == "" {
		return c.JSON(400, errs.RequiredField("organization_id"))
	}

	anchors, err := h.service.GetTrustAnchors(c.Request().Context(), orgID)
	if err != nil {
		return c.JSON(500, errs.InternalError(err))
	}

	return c.JSON(200, map[string]any{
		"trustAnchors": anchors,
		"total":        len(anchors),
	})
}

// ===== Policy Endpoints =====

// CreatePolicy creates a certificate policy
// CreatePolicy /auth/mtls/policies.
func (h *Handler) CreatePolicy(c forge.Context) error {
	var req CreatePolicyRequest
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(400, errs.BadRequest("invalid request"))
	}

	policy, err := h.service.CreatePolicy(c.Request().Context(), &req)
	if err != nil {
		return c.JSON(400, errs.BadRequest(err.Error()))
	}

	return c.JSON(201, policy)
}

// GetPolicy retrieves a policy by ID
// GetPolicy /auth/mtls/policies/:id.
func (h *Handler) GetPolicy(c forge.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(400, errs.RequiredField("policy_id"))
	}

	policy, err := h.service.GetPolicy(c.Request().Context(), id)
	if err != nil {
		return c.JSON(404, errs.NotFound("policy not found"))
	}

	return c.JSON(200, policy)
}

// ===== Statistics Endpoints =====

// GetAuthStats retrieves authentication statistics
// GetAuthStats /auth/mtls/stats/auth.
func (h *Handler) GetAuthStats(c forge.Context) error {
	orgID := c.Query("organizationId")
	if orgID == "" {
		return c.JSON(400, errs.RequiredField("organization_id"))
	}

	// Default to last 30 days
	since := time.Now().AddDate(0, 0, -30)

	if sinceParam := c.Query("since"); sinceParam != "" {
		if parsed, err := time.Parse(time.RFC3339, sinceParam); err == nil {
			since = parsed
		}
	}

	stats, err := h.service.GetAuthEventStats(c.Request().Context(), orgID, since)
	if err != nil {
		return c.JSON(500, errs.InternalError(err))
	}

	return c.JSON(200, stats)
}

// GetExpiringCertificates retrieves certificates expiring soon
// GetExpiringCertificates /auth/mtls/certificates/expiring.
func (h *Handler) GetExpiringCertificates(c forge.Context) error {
	orgID := c.Query("organizationId")
	if orgID == "" {
		return c.JSON(400, errs.RequiredField("organization_id"))
	}

	days := 30 // Default to 30 days

	if daysParam := c.Query("days"); daysParam != "" {
		if d, err := strconv.Atoi(daysParam); err == nil {
			days = d
		}
	}

	certs, err := h.service.GetExpiringCertificates(c.Request().Context(), orgID, days)
	if err != nil {
		return c.JSON(500, errs.InternalError(err))
	}

	return c.JSON(200, map[string]any{
		"certificates": certs,
		"total":        len(certs),
		"days":         days,
	})
}

// ===== Validation Endpoint =====

// ValidateCertificate validates a certificate without authentication
// ValidateCertificate /auth/mtls/validate.
func (h *Handler) ValidateCertificate(c forge.Context) error {
	var req struct {
		CertificatePEM string `json:"certificatePem"`
		OrganizationID string `json:"organizationId"`
	}

	if err := c.BindJSON(&req); err != nil {
		return c.JSON(400, errs.BadRequest("invalid request"))
	}

	result, err := h.service.validator.ValidateCertificate(
		c.Request().Context(),
		[]byte(req.CertificatePEM),
		req.OrganizationID,
	)
	if err != nil {
		return c.JSON(500, errs.InternalError(err))
	}

	return c.JSON(200, map[string]any{
		"valid":            result.Valid,
		"errors":           result.Errors,
		"warnings":         result.Warnings,
		"validationSteps":  result.ValidationSteps,
		"revocationStatus": result.RevocationStatus,
	})
}

// ===== Helper Functions =====

func certToPEM(derBytes []byte) []byte {
	pemBlock := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: derBytes,
	}

	return pem.EncodeToMemory(pemBlock)
}
