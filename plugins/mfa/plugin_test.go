package mfa_test

import (
	"bytes"
	"context"
	"encoding/json"
	log "github.com/xraph/go-utils/log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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
		req = httptest.NewRequest(method, path, body)
	} else {
		req = httptest.NewRequest(method, path, nil)
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

func TestPlugin_ImplementsInterfaces(t *testing.T) {
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
	req := authedRequestWithUser(t, "POST", "/v1/auth/mfa/enroll", body, u)
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
	req := authedRequest(t, "POST", "/v1/auth/mfa/enroll", body, userID)
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
	req := authedRequest(t, "POST", "/v1/auth/mfa/enroll", body, userID)
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
	req := httptest.NewRequest("POST", "/v1/auth/mfa/enroll", body)
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
	req := authedRequest(t, "POST", "/v1/auth/mfa/enroll", body, userID)
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
	req := authedRequest(t, "POST", "/v1/auth/mfa/verify", body, userID)
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
	req := authedRequest(t, "POST", "/v1/auth/mfa/verify", body, userID)
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
	req := authedRequest(t, "POST", "/v1/auth/mfa/verify", body, userID)
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
	req := authedRequest(t, "POST", "/v1/auth/mfa/verify", body, userID)
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
	req := httptest.NewRequest("POST", "/v1/auth/mfa/verify", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// ──────────────────────────────────────────────────
// Challenge endpoint tests
// ──────────────────────────────────────────────────

func TestHandleChallenge_Success(t *testing.T) {
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
		Verified:  true, // Already verified
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err = s.CreateEnrollment(context.Background(), enrollment)
	require.NoError(t, err)

	code, err := mfa.GenerateTOTPCode(key.Secret())
	require.NoError(t, err)

	body := jsonBody(t, map[string]string{"code": code})
	req := authedRequest(t, "POST", "/v1/auth/mfa/challenge", body, userID)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]any
	err = json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, true, resp["challenge_passed"])
}

func TestHandleChallenge_NotVerified(t *testing.T) {
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
		Verified:  false, // Not yet verified
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err = s.CreateEnrollment(context.Background(), enrollment)
	require.NoError(t, err)

	code, err := mfa.GenerateTOTPCode(key.Secret())
	require.NoError(t, err)

	body := jsonBody(t, map[string]string{"code": code})
	req := authedRequest(t, "POST", "/v1/auth/mfa/challenge", body, userID)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
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

	req := authedRequest(t, "DELETE", "/v1/auth/mfa/enrollment", nil, userID)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

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
	req := authedRequest(t, "DELETE", "/v1/auth/mfa/enrollment", nil, userID)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestHandleDisable_Unauthenticated(t *testing.T) {
	p, _ := newTestPlugin(t)

	mux := forge.NewRouter()
	err := p.RegisterRoutes(mux)
	require.NoError(t, err)

	req := httptest.NewRequest("DELETE", "/v1/auth/mfa/enrollment", nil)
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
	enrollReq := authedRequestWithUser(t, "POST", "/v1/auth/mfa/enroll", enrollBody, u)
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
	verifyReq := authedRequest(t, "POST", "/v1/auth/mfa/verify", verifyBody, userID)
	verifyRec := httptest.NewRecorder()
	mux.ServeHTTP(verifyRec, verifyReq)
	require.Equal(t, http.StatusOK, verifyRec.Code)

	// Step 3: Challenge with valid code
	code2, err := mfa.GenerateTOTPCode(secret)
	require.NoError(t, err)

	challengeBody := jsonBody(t, map[string]string{"code": code2})
	challengeReq := authedRequest(t, "POST", "/v1/auth/mfa/challenge", challengeBody, userID)
	challengeRec := httptest.NewRecorder()
	mux.ServeHTTP(challengeRec, challengeReq)
	require.Equal(t, http.StatusOK, challengeRec.Code)

	// Step 4: Disable
	disableReq := authedRequest(t, "DELETE", "/v1/auth/mfa/enrollment", nil, userID)
	disableRec := httptest.NewRecorder()
	mux.ServeHTTP(disableRec, disableReq)
	assert.Equal(t, http.StatusOK, disableRec.Code)

	// Step 5: Challenge should now fail (no enrollment)
	code3, err := mfa.GenerateTOTPCode(secret)
	require.NoError(t, err)

	challengeBody2 := jsonBody(t, map[string]string{"code": code3})
	challengeReq2 := authedRequest(t, "POST", "/v1/auth/mfa/challenge", challengeBody2, userID)
	challengeRec2 := httptest.NewRecorder()
	mux.ServeHTTP(challengeRec2, challengeReq2)
	assert.Equal(t, http.StatusNotFound, challengeRec2.Code)
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
	req := authedRequest(t, "POST", "/v1/auth/mfa/verify", body, userID)
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
	req := authedRequest(t, "POST", "/v1/auth/mfa/recovery/verify", body, userID)
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
	req2 := authedRequest(t, "POST", "/v1/auth/mfa/recovery/verify", body2, userID)
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
	req := authedRequest(t, "POST", "/v1/auth/mfa/recovery/verify", body, userID)
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
	req := authedRequest(t, "POST", "/v1/auth/mfa/recovery/regenerate", body, userID)
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
	req := authedRequest(t, "POST", "/v1/auth/mfa/recovery/verify", body, userID)
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
	req := authedRequest(t, "POST", "/v1/auth/mfa/recovery/regenerate", body, userID)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
