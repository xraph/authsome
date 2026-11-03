package mfa

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/session"
)

// The handlers refuse an agent session on their own, so a mount that forgets
// the route guard still changes nothing on the account.
func TestHandlers_RefuseAgentSessionWithoutRouteGuard(t *testing.T) {
	p := New(Config{Issuer: "TestApp"})
	store := NewMemoryStore()
	p.SetStore(store)
	router := forge.NewRouter()
	require.NoError(t, router.POST("/bare/enroll", p.handleEnroll))
	require.NoError(t, router.POST("/bare/regenerate", p.handleRecoveryRegenerate))
	require.NoError(t, router.DELETE("/bare/enrollment", p.handleDisable))

	userID := id.NewUserID()
	key, err := GenerateTOTPKey(TOTPConfig{Issuer: "TestApp", AccountName: "u@example.com"})
	require.NoError(t, err)
	other := id.NewUserID()
	require.NoError(t, store.CreateEnrollment(context.Background(), &Enrollment{
		ID: id.NewMFAID(), UserID: other, Method: "totp", Secret: key.Secret(), Verified: true,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))
	code, err := GenerateTOTPCode(key.Secret())
	require.NoError(t, err)

	do := func(method, path, body string, as id.UserID) int {
		ctx := middleware.WithUserID(context.Background(), as)
		ctx = middleware.WithSession(ctx, &session.Session{
			ID: id.NewSessionID(), UserID: as, CreatedAt: time.Now(),
			PrincipalKind: session.PrincipalKindAgent, AgentID: id.NewAgentID(), GrantID: id.NewAgentGrantID(),
		})
		req := httptest.NewRequestWithContext(ctx, method, path, bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec.Code
	}

	assert.Equal(t, http.StatusForbidden, do(http.MethodPost, "/bare/enroll", `{"method":"totp"}`, userID))
	list, err := store.ListEnrollments(context.Background(), userID)
	require.NoError(t, err)
	assert.Empty(t, list, "no factor is enrolled")

	assert.Equal(t, http.StatusForbidden, do(http.MethodPost, "/bare/regenerate", `{}`, other))
	codes, err := store.GetRecoveryCodes(context.Background(), other)
	require.NoError(t, err)
	assert.Empty(t, codes, "no recovery codes are minted")

	assert.Equal(t, http.StatusForbidden, do(http.MethodDelete, "/bare/enrollment", `{"code":"`+code+`"}`, other))
	_, err = store.GetEnrollment(context.Background(), other, "totp")
	assert.NoError(t, err, "the factor stays enrolled")
}
