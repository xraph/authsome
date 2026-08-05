package mfa_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	log "github.com/xraph/go-utils/log"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/plugin"
	"github.com/xraph/authsome/plugins/mfa"
	"github.com/xraph/authsome/user"
)

func newTestPlugin(t *testing.T) (*mfa.Plugin, *mfa.MemoryStore) {
	t.Helper()
	p := mfa.New(mfa.Config{
		Issuer: "TestApp",
	})
	s := mfa.NewMemoryStore()
	p.SetStore(s)
	return p, s
}

func jsonBody(t *testing.T, v any) *bytes.Buffer {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return bytes.NewBuffer(b)
}

func authedRequest(t *testing.T, method, path string, body *bytes.Buffer, userID id.UserID) *http.Request {
	t.Helper()
	var req *http.Request
	if body != nil {
		req = httptest.NewRequestWithContext(context.Background(), method, path, body)
	} else {
		req = httptest.NewRequestWithContext(context.Background(), method, path, nil)
	}
	req.Header.Set("Content-Type", "application/json")
	ctx := middleware.WithUserID(req.Context(), userID)
	return req.WithContext(ctx)
}

func authedRequestWithUser(t *testing.T, method, path string, body *bytes.Buffer, u *user.User) *http.Request {
	t.Helper()
	req := authedRequest(t, method, path, body, u.ID)
	ctx := middleware.WithUser(req.Context(), u)
	return req.WithContext(ctx)
}

// ──────────────────────────────────────────────────
// Unit tests
// ──────────────────────────────────────────────────

func TestPlugin_Name(t *testing.T) {
	p := mfa.New(mfa.Config{})
	assert.Equal(t, "mfa", p.Name())
}

func TestPlugin_DefaultIssuer(t *testing.T) {
	p := mfa.New(mfa.Config{})
	assert.Equal(t, "mfa", p.Name())
}

func TestPlugin_ImplementsInterfaces(t *testing.T) { //nolint:revive // test function signature
	p := mfa.New(mfa.Config{})

	var _ plugin.Plugin = p
	var _ plugin.RouteProvider = p
	var _ plugin.OnInit = p
}

func TestPlugin_RegisterInRegistry(t *testing.T) {
	reg := plugin.NewRegistry(log.NewNoopLogger())
	p := mfa.New(mfa.Config{})
	reg.Register(p)

	assert.Len(t, reg.Plugins(), 1)
	assert.Equal(t, "mfa", reg.Plugins()[0].Name())
	assert.Len(t, reg.RouteProviders(), 1)
}

// ──────────────────────────────────────────────────
// TOTP generation and validation
// ──────────────────────────────────────────────────

