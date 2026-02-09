package oidcprovider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/internal/errs"
)

// ConsentManager handles OAuth consent with optional integration to enterprise consent plugin.
type ConsentManager struct {
	consentSvc        *ConsentService          // Internal OIDC consent
	enterpriseConsent EnterpriseConsentService // Optional enterprise plugin
}

// EnterpriseConsentService interface for enterprise consent plugin
// This allows optional integration without hard dependency.
type EnterpriseConsentService interface {
	// CreateConsent records user consent
	CreateConsent(ctx context.Context, orgID, userID string, req interface{}) (interface{}, error)

	// GetConsent retrieves consent status
	GetConsent(ctx context.Context, id string) (interface{}, error)

	// RevokeConsent revokes a consent
	RevokeConsent(ctx context.Context, id string) error
}

// NewConsentManager creates a consent manager with optional enterprise integration.
func NewConsentManager(consentSvc *ConsentService, enterpriseConsent EnterpriseConsentService) *ConsentManager {
	return &ConsentManager{
		consentSvc:        consentSvc,
		enterpriseConsent: enterpriseConsent,
	}
}

// CheckConsent checks if user has granted consent for the client and scopes.
func (cm *ConsentManager) CheckConsent(ctx context.Context, userID xid.ID, clientID, scope string, appID, envID xid.ID, orgID *xid.ID) (bool, error) {
	// Parse scopes
	scopes := cm.consentSvc.ParseScopes(scope)

	// Check internal OIDC consent
	hasConsent, err := cm.consentSvc.CheckConsent(ctx, userID, clientID, scopes, appID, envID, orgID)
	if err != nil {
		return false, err
	}

	return hasConsent, nil
}

// RecordConsent records user's consent decision.
func (cm *ConsentManager) RecordConsent(ctx context.Context, userID xid.ID, clientID, scope string, granted bool, appID, envID xid.ID, orgID *xid.ID, expiresAt *time.Time) error {
	if !granted {
		// User denied consent - nothing to record
		return nil
	}

	// Parse scopes
	scopes := cm.consentSvc.ParseScopes(scope)

	// Calculate expiration duration
	var expiresIn *time.Duration

	if expiresAt != nil {
		duration := time.Until(*expiresAt)
		expiresIn = &duration
	}

	// Record in internal OIDC consent
	err := cm.consentSvc.GrantConsent(ctx, userID, clientID, scopes, appID, envID, orgID, expiresIn)
	if err != nil {
		return err
	}

	// If enterprise consent plugin is available, record there too for audit trail
	if cm.enterpriseConsent != nil {
		orgID := "default" // TODO: Extract from context

		// Create consent request for enterprise plugin
		// Note: This uses map[string]interface{} to avoid hard dependency
		consentReq := map[string]interface{}{
			"userId":      userID.String(),
			"consentType": "oauth_authorization",
			"purpose":     "OAuth client: " + clientID,
			"granted":     granted,
			"version":     "1.0",
			"metadata": map[string]interface{}{
				"client_id": clientID,
				"scope":     scope,
			},
		}

		if expiresAt != nil {
			days := int(time.Until(*expiresAt).Hours() / 24)
			consentReq["expiresIn"] = days
		}

		_, err := cm.enterpriseConsent.CreateConsent(ctx, orgID, userID.String(), consentReq)
		if err != nil {
			// Log error but don't fail - enterprise consent is optional
			// TODO: Add proper logging
			_ = err
		}
	}

	return nil
}

// RevokeConsent revokes user's consent for a client.
func (cm *ConsentManager) RevokeConsent(ctx context.Context, userID xid.ID, clientID string) error {
	// Revoke internal OIDC consent
	err := cm.consentSvc.RevokeConsent(ctx, userID, clientID)
	if err != nil {
		return err
	}

	// If enterprise consent plugin is available, revoke there too
	if cm.enterpriseConsent != nil {
		// Note: Would need to track consent ID from enterprise plugin
		// For now, we just handle internal consent
	}

	return nil
}

