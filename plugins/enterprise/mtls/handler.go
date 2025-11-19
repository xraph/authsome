package mtls

import (
	"encoding/json"
	"encoding/pem"
	"strconv"
	"time"

	"github.com/xraph/forge"
)

// Handler handles HTTP requests for mTLS operations
type Handler struct {
	service *Service
}

// Response types
type ErrorResponse struct {
	Error string `json:"error"`
}

type MessageResponse struct {
	Message string `json:"message"`
}

type StatusResponse struct {
	Status string `json:"status"`
}

type SuccessResponse struct {
	Success bool `json:"success"`
}



type CertificatesResponse struct {
	Certificates interface{} `json:"certificates"`
	Count        int         `json:"count"`
}

type CertificateResponse struct {
	Certificate interface{} `json:"certificate"`
}

type TrustStoresResponse struct {
	TrustStores interface{} `json:"trust_stores"`
	Count       int         `json:"count"`
}

// NewHandler creates a new mTLS handler
func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

// ===== Certificate Management Endpoints =====

// RegisterCertificate registers a new certificate
// POST /auth/mtls/certificates
func (h *Handler) RegisterCertificate(c forge.Context) error {
	var req RegisterCertificateRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(400, &ErrorResponse{Error: "invalid request",})
	}

	cert, err := h.service.RegisterCertificate(c.Request().Context(), &req)
	if err != nil {
		return c.JSON(400, &ErrorResponse{Error: err.Error(),})
	}

	return c.JSON(201, cert)
}

// AuthenticateWithCertificate authenticates using client certificate
// POST /auth/mtls/authenticate
func (h *Handler) AuthenticateWithCertificate(c forge.Context) error {
	// Get certificate from TLS connection
	if c.Request().TLS == nil || len(c.Request().TLS.PeerCertificates) == 0 {
		return c.JSON(400, &ErrorResponse{Error: "no client certificate provided",})
	}

	// Get organization ID from query parameter
	orgID := c.Query("organizationId")
	if orgID == "" {
		return c.JSON(400, &ErrorResponse{Error: "organization ID required",})
	}

	// Get certificate PEM
	peerCert := c.Request().TLS.PeerCertificates[0]
	certPEM := certToPEM(peerCert.Raw)

	result, err := h.service.AuthenticateWithCertificate(c.Request().Context(), certPEM, orgID)
	if err != nil {
		return c.JSON(500, &ErrorResponse{Error: err.Error(),})
	}

	if !result.Success {
		return c.JSON(401, map[string]interface{}{
			"success": false,
			"errors":  result.Errors,
		})
	}

	return c.JSON(200, map[string]interface{}{
		"success":       true,
		"userId":        result.UserID,
		"certificateId": result.CertificateID,
	})
}

// GetCertificate retrieves a certificate by ID
// GET /auth/mtls/certificates/:id
func (h *Handler) GetCertificate(c forge.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(400, &ErrorResponse{Error: "certificate ID required",})
	}

	cert, err := h.service.GetCertificate(c.Request().Context(), id)
	if err != nil {
		return c.JSON(404, &ErrorResponse{Error: "certificate not found",})
	}

	return c.JSON(200, cert)
}

// ListCertificates lists certificates with filters
// GET /auth/mtls/certificates
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
		return c.JSON(500, &ErrorResponse{Error: err.Error(),})
	}

	return c.JSON(200, map[string]interface{}{
		"certificates": certs,
		"total":        len(certs),
	})
}

// RevokeCertificate revokes a certificate
// POST /auth/mtls/certificates/:id/revoke
func (h *Handler) RevokeCertificate(c forge.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(400, &ErrorResponse{Error: "certificate ID required",})
	}

	var req struct {
		Reason string `json:"reason"`
	}
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(400, &ErrorResponse{Error: "invalid request",})
	}

	if err := h.service.RevokeCertificate(c.Request().Context(), id, req.Reason); err != nil {
		return c.JSON(500, &ErrorResponse{Error: err.Error(),})
	}

	return c.JSON(200, map[string]interface{}{
		"success": true,
		"message": "certificate revoked",
	})
}

// ===== Trust Anchor Endpoints =====

