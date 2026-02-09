package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

// DeviceAuthorizationResponse matches the server's response
type DeviceAuthorizationResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

// TokenResponse matches the OAuth token response
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	IDToken      string `json:"id_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

// ErrorResponse matches OAuth error responses
type ErrorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description,omitempty"`
}

func main() {
	// Parse command-line flags
	authServerURL := flag.String("server", "http://localhost:3001", "Auth server base URL")
	clientID := flag.String("client", "", "OAuth client ID (required)")
	scope := flag.String("scope", "openid profile email", "OAuth scope")
	flag.Parse()

	if *clientID == "" {
		fmt.Println("Error: --client flag is required")
		flag.Usage()
		os.Exit(1)
	}

	fmt.Println("🔐 Device Flow Authentication Demo")
	fmt.Println("====================================")

	// Step 1: Initiate device authorization
	fmt.Println("Step 1: Requesting device code...")
	deviceAuth, err := initiateDeviceAuthorization(*authServerURL, *clientID, *scope)
	if err != nil {
		fmt.Printf("❌ Failed to initiate device authorization: %v\n", err)
		os.Exit(1)
	}

	// Step 2: Display user code to user
	fmt.Println("\n✅ Device authorization initiated successfully!")
	fmt.Println("\n" + "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("📱 Please visit: %s\n", deviceAuth.VerificationURI)
	fmt.Printf("🔢 Enter code: %s\n", deviceAuth.UserCode)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("⏱️  Code expires in %d seconds\n", deviceAuth.ExpiresIn)
	fmt.Printf("🔄 Polling interval: %d seconds\n\n", deviceAuth.Interval)

	// Display QR-friendly URL
	fmt.Printf("Direct link: %s\n\n", deviceAuth.VerificationURIComplete)

	// Step 3: Poll for token
	fmt.Println("Step 3: Waiting for authorization...")
	fmt.Println("(Polling every", deviceAuth.Interval, "seconds...)")

	tokens, err := pollForToken(*authServerURL, *clientID, deviceAuth.DeviceCode, deviceAuth.Interval, deviceAuth.ExpiresIn)
	if err != nil {
		fmt.Printf("❌ Failed to get token: %v\n", err)
		os.Exit(1)
	}

	// Step 4: Display tokens
	fmt.Println("\n✅ Authorization successful!")
	fmt.Println("\n" + "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("🎉 Tokens Received:")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("Token Type: %s\n", tokens.TokenType)
	fmt.Printf("Expires In: %d seconds\n", tokens.ExpiresIn)
	fmt.Printf("Scope: %s\n\n", tokens.Scope)

	fmt.Printf("Access Token: %s\n", truncate(tokens.AccessToken, 50))
	if tokens.RefreshToken != "" {
		fmt.Printf("Refresh Token: %s\n", truncate(tokens.RefreshToken, 50))
	}
	if tokens.IDToken != "" {
		fmt.Printf("ID Token: %s\n", truncate(tokens.IDToken, 50))
	}
	fmt.Println("\n✨ You can now use these tokens to access protected resources!")
}

// initiateDeviceAuthorization requests a device code from the auth server
func initiateDeviceAuthorization(baseURL, clientID, scope string) (*DeviceAuthorizationResponse, error) {
	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("scope", scope)

	resp, err := http.Post(
		baseURL+"/oauth2/device/authorize",
		"application/x-www-form-urlencoded",
		bytes.NewBufferString(data.Encode()),
	)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned %d: %s", resp.StatusCode, string(body))
	}

	var result DeviceAuthorizationResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// pollForToken polls the token endpoint until authorization is complete
func pollForToken(baseURL, clientID, deviceCode string, interval, expiresIn int) (*TokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")
	data.Set("device_code", deviceCode)
	data.Set("client_id", clientID)

	timeout := time.Now().Add(time.Duration(expiresIn) * time.Second)
	pollInterval := time.Duration(interval) * time.Second
	attempts := 0

	for time.Now().Before(timeout) {
		attempts++
		fmt.Printf("  [%d] Polling... ", attempts)

		resp, err := http.Post(
			baseURL+"/oauth2/token",
			"application/x-www-form-urlencoded",
			bytes.NewBufferString(data.Encode()),
		)
		if err != nil {
			fmt.Printf("❌ Request failed: %v\n", err)
			time.Sleep(pollInterval)
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			// Success! Parse tokens
			var tokens TokenResponse
			if err := json.Unmarshal(body, &tokens); err != nil {
				return nil, fmt.Errorf("failed to parse token response: %w", err)
			}
			fmt.Println("✅ Authorized!")
			return &tokens, nil
		}

		// Parse error response
		var errResp ErrorResponse
		if err := json.Unmarshal(body, &errResp); err != nil {
			return nil, fmt.Errorf("failed to parse error response: %w", err)
		}

		switch errResp.Error {
		case "authorization_pending":
			fmt.Println("⏳ Pending")
		case "slow_down":
			fmt.Println("🐌 Slow down")
			pollInterval += 5 * time.Second // Add 5 seconds
		case "expired_token":
			return nil, fmt.Errorf("device code expired")
		case "access_denied":
			return nil, fmt.Errorf("user denied authorization")
		default:
			return nil, fmt.Errorf("error: %s - %s", errResp.Error, errResp.ErrorDescription)
		}

		time.Sleep(pollInterval)
	}

	return nil, fmt.Errorf("authorization timeout")
}

// truncate truncates a string to the specified length
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