// GenerateConsentHTML generates HTML for the OAuth consent screen.
func (cm *ConsentManager) GenerateConsentHTML(clientName, clientLogoURI, scope string, redirectURL string) string {
	// Parse scopes
	scopes := strings.Split(scope, " ")
	scopeDescriptions := map[string]string{
		"openid":         "Authenticate your identity",
		"profile":        "Access your basic profile information (name, username)",
		"email":          "Access your email address",
		"phone":          "Access your phone number",
		"address":        "Access your postal address",
		"offline_access": "Maintain access when you're offline (refresh tokens)",
		"api:read":       "Read data from APIs",
		"api:write":      "Write data to APIs",
	}

	scopeHTML := ""
	var scopeHTMLSb144 strings.Builder

	for _, s := range scopes {
		description := scopeDescriptions[s]
		if description == "" {
			description = s // Fallback to scope name
		}

		scopeHTMLSb144.WriteString(fmt.Sprintf(`<li class="scope-item"><span class="scope-icon">✓</span> %s</li>`, description))
	}
	scopeHTML += scopeHTMLSb144.String()

	// If no logo, use placeholder
	logoHTML := ""
	if clientLogoURI != "" {
		logoHTML = fmt.Sprintf(`<img src="%s" alt="%s logo" class="client-logo">`, clientLogoURI, clientName)
	} else {
		logoHTML = `<div class="client-logo-placeholder">🔐</div>`
	}

	html := fmt.Sprintf(`
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Authorize %s</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
            background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 20px;
        }
        .consent-container {
            background: white;
            border-radius: 12px;
            box-shadow: 0 20px 60px rgba(0,0,0,0.3);
            max-width: 500px;
            width: 100%%;
            overflow: hidden;
        }
        .consent-header {
            background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);
            color: white;
            padding: 30px;
            text-align: center;
        }
        .client-logo {
            width: 80px;
            height: 80px;
            border-radius: 50%%;
            margin-bottom: 15px;
            background: white;
            padding: 10px;
        }
        .client-logo-placeholder {
            width: 80px;
            height: 80px;
            border-radius: 50%%;
            margin: 0 auto 15px;
            background: rgba(255,255,255,0.2);
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 40px;
        }
        .consent-header h1 {
            font-size: 24px;
            font-weight: 600;
            margin-bottom: 8px;
        }
        .consent-header p {
            font-size: 14px;
            opacity: 0.9;
        }
        .consent-body {
            padding: 30px;
        }
        .consent-message {
            text-align: center;
            margin-bottom: 30px;
            color: #4a5568;
            font-size: 15px;
            line-height: 1.6;
        }
        .scopes-section {
            margin-bottom: 30px;
        }
        .scopes-section h3 {
            font-size: 14px;
            font-weight: 600;
            color: #2d3748;
            margin-bottom: 15px;
            text-transform: uppercase;
            letter-spacing: 0.5px;
        }
        .scopes-list {
            list-style: none;
            background: #f7fafc;
            border-radius: 8px;
            padding: 20px;
        }
        .scope-item {
            padding: 12px 0;
            border-bottom: 1px solid #e2e8f0;
            display: flex;
            align-items: center;
            color: #2d3748;
            font-size: 14px;
        }
        .scope-item:last-child {
            border-bottom: none;
        }
        .scope-icon {
            color: #48bb78;
            font-weight: bold;
            margin-right: 12px;
            font-size: 16px;
        }
        .action-buttons {
            display: flex;
            gap: 15px;
            margin-top: 30px;
        }
        .btn {
            flex: 1;
            padding: 14px;
            border: none;
            border-radius: 8px;
            font-size: 15px;
            font-weight: 600;
            cursor: pointer;
            transition: all 0.2s;
            text-decoration: none;
            display: inline-block;
            text-align: center;
        }
        .btn-allow {
            background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);
            color: white;
        }
        .btn-allow:hover {
            transform: translateY(-2px);
            box-shadow: 0 10px 20px rgba(102, 126, 234, 0.3);
        }
        .btn-deny {
            background: #e2e8f0;
            color: #4a5568;
        }
        .btn-deny:hover {
            background: #cbd5e0;
        }
        .security-notice {
            margin-top: 20px;
            padding: 15px;
            background: #f0fdf4;
            border-left: 3px solid #48bb78;
            border-radius: 6px;
            font-size: 13px;
            color: #22543d;
        }
        .security-notice strong {
            display: block;
            margin-bottom: 5px;
        }
    </style>
</head>
<body>
    <div class="consent-container">
        <div class="consent-header">
            %s
            <h1>%s</h1>
            <p>wants to access your account</p>
        </div>
        <div class="consent-body">
            <div class="consent-message">
                By authorizing this application, you allow <strong>%s</strong> to access the following information:
            </div>
            <div class="scopes-section">
                <h3>Permissions Requested</h3>
                <ul class="scopes-list">
                    %s
                </ul>
            </div>
            <form method="POST" action="%s">
                <div class="action-buttons">
                    <button type="submit" name="consent" value="deny" class="btn btn-deny">Deny</button>
                    <button type="submit" name="consent" value="allow" class="btn btn-allow">Allow Access</button>
                </div>
            </form>
            <div class="security-notice">
                <strong>🔒 Your data is secure</strong>
                You can revoke access at any time from your account settings.
            </div>
        </div>
    </div>
</body>
</html>
`, clientName, logoHTML, clientName, clientName, scopeHTML, redirectURL)

	return html
}

// GetConsentPageData returns data for rendering custom consent templates.
func (cm *ConsentManager) GetConsentPageData(clientName, clientLogoURI, clientDescription, scope string) map[string]interface{} {
	scopes := strings.Split(scope, " ")
	scopeDescriptions := []map[string]string{}

	scopeMap := map[string]string{
		"openid":         "Authenticate your identity",
		"profile":        "Access your basic profile information",
		"email":          "Access your email address",
		"phone":          "Access your phone number",
		"address":        "Access your postal address",
		"offline_access": "Maintain access when you're offline",
	}

	for _, s := range scopes {
		description := scopeMap[s]
		if description == "" {
			description = s
		}

		scopeDescriptions = append(scopeDescriptions, map[string]string{
			"scope":       s,
			"description": description,
		})
	}

	return map[string]interface{}{
		"client_name":        clientName,
		"client_logo_uri":    clientLogoURI,
		"client_description": clientDescription,
		"scopes":             scopeDescriptions,
	}
}

// ValidateConsentRequest validates the consent decision from user.
func (cm *ConsentManager) ValidateConsentRequest(consentDecision string) error {
	if consentDecision != "allow" && consentDecision != "deny" {
		return errs.BadRequest("invalid consent decision")
	}

	return nil
}
