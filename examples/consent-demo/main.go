package main

import (
	"fmt"
	"log"

	"github.com/xraph/authsome"
	"github.com/xraph/authsome/plugins/enterprise/consent"
	"github.com/xraph/forge"
)

func main() {
	fmt.Println("Starting Consent Plugin Demo Application...")

	// Create Forge app
	app := forge.New()

	// Create AuthSome instance in SaaS mode for multi-tenancy
	auth := authsome.New(authsome.Config{
		Mode:     authsome.ModeSaaS, // or ModeStandalone
		BasePath: "/api/auth",
		Secret:   "your-secret-key-change-in-production",
		TrustedOrigins: []string{
			"http://localhost:3000",
			"http://localhost:8080",
		},
	})

	// Register consent plugin
	consentPlugin := consent.NewPlugin()
	if err := auth.RegisterPlugin(consentPlugin); err != nil {
		log.Fatalf("Failed to register consent plugin: %v", err)
	}

	// Initialize AuthSome (this will call plugin.Init())
	if err := auth.Init(); err != nil {
		log.Fatalf("Failed to initialize AuthSome: %v", err)
	}

	// Mount AuthSome to Forge app
	auth.Mount(app, "/api/auth")

	// Setup demo routes
	setupDemoRoutes(app, auth, consentPlugin)

	fmt.Println("\n" + "=".repeat(60))
	fmt.Println("Consent Plugin Demo Server Started!")
	fmt.Println("=".repeat(60))
	fmt.Println("\nAPI Endpoints:")
	fmt.Println("  Auth:")
	fmt.Println("    POST   http://localhost:8080/api/auth/signup")
	fmt.Println("    POST   http://localhost:8080/api/auth/signin")
	fmt.Println("    POST   http://localhost:8080/api/auth/signout")
	fmt.Println("\n  Consent Management:")
	fmt.Println("    POST   http://localhost:8080/api/auth/consent/records")
	fmt.Println("    GET    http://localhost:8080/api/auth/consent/records")
	fmt.Println("    POST   http://localhost:8080/api/auth/consent/revoke")
	fmt.Println("    GET    http://localhost:8080/api/auth/consent/summary")
	fmt.Println("\n  Cookie Consent:")
	fmt.Println("    POST   http://localhost:8080/api/auth/consent/cookies")
	fmt.Println("    GET    http://localhost:8080/api/auth/consent/cookies")
	fmt.Println("\n  Data Export (GDPR Article 20):")
	fmt.Println("    POST   http://localhost:8080/api/auth/consent/export")
	fmt.Println("    GET    http://localhost:8080/api/auth/consent/export")
	fmt.Println("    GET    http://localhost:8080/api/auth/consent/export/:id")
	fmt.Println("    GET    http://localhost:8080/api/auth/consent/export/:id/download")
	fmt.Println("\n  Data Deletion (GDPR Article 17):")
	fmt.Println("    POST   http://localhost:8080/api/auth/consent/deletion")
	fmt.Println("    GET    http://localhost:8080/api/auth/consent/deletion")
	fmt.Println("    GET    http://localhost:8080/api/auth/consent/deletion/:id")
	fmt.Println("\n  Demo Endpoints:")
	fmt.Println("    GET    http://localhost:8080/demo")
	fmt.Println("    GET    http://localhost:8080/marketing/subscribe (requires marketing consent)")
	fmt.Println("    GET    http://localhost:8080/analytics/track (requires analytics consent)")
	fmt.Println("=".repeat(60))
	fmt.Println("\nServer listening on :8080")
	fmt.Println("Press Ctrl+C to stop\n")

	// Start server
	if err := app.Start(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func setupDemoRoutes(app *forge.App, auth *authsome.Auth, consentPlugin *consent.Plugin) {
	// Demo home page
	app.GET("/demo", func(c forge.Context) error {
		html := `
<!DOCTYPE html>
<html>
<head>
    <title>Consent Plugin Demo</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 1200px; margin: 0 auto; padding: 20px; }
        h1 { color: #333; }
        .section { margin: 30px 0; padding: 20px; border: 1px solid #ddd; border-radius: 5px; }
        button { padding: 10px 20px; margin: 5px; cursor: pointer; background: #007bff; color: white; border: none; border-radius: 4px; }
        button:hover { background: #0056b3; }
        .response { margin-top: 10px; padding: 10px; background: #f8f9fa; border-radius: 4px; white-space: pre-wrap; }
        input { padding: 8px; margin: 5px; width: 200px; }
    </style>
</head>
<body>
    <h1>🔒 Consent & Privacy Plugin Demo</h1>
    
    <div class="section">
        <h2>1. Authentication</h2>
        <input type="email" id="email" placeholder="email@example.com" value="test@example.com">
        <input type="password" id="password" placeholder="password" value="password123">
        <br>
        <button onclick="signup()">Sign Up</button>
        <button onclick="signin()">Sign In</button>
        <button onclick="signout()">Sign Out</button>
        <div id="auth-response" class="response"></div>
    </div>
    
    <div class="section">
        <h2>2. Grant Marketing Consent</h2>
        <button onclick="grantConsent('marketing', 'email_campaigns')">Grant Marketing Consent</button>
        <button onclick="revokeConsent('marketing', 'email_campaigns')">Revoke Marketing Consent</button>
        <div id="consent-response" class="response"></div>
    </div>
    
    <div class="section">
        <h2>3. Cookie Preferences</h2>
        <label><input type="checkbox" id="functional" checked> Functional</label>
        <label><input type="checkbox" id="analytics"> Analytics</label>
        <label><input type="checkbox" id="marketing"> Marketing</label>
        <br>
        <button onclick="saveCookiePreferences()">Save Cookie Preferences</button>
        <div id="cookie-response" class="response"></div>
    </div>
    
    <div class="section">
        <h2>4. GDPR Data Export (Article 20)</h2>
        <button onclick="requestDataExport()">Request Data Export</button>
        <button onclick="listExports()">List My Exports</button>
        <div id="export-response" class="response"></div>
    </div>
    
    <div class="section">
        <h2>5. GDPR Right to be Forgotten (Article 17)</h2>
        <input type="text" id="deletion-reason" placeholder="Reason for deletion" value="No longer need the account">
        <br>
        <button onclick="requestDeletion()">Request Account Deletion</button>
        <button onclick="listDeletions()">List Deletion Requests</button>
        <div id="deletion-response" class="response"></div>
    </div>
    
    <div class="section">
        <h2>6. Test Consent-Protected Endpoints</h2>
        <button onclick="testMarketingEndpoint()">Test Marketing Endpoint (requires consent)</button>
        <button onclick="testAnalyticsEndpoint()">Test Analytics Endpoint (requires consent)</button>
        <div id="protected-response" class="response"></div>
    </div>
    
    <div class="section">
        <h2>7. View Consent Summary</h2>
        <button onclick="getConsentSummary()">Get My Consent Summary</button>
        <div id="summary-response" class="response"></div>
    </div>

    <script>
        let authToken = localStorage.getItem('authToken') || '';

        async function signup() {
            const email = document.getElementById('email').value;
            const password = document.getElementById('password').value;
            
            const res = await fetch('/api/auth/signup', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ email, password })
            });
            
            const data = await res.json();
            document.getElementById('auth-response').textContent = JSON.stringify(data, null, 2);
            
            if (data.token) {
                authToken = data.token;
                localStorage.setItem('authToken', authToken);
            }
        }

        async function signin() {
            const email = document.getElementById('email').value;
            const password = document.getElementById('password').value;
            
            const res = await fetch('/api/auth/signin', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ email, password })
            });
            
            const data = await res.json();
            document.getElementById('auth-response').textContent = JSON.stringify(data, null, 2);
            
            if (data.token) {
                authToken = data.token;
                localStorage.setItem('authToken', authToken);
            }
        }

        async function signout() {
            await fetch('/api/auth/signout', {
                method: 'POST',
                headers: { 'Authorization': 'Bearer ' + authToken }
            });
            authToken = '';
            localStorage.removeItem('authToken');
            document.getElementById('auth-response').textContent = 'Signed out';
        }

        async function grantConsent(type, purpose) {
            const res = await fetch('/api/auth/consent/records', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': 'Bearer ' + authToken
                },
                body: JSON.stringify({
                    consentType: type,
                    purpose: purpose,
                    granted: true,
                    version: '1.0'
                })
            });
            
            const data = await res.json();
            document.getElementById('consent-response').textContent = JSON.stringify(data, null, 2);
        }

        async function revokeConsent(type, purpose) {
            const res = await fetch('/api/auth/consent/revoke', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': 'Bearer ' + authToken
                },
                body: JSON.stringify({
                    consentType: type,
                    purpose: purpose
                })
            });
            
            const data = await res.json();
            document.getElementById('consent-response').textContent = JSON.stringify(data, null, 2);
        }

        async function saveCookiePreferences() {
            const res = await fetch('/api/auth/consent/cookies', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': 'Bearer ' + authToken
                },
                body: JSON.stringify({
                    essential: true,
                    functional: document.getElementById('functional').checked,
                    analytics: document.getElementById('analytics').checked,
                    marketing: document.getElementById('marketing').checked,
                    personalization: false,
                    thirdParty: false,
                    bannerVersion: '1.0'
                })
            });
            
            const data = await res.json();
            document.getElementById('cookie-response').textContent = JSON.stringify(data, null, 2);
        }

        async function requestDataExport() {
            const res = await fetch('/api/auth/consent/export', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': 'Bearer ' + authToken
                },
                body: JSON.stringify({
                    format: 'json',
                    includeSections: ['profile', 'consents', 'audit']
                })
            });
            
            const data = await res.json();
            document.getElementById('export-response').textContent = JSON.stringify(data, null, 2);
        }

        async function listExports() {
            const res = await fetch('/api/auth/consent/export', {
                headers: { 'Authorization': 'Bearer ' + authToken }
            });
            
            const data = await res.json();
            document.getElementById('export-response').textContent = JSON.stringify(data, null, 2);
        }

        async function requestDeletion() {
            const reason = document.getElementById('deletion-reason').value;
            
            const res = await fetch('/api/auth/consent/deletion', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': 'Bearer ' + authToken
                },
                body: JSON.stringify({
                    reason: reason,
                    deleteSections: ['all']
                })
            });
            
            const data = await res.json();
            document.getElementById('deletion-response').textContent = JSON.stringify(data, null, 2);
        }

        async function listDeletions() {
            const res = await fetch('/api/auth/consent/deletion', {
                headers: { 'Authorization': 'Bearer ' + authToken }
            });
            
            const data = await res.json();
            document.getElementById('deletion-response').textContent = JSON.stringify(data, null, 2);
        }

        async function testMarketingEndpoint() {
            const res = await fetch('/marketing/subscribe', {
                headers: { 'Authorization': 'Bearer ' + authToken }
            });
            
            const data = await res.json();
            document.getElementById('protected-response').textContent = JSON.stringify(data, null, 2);
        }

        async function testAnalyticsEndpoint() {
            const res = await fetch('/analytics/track', {
                headers: { 'Authorization': 'Bearer ' + authToken }
            });
            
            const data = await res.json();
            document.getElementById('protected-response').textContent = JSON.stringify(data, null, 2);
        }

        async function getConsentSummary() {
            const res = await fetch('/api/auth/consent/summary', {
                headers: { 'Authorization': 'Bearer ' + authToken }
            });
            
            const data = await res.json();
            document.getElementById('summary-response').textContent = JSON.stringify(data, null, 2);
        }

        // Load auth token on page load
        if (authToken) {
            document.getElementById('auth-response').textContent = 'Authenticated (token in localStorage)';
        }
    </script>
</body>
</html>
`
		return c.HTML(200, html)
	})

	// Marketing endpoint protected by consent
	app.GET("/marketing/subscribe",
		consentPlugin.RequireConsent("marketing", "email_campaigns")(
			func(c forge.Context) error {
				return c.JSON(200, map[string]string{
					"message":    "✅ You have access to marketing features!",
					"subscribed": "You are now subscribed to our newsletter",
				})
			},
		),
	)

	// Analytics endpoint protected by consent
	app.GET("/analytics/track",
		consentPlugin.RequireConsent("analytics", "usage_tracking")(
			func(c forge.Context) error {
				return c.JSON(200, map[string]string{
					"message": "✅ Analytics tracking enabled",
					"tracked": "Your usage is being tracked",
				})
			},
		),
	)
}