func TestGenerateTOTPKey(t *testing.T) {
	key, err := mfa.GenerateTOTPKey(mfa.TOTPConfig{
		Issuer:      "TestApp",
		AccountName: "user@example.com",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, key.Secret())
	assert.Contains(t, key.URL(), "TestApp")
	assert.Contains(t, key.URL(), "user@example.com")
}

func TestValidateTOTP(t *testing.T) {
	key, err := mfa.GenerateTOTPKey(mfa.TOTPConfig{
		Issuer:      "TestApp",
		AccountName: "user@example.com",
	})
	require.NoError(t, err)

	// Generate a valid code
	code, err := mfa.GenerateTOTPCode(key.Secret())
	require.NoError(t, err)

	// Validate
	assert.True(t, mfa.ValidateTOTP(code, key.Secret()))

	// Invalid code
	assert.False(t, mfa.ValidateTOTP("000000", key.Secret()))
}

// ──────────────────────────────────────────────────
// Memory store tests
// ──────────────────────────────────────────────────

func TestMemoryStore_CRUD(t *testing.T) {
	s := mfa.NewMemoryStore()
	ctx := context.Background()
	userID := id.NewUserID()

	enrollment := &mfa.Enrollment{
		ID:        id.NewMFAID(),
		UserID:    userID,
		Method:    "totp",
		Secret:    "TESTSECRET",
		Verified:  false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Create
	err := s.CreateEnrollment(ctx, enrollment)
	require.NoError(t, err)

	// Get by user+method
	got, err := s.GetEnrollment(ctx, userID, "totp")
	require.NoError(t, err)
	assert.Equal(t, enrollment.ID, got.ID)
	assert.Equal(t, "TESTSECRET", got.Secret)

	// Get by ID
	got2, err := s.GetEnrollmentByID(ctx, enrollment.ID)
	require.NoError(t, err)
	assert.Equal(t, enrollment.ID, got2.ID)

	// List
	list, err := s.ListEnrollments(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, list, 1)

	// Update
	enrollment.Verified = true
	err = s.UpdateEnrollment(ctx, enrollment)
	require.NoError(t, err)

	got3, err := s.GetEnrollment(ctx, userID, "totp")
	require.NoError(t, err)
	assert.True(t, got3.Verified)

	// Delete
	err = s.DeleteEnrollment(ctx, enrollment.ID)
	require.NoError(t, err)

	_, err = s.GetEnrollment(ctx, userID, "totp")
	assert.ErrorIs(t, err, mfa.ErrEnrollmentNotFound)

	// Delete nonexistent
	err = s.DeleteEnrollment(ctx, id.NewMFAID())
	assert.ErrorIs(t, err, mfa.ErrEnrollmentNotFound)

	// Update nonexistent
	err = s.UpdateEnrollment(ctx, enrollment)
	assert.ErrorIs(t, err, mfa.ErrEnrollmentNotFound)
}

// ──────────────────────────────────────────────────
// Enroll endpoint tests
// ──────────────────────────────────────────────────

func TestHandleEnroll_Success(t *testing.T) {
	p, _ := newTestPlugin(t)

	mux := forge.NewRouter()
	err := p.RegisterRoutes(mux)
	require.NoError(t, err)

	userID := id.NewUserID()
	u := &user.User{ID: userID, Email: "user@example.com"}
	body := jsonBody(t, map[string]string{"method": "totp"})
	req := authedRequestWithUser(t, "POST", "/v1/mfa/enroll", body, u)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]any
	err = json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	assert.NotEmpty(t, resp["id"])
	assert.Equal(t, "totp", resp["method"])
	assert.NotEmpty(t, resp["secret"])
	assert.NotEmpty(t, resp["otpauth_url"])
	assert.Contains(t, resp["otpauth_url"].(string), "TestApp")
}

func TestHandleEnroll_DefaultMethod(t *testing.T) {
	p, _ := newTestPlugin(t)

	mux := forge.NewRouter()
	err := p.RegisterRoutes(mux)
	require.NoError(t, err)

	userID := id.NewUserID()
	body := jsonBody(t, map[string]string{})
	req := authedRequest(t, "POST", "/v1/mfa/enroll", body, userID)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestHandleEnroll_UnsupportedMethod(t *testing.T) {
	p, _ := newTestPlugin(t)

	mux := forge.NewRouter()
	err := p.RegisterRoutes(mux)
	require.NoError(t, err)

	userID := id.NewUserID()
	body := jsonBody(t, map[string]string{"method": "sms"})
	req := authedRequest(t, "POST", "/v1/mfa/enroll", body, userID)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandleEnroll_Unauthenticated(t *testing.T) {
	p, _ := newTestPlugin(t)

	mux := forge.NewRouter()
	err := p.RegisterRoutes(mux)
	require.NoError(t, err)

	body := jsonBody(t, map[string]string{"method": "totp"})
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/v1/mfa/enroll", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestHandleEnroll_AlreadyVerified(t *testing.T) {
	p, s := newTestPlugin(t)

	mux := forge.NewRouter()
	err := p.RegisterRoutes(mux)
	require.NoError(t, err)

	userID := id.NewUserID()

	// Pre-create a verified enrollment
	enrollment := &mfa.Enrollment{
		ID:        id.NewMFAID(),
		UserID:    userID,
		Method:    "totp",
		Secret:    "EXISTINGSECRET",
		Verified:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err = s.CreateEnrollment(context.Background(), enrollment)
	require.NoError(t, err)

	body := jsonBody(t, map[string]string{"method": "totp"})
	req := authedRequest(t, "POST", "/v1/mfa/enroll", body, userID)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusConflict, rec.Code)
}

// ──────────────────────────────────────────────────
// Verify endpoint tests
// ──────────────────────────────────────────────────

func TestHandleVerify_Success(t *testing.T) {
	p, s := newTestPlugin(t)

	mux := forge.NewRouter()
	err := p.RegisterRoutes(mux)
	require.NoError(t, err)

	userID := id.NewUserID()

	// Generate a real TOTP key
	key, err := mfa.GenerateTOTPKey(mfa.TOTPConfig{Issuer: "TestApp", AccountName: "user@test.com"})
	require.NoError(t, err)

	enrollment := &mfa.Enrollment{
		ID:        id.NewMFAID(),
		UserID:    userID,
		Method:    "totp",
		Secret:    key.Secret(),
		Verified:  false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err = s.CreateEnrollment(context.Background(), enrollment)
	require.NoError(t, err)

	// Generate valid TOTP code
	code, err := mfa.GenerateTOTPCode(key.Secret())
	require.NoError(t, err)

	body := jsonBody(t, map[string]string{"code": code})
	req := authedRequest(t, "POST", "/v1/mfa/verify", body, userID)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]any
	err = json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, true, resp["verified"])

	// Enrollment should now be verified
	got, err := s.GetEnrollment(context.Background(), userID, "totp")
	require.NoError(t, err)
	assert.True(t, got.Verified)
}

func TestHandleVerify_InvalidCode(t *testing.T) {
	p, s := newTestPlugin(t)

	mux := forge.NewRouter()
	err := p.RegisterRoutes(mux)
	require.NoError(t, err)

	userID := id.NewUserID()

	key, err := mfa.GenerateTOTPKey(mfa.TOTPConfig{Issuer: "TestApp", AccountName: "user@test.com"})
	require.NoError(t, err)

	enrollment := &mfa.Enrollment{
		ID:        id.NewMFAID(),
		UserID:    userID,
		Method:    "totp",
		Secret:    key.Secret(),
		Verified:  false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err = s.CreateEnrollment(context.Background(), enrollment)
	require.NoError(t, err)

	body := jsonBody(t, map[string]string{"code": "000000"})
	req := authedRequest(t, "POST", "/v1/mfa/verify", body, userID)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestHandleVerify_MissingCode(t *testing.T) {
	p, _ := newTestPlugin(t)

	mux := forge.NewRouter()
	err := p.RegisterRoutes(mux)
	require.NoError(t, err)

	userID := id.NewUserID()
	body := jsonBody(t, map[string]string{})
	req := authedRequest(t, "POST", "/v1/mfa/verify", body, userID)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandleVerify_NoEnrollment(t *testing.T) {
	p, _ := newTestPlugin(t)

	mux := forge.NewRouter()
	err := p.RegisterRoutes(mux)
	require.NoError(t, err)

	userID := id.NewUserID()
	body := jsonBody(t, map[string]string{"code": "123456"})
	req := authedRequest(t, "POST", "/v1/mfa/verify", body, userID)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestHandleVerify_Unauthenticated(t *testing.T) {
	p, _ := newTestPlugin(t)

	mux := forge.NewRouter()
	err := p.RegisterRoutes(mux)
	require.NoError(t, err)

	body := jsonBody(t, map[string]string{"code": "123456"})
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/v1/mfa/verify", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// ──────────────────────────────────────────────────
// Challenge endpoint tests
// ──────────────────────────────────────────────────

// TestHandleChallenge_RequiresTicket pins the new ticket-based
// contract for /v1/mfa/challenge: missing the mfa_ticket field
// returns 400 unconditionally — even with a valid code and an
// enrolled user — because the endpoint is no longer a step-up
// auth surface for already-signed-in users; it's the second leg
// of the sign-in MFA gate. The full happy-path round-trip
// (sign-in → ticket → challenge → session) is exercised in
// api/api_test.go where the real engine is wired.
func TestHandleChallenge_RequiresTicket(t *testing.T) {
	p, _ := newTestPlugin(t)
	mux := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(mux))

	body := jsonBody(t, map[string]string{"code": "123456"})
	req, _ := http.NewRequestWithContext(context.Background(), "POST", "/v1/mfa/challenge", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code,
		"missing mfa_ticket must 400 before any TOTP work happens; body=%s", rec.Body.String())
}

// ──────────────────────────────────────────────────
// Disable endpoint tests
// ──────────────────────────────────────────────────

func TestHandleDisable_Success(t *testing.T) {
	p, s := newTestPlugin(t)

	mux := forge.NewRouter()
	err := p.RegisterRoutes(mux)
	require.NoError(t, err)

	userID := id.NewUserID()

	key, err := mfa.GenerateTOTPKey(mfa.TOTPConfig{Issuer: "Test", AccountName: "u@example.com"})
	require.NoError(t, err)

	enrollment := &mfa.Enrollment{
		ID:        id.NewMFAID(),
		UserID:    userID,
		Method:    "totp",
		Secret:    key.Secret(),
		Verified:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err = s.CreateEnrollment(context.Background(), enrollment)
	require.NoError(t, err)

	// Disabling now requires a fresh proof of possession.
	code, err := mfa.GenerateTOTPCode(key.Secret())
	require.NoError(t, err)
	req := authedRequest(t, "DELETE", "/v1/mfa/enrollment", jsonBody(t, map[string]string{"code": code}), userID)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

	// Should be gone
	_, err = s.GetEnrollment(context.Background(), userID, "totp")
	assert.ErrorIs(t, err, mfa.ErrEnrollmentNotFound)
}

func TestHandleDisable_NoEnrollment(t *testing.T) {
	p, _ := newTestPlugin(t)

	mux := forge.NewRouter()
	err := p.RegisterRoutes(mux)
	require.NoError(t, err)

	userID := id.NewUserID()
	req := authedRequest(t, "DELETE", "/v1/mfa/enrollment", nil, userID)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestHandleDisable_Unauthenticated(t *testing.T) {
	p, _ := newTestPlugin(t)

	mux := forge.NewRouter()
	err := p.RegisterRoutes(mux)
	require.NoError(t, err)

	req := httptest.NewRequestWithContext(context.Background(), "DELETE", "/v1/mfa/enrollment", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// ──────────────────────────────────────────────────
// HasMFA helper
// ──────────────────────────────────────────────────

func TestHasMFA(t *testing.T) {
	p, s := newTestPlugin(t)
	ctx := context.Background()
	userID := id.NewUserID()

	// No enrollment
	assert.False(t, p.HasMFA(ctx, userID))

	// Unverified enrollment
	enrollment := &mfa.Enrollment{
		ID:        id.NewMFAID(),
		UserID:    userID,
		Method:    "totp",
		Secret:    "SECRET",
		Verified:  false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err := s.CreateEnrollment(ctx, enrollment)
	require.NoError(t, err)
	assert.False(t, p.HasMFA(ctx, userID))

	// Verified enrollment
	enrollment.Verified = true
	err = s.UpdateEnrollment(ctx, enrollment)
	require.NoError(t, err)
	assert.True(t, p.HasMFA(ctx, userID))
}

// ──────────────────────────────────────────────────
// Full flow: enroll → verify → challenge → disable
// ──────────────────────────────────────────────────

func TestFullFlow_EnrollVerifyChallengeDisable(t *testing.T) {
	p, _ := newTestPlugin(t)

	mux := forge.NewRouter()
	err := p.RegisterRoutes(mux)
	require.NoError(t, err)

	userID := id.NewUserID()
	u := &user.User{ID: userID, Email: "flow@example.com"}

	// Step 1: Enroll
	enrollBody := jsonBody(t, map[string]string{"method": "totp"})
	enrollReq := authedRequestWithUser(t, "POST", "/v1/mfa/enroll", enrollBody, u)
	enrollRec := httptest.NewRecorder()
	mux.ServeHTTP(enrollRec, enrollReq)
	require.Equal(t, http.StatusOK, enrollRec.Code)

	var enrollResp map[string]any
	err = json.NewDecoder(enrollRec.Body).Decode(&enrollResp)
	require.NoError(t, err)
	secret := enrollResp["secret"].(string)

	// Step 2: Verify with valid TOTP code
	code, err := mfa.GenerateTOTPCode(secret)
	require.NoError(t, err)

	verifyBody := jsonBody(t, map[string]string{"code": code})
	verifyReq := authedRequest(t, "POST", "/v1/mfa/verify", verifyBody, userID)
	verifyRec := httptest.NewRecorder()
	mux.ServeHTTP(verifyRec, verifyReq)
	require.Equal(t, http.StatusOK, verifyRec.Code)

	// Step 3: Challenge step skipped at the plugin level — the new
	// ticket-based contract requires a real *authsome.Engine and is
	// covered end-to-end in api/api_test.go's TestSignIn_MFARequired*
	// tests where a full engine is wired. Here we just confirm the
	// route is wired and rejects an unticketed call.
	noTicketBody := jsonBody(t, map[string]string{"code": "000000"})
	noTicketReq, _ := http.NewRequestWithContext(context.Background(), "POST", "/v1/mfa/challenge", noTicketBody)
	noTicketReq.Header.Set("Content-Type", "application/json")
	noTicketRec := httptest.NewRecorder()
	mux.ServeHTTP(noTicketRec, noTicketReq)
	require.Equal(t, http.StatusBadRequest, noTicketRec.Code,
		"challenge without mfa_ticket must 400; body=%s", noTicketRec.Body.String())

	// Step 4: Disable. Requires a fresh proof of possession — a code from a
	// later time-step, since the one used at step 2 is spent for replay
	// purposes.
	disableCode, err := mfa.GenerateTOTPCodeAt(secret, time.Now().Add(30*time.Second))
	require.NoError(t, err)
	disableBody := jsonBody(t, map[string]string{"code": disableCode})
	disableReq := authedRequest(t, "DELETE", "/v1/mfa/enrollment", disableBody, userID)
	disableRec := httptest.NewRecorder()
	mux.ServeHTTP(disableRec, disableReq)
	assert.Equal(t, http.StatusOK, disableRec.Code, "body: %s", disableRec.Body.String())
}

// ──────────────────────────────────────────────────
// Recovery code unit tests
// ──────────────────────────────────────────────────

func TestGenerateRecoveryCodes(t *testing.T) {
	userID := id.NewUserID()
	codes, plaintexts, err := mfa.GenerateRecoveryCodes(userID, 8)
	require.NoError(t, err)
	assert.Len(t, codes, 8)
	assert.Len(t, plaintexts, 8)

	// Each plaintext is 8 chars, each code has a hash
	for i, pt := range plaintexts {
		assert.Len(t, pt, 8)
		assert.NotEmpty(t, codes[i].CodeHash)
		assert.Equal(t, userID, codes[i].UserID)
		assert.False(t, codes[i].Used)
	}

	// Plaintexts are all unique
	seen := make(map[string]bool, len(plaintexts))
	for _, pt := range plaintexts {
		assert.False(t, seen[pt], "duplicate recovery code: %s", pt)
		seen[pt] = true
	}
}

func TestVerifyRecoveryCode(t *testing.T) {
	userID := id.NewUserID()
	codes, plaintexts, err := mfa.GenerateRecoveryCodes(userID, 3)
	require.NoError(t, err)

	// Valid code matches
	assert.True(t, mfa.VerifyRecoveryCode(plaintexts[0], codes[0]))

	// Wrong code doesn't match
	assert.False(t, mfa.VerifyRecoveryCode("wrongcode", codes[0]))

	// Used code doesn't match
	codes[1].Used = true
	assert.False(t, mfa.VerifyRecoveryCode(plaintexts[1], codes[1]))
}

func TestMemoryStore_RecoveryCodes(t *testing.T) {
	s := mfa.NewMemoryStore()
	ctx := context.Background()
	userID := id.NewUserID()

	codes, _, err := mfa.GenerateRecoveryCodes(userID, 4)
	require.NoError(t, err)

	// Create
	err = s.CreateRecoveryCodes(ctx, codes)
	require.NoError(t, err)

	// Get
	got, err := s.GetRecoveryCodes(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, got, 4)

	// Consume
	err = s.ConsumeRecoveryCode(ctx, codes[0].ID)
	require.NoError(t, err)

	got, err = s.GetRecoveryCodes(ctx, userID)
	require.NoError(t, err)
	for _, c := range got {
		if c.ID == codes[0].ID {
			assert.True(t, c.Used)
			assert.NotNil(t, c.UsedAt)
		}
	}

	// Delete all
	err = s.DeleteRecoveryCodes(ctx, userID)
	require.NoError(t, err)

	got, err = s.GetRecoveryCodes(ctx, userID)
	require.NoError(t, err)
	assert.Empty(t, got)
}

// ──────────────────────────────────────────────────
// Recovery code endpoint tests
// ──────────────────────────────────────────────────

func TestHandleVerify_ReturnsRecoveryCodes(t *testing.T) {
	p, s := newTestPlugin(t)

	mux := forge.NewRouter()
	err := p.RegisterRoutes(mux)
	require.NoError(t, err)

	userID := id.NewUserID()

	key, err := mfa.GenerateTOTPKey(mfa.TOTPConfig{Issuer: "TestApp", AccountName: "user@test.com"})
	require.NoError(t, err)

	enrollment := &mfa.Enrollment{
		ID:        id.NewMFAID(),
		UserID:    userID,
		Method:    "totp",
		Secret:    key.Secret(),
		Verified:  false, // Not yet verified — first verify will generate codes
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err = s.CreateEnrollment(context.Background(), enrollment)
	require.NoError(t, err)

	code, err := mfa.GenerateTOTPCode(key.Secret())
	require.NoError(t, err)

	body := jsonBody(t, map[string]string{"code": code})
	req := authedRequest(t, "POST", "/v1/mfa/verify", body, userID)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]any
	err = json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, true, resp["verified"])

	// Should include recovery codes on first verification
	rcRaw, ok := resp["recovery_codes"]
	require.True(t, ok, "response should contain recovery_codes")
	rcSlice, ok := rcRaw.([]any)
	require.True(t, ok)
	assert.Len(t, rcSlice, mfa.DefaultRecoveryCodeCount)

	// Recovery codes should be persisted in the store
	stored, err := s.GetRecoveryCodes(context.Background(), userID)
	require.NoError(t, err)
	assert.Len(t, stored, mfa.DefaultRecoveryCodeCount)
}

func TestHandleRecoveryVerify_Success(t *testing.T) {
	p, s := newTestPlugin(t)

	mux := forge.NewRouter()
	err := p.RegisterRoutes(mux)
	require.NoError(t, err)

	userID := id.NewUserID()

	// Create a verified enrollment
	enrollment := &mfa.Enrollment{
		ID:        id.NewMFAID(),
		UserID:    userID,
		Method:    "totp",
		Secret:    "SECRET",
		Verified:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err = s.CreateEnrollment(context.Background(), enrollment)
	require.NoError(t, err)

	// Generate and store recovery codes
	codes, plaintexts, err := mfa.GenerateRecoveryCodes(userID, 4)
	require.NoError(t, err)
	err = s.CreateRecoveryCodes(context.Background(), codes)
	require.NoError(t, err)

	// Use the first recovery code
	body := jsonBody(t, map[string]string{"code": plaintexts[0]})
	req := authedRequest(t, "POST", "/v1/mfa/recovery/verify", body, userID)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]any
	err = json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, true, resp["challenge_passed"])
	assert.Equal(t, float64(3), resp["codes_remaining"])

	// Same code should not work again (one-time use)
	body2 := jsonBody(t, map[string]string{"code": plaintexts[0]})
	req2 := authedRequest(t, "POST", "/v1/mfa/recovery/verify", body2, userID)
	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, req2)

	assert.Equal(t, http.StatusUnauthorized, rec2.Code)
}

func TestHandleRecoveryVerify_InvalidCode(t *testing.T) {
	p, s := newTestPlugin(t)

	mux := forge.NewRouter()
	err := p.RegisterRoutes(mux)
	require.NoError(t, err)

	userID := id.NewUserID()

	enrollment := &mfa.Enrollment{
		ID:        id.NewMFAID(),
		UserID:    userID,
		Method:    "totp",
		Secret:    "SECRET",
		Verified:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err = s.CreateEnrollment(context.Background(), enrollment)
	require.NoError(t, err)

	codes, _, err := mfa.GenerateRecoveryCodes(userID, 4)
	require.NoError(t, err)
	err = s.CreateRecoveryCodes(context.Background(), codes)
	require.NoError(t, err)

	body := jsonBody(t, map[string]string{"code": "badcode1"})
	req := authedRequest(t, "POST", "/v1/mfa/recovery/verify", body, userID)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestHandleRecoveryRegenerate(t *testing.T) {
	p, s := newTestPlugin(t)

	mux := forge.NewRouter()
	err := p.RegisterRoutes(mux)
	require.NoError(t, err)

	userID := id.NewUserID()

	enrollment := &mfa.Enrollment{
		ID:        id.NewMFAID(),
		UserID:    userID,
		Method:    "totp",
		Secret:    "SECRET",
		Verified:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err = s.CreateEnrollment(context.Background(), enrollment)
	require.NoError(t, err)

	// Create initial codes
	oldCodes, oldPlaintexts, err := mfa.GenerateRecoveryCodes(userID, 4)
	require.NoError(t, err)
	err = s.CreateRecoveryCodes(context.Background(), oldCodes)
	require.NoError(t, err)

	// Regenerate
	body := jsonBody(t, map[string]string{})
	req := authedRequest(t, "POST", "/v1/mfa/recovery/regenerate", body, userID)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]any
	err = json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	newCodesRaw := resp["codes"].([]any)
	assert.Len(t, newCodesRaw, mfa.DefaultRecoveryCodeCount)

	// Old codes should no longer work
	stored, err := s.GetRecoveryCodes(context.Background(), userID)
	require.NoError(t, err)
	assert.Len(t, stored, mfa.DefaultRecoveryCodeCount)

	// Verify old code doesn't match any new code
	for _, c := range stored {
		assert.False(t, mfa.VerifyRecoveryCode(oldPlaintexts[0], c))
	}
}

func TestHandleRecoveryVerify_NoEnrollment(t *testing.T) {
	p, _ := newTestPlugin(t)

	mux := forge.NewRouter()
	err := p.RegisterRoutes(mux)
	require.NoError(t, err)

	userID := id.NewUserID()
	body := jsonBody(t, map[string]string{"code": "testcode"})
	req := authedRequest(t, "POST", "/v1/mfa/recovery/verify", body, userID)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandleRecoveryRegenerate_NoEnrollment(t *testing.T) {
	p, _ := newTestPlugin(t)

	mux := forge.NewRouter()
	err := p.RegisterRoutes(mux)
	require.NoError(t, err)

	userID := id.NewUserID()
	body := jsonBody(t, map[string]string{})
	req := authedRequest(t, "POST", "/v1/mfa/recovery/regenerate", body, userID)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// ──────────────────────────────────────────────────
// Step-up on disable
// ──────────────────────────────────────────────────

// enrolledUser creates a verified TOTP enrollment and returns its secret.
func enrolledUser(t *testing.T, s mfa.Store, userID id.UserID) string {
	t.Helper()
	key, err := mfa.GenerateTOTPKey(mfa.TOTPConfig{Issuer: "Test", AccountName: "u@example.com"})
	require.NoError(t, err)
	require.NoError(t, s.CreateEnrollment(context.Background(), &mfa.Enrollment{
		ID:        id.NewMFAID(),
		UserID:    userID,
		Method:    "totp",
		Secret:    key.Secret(),
		Verified:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}))
	return key.Secret()
}

func disableWith(t *testing.T, mux forge.Router, userID id.UserID, body *bytes.Buffer) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, authedRequest(t, "DELETE", "/v1/mfa/enrollment", body, userID))
	return rec
}

// Turning MFA off is the action MFA exists to protect. On session auth alone,
// a stolen or hijacked session can strip the second factor silently and keep
// long-term access.
func TestHandleDisable_RequiresStepUpCode(t *testing.T) {
	p, s := newTestPlugin(t)
	mux := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(mux))

	userID := id.NewUserID()
	enrolledUser(t, s, userID)

	rec := disableWith(t, mux, userID, nil)
	assert.Equal(t, http.StatusBadRequest, rec.Code, "a session alone must not disable MFA")

	_, err := s.GetEnrollment(context.Background(), userID, "totp")
	require.NoError(t, err, "enrollment must survive a code-less disable")
}

func TestHandleDisable_RejectsWrongCode(t *testing.T) {
	p, s := newTestPlugin(t)
	mux := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(mux))

	userID := id.NewUserID()
	enrolledUser(t, s, userID)

	rec := disableWith(t, mux, userID, jsonBody(t, map[string]string{"code": "000000"}))
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	_, err := s.GetEnrollment(context.Background(), userID, "totp")
	require.NoError(t, err, "enrollment must survive a wrong-code disable")
}

// A recovery code is a valid second factor for step-up: a user whose
// authenticator is lost still needs to be able to turn MFA off.
func TestHandleDisable_AcceptsRecoveryCode(t *testing.T) {
	p, s := newTestPlugin(t)
	mux := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(mux))

	userID := id.NewUserID()
	enrolledUser(t, s, userID)

	codes, plaintexts, err := mfa.GenerateRecoveryCodes(userID, 3)
	require.NoError(t, err)
	require.NoError(t, s.CreateRecoveryCodes(context.Background(), codes))

	rec := disableWith(t, mux, userID, jsonBody(t, map[string]string{"code": plaintexts[0]}))
	assert.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

	_, err = s.GetEnrollment(context.Background(), userID, "totp")
	assert.ErrorIs(t, err, mfa.ErrEnrollmentNotFound)
}

// The recovery code must be burned, or it stays usable for a second
// sensitive action.
func TestHandleDisable_BurnsUsedRecoveryCode(t *testing.T) {
	p, s := newTestPlugin(t)
	mux := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(mux))

	userID := id.NewUserID()
	enrolledUser(t, s, userID)

	codes, plaintexts, err := mfa.GenerateRecoveryCodes(userID, 3)
	require.NoError(t, err)
	require.NoError(t, s.CreateRecoveryCodes(context.Background(), codes))

	require.Equal(t, http.StatusOK,
		disableWith(t, mux, userID, jsonBody(t, map[string]string{"code": plaintexts[0]})).Code)

	remaining, err := s.GetRecoveryCodes(context.Background(), userID)
	require.NoError(t, err)
	unused := 0
	for _, c := range remaining {
		if !c.Used {
			unused++
		}
	}
	assert.Equal(t, 2, unused, "the redeemed recovery code must be burned")
}

// A TOTP code spent on one sensitive action must not authorise another within
// its acceptance window.
func TestHandleDisable_RejectsReplayedTOTPCode(t *testing.T) {
	p, s := newTestPlugin(t)
	mux := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(mux))

	userID := id.NewUserID()
	secret := enrolledUser(t, s, userID)

	code, err := mfa.GenerateTOTPCode(secret)
	require.NoError(t, err)

	// Spend the code on /verify first.
	verifyRec := httptest.NewRecorder()
	mux.ServeHTTP(verifyRec, authedRequest(t, "POST", "/v1/mfa/verify",
		jsonBody(t, map[string]string{"code": code}), userID))
	require.Equal(t, http.StatusOK, verifyRec.Code)

	rec := disableWith(t, mux, userID, jsonBody(t, map[string]string{"code": code}))
	assert.Equal(t, http.StatusUnauthorized, rec.Code, "a spent TOTP code must not authorise a disable")
}