// AddTrustAnchor adds a new trust anchor
// POST /auth/mtls/trust-anchors
func (h *Handler) AddTrustAnchor(c forge.Context) error {
	var req AddTrustAnchorRequest
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(400, &ErrorResponse{Error: "invalid request",})
	}

	anchor, err := h.service.AddTrustAnchor(c.Request().Context(), &req)
	if err != nil {
		return c.JSON(400, &ErrorResponse{Error: err.Error(),})
	}

	return c.JSON(201, anchor)
}

// GetTrustAnchors lists trust anchors for an organization
// GET /auth/mtls/trust-anchors
func (h *Handler) GetTrustAnchors(c forge.Context) error {
	orgID := c.Query("organizationId")
	if orgID == "" {
		return c.JSON(400, &ErrorResponse{Error: "organization ID required",})
	}

	anchors, err := h.service.GetTrustAnchors(c.Request().Context(), orgID)
	if err != nil {
		return c.JSON(500, &ErrorResponse{Error: err.Error(),})
	}

	return c.JSON(200, map[string]interface{}{
		"trustAnchors": anchors,
		"total":        len(anchors),
	})
}

// ===== Policy Endpoints =====

// CreatePolicy creates a certificate policy
// POST /auth/mtls/policies
func (h *Handler) CreatePolicy(c forge.Context) error {
	var req CreatePolicyRequest
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(400, &ErrorResponse{Error: "invalid request",})
	}

	policy, err := h.service.CreatePolicy(c.Request().Context(), &req)
	if err != nil {
		return c.JSON(400, &ErrorResponse{Error: err.Error(),})
	}

	return c.JSON(201, policy)
}

// GetPolicy retrieves a policy by ID
// GET /auth/mtls/policies/:id
func (h *Handler) GetPolicy(c forge.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(400, &ErrorResponse{Error: "policy ID required",})
	}

	policy, err := h.service.GetPolicy(c.Request().Context(), id)
	if err != nil {
		return c.JSON(404, &ErrorResponse{Error: "policy not found",})
	}

	return c.JSON(200, policy)
}

// ===== Statistics Endpoints =====

// GetAuthStats retrieves authentication statistics
// GET /auth/mtls/stats/auth
func (h *Handler) GetAuthStats(c forge.Context) error {
	orgID := c.Query("organizationId")
	if orgID == "" {
		return c.JSON(400, &ErrorResponse{Error: "organization ID required",})
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
		return c.JSON(500, &ErrorResponse{Error: err.Error(),})
	}

	return c.JSON(200, stats)
}

// GetExpiringCertificates retrieves certificates expiring soon
// GET /auth/mtls/certificates/expiring
func (h *Handler) GetExpiringCertificates(c forge.Context) error {
	orgID := c.Query("organizationId")
	if orgID == "" {
		return c.JSON(400, &ErrorResponse{Error: "organization ID required",})
	}

	days := 30 // Default to 30 days
	if daysParam := c.Query("days"); daysParam != "" {
		if d, err := strconv.Atoi(daysParam); err == nil {
			days = d
		}
	}

	certs, err := h.service.GetExpiringCertificates(c.Request().Context(), orgID, days)
	if err != nil {
		return c.JSON(500, &ErrorResponse{Error: err.Error(),})
	}

	return c.JSON(200, map[string]interface{}{
		"certificates": certs,
		"total":        len(certs),
		"days":         days,
	})
}

// ===== Validation Endpoint =====

// ValidateCertificate validates a certificate without authentication
// POST /auth/mtls/validate
func (h *Handler) ValidateCertificate(c forge.Context) error {
	var req struct {
		CertificatePEM string `json:"certificatePem"`
		OrganizationID string `json:"organizationId"`
	}

	if err := c.BindJSON(&req); err != nil {
		return c.JSON(400, &ErrorResponse{Error: "invalid request",})
	}

	result, err := h.service.validator.ValidateCertificate(
		c.Request().Context(),
		[]byte(req.CertificatePEM),
		req.OrganizationID,
	)

	if err != nil {
		return c.JSON(500, &ErrorResponse{Error: err.Error(),})
	}

	return c.JSON(200, map[string]interface{}{
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
