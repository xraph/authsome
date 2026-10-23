package apikey

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/store/memory"
)

// The handler refuses an agent session on its own, so a mount that forgets
// the route guard still mints nothing.
func TestHandleCreate_RefusesAgentSessionWithoutRouteGuard(t *testing.T) {
	p := New()
	keys := memory.New()
	p.SetStore(keys)
	router := forge.NewRouter()
	require.NoError(t, router.POST("/bare/keys", p.handleCreate))

	userID, appID := id.NewUserID(), id.NewAppID()
	ctx := middleware.WithUserID(context.Background(), userID)
	ctx = middleware.WithSession(ctx, &session.Session{
		ID: id.NewSessionID(), AppID: appID, UserID: userID,
		PrincipalKind: session.PrincipalKindAgent, AgentID: id.NewAgentID(), GrantID: id.NewAgentGrantID(),
	})
	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/bare/keys",
		bytes.NewReader([]byte(`{"app_id":"`+appID.String()+`","name":"laundered"}`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code, "body=%s", rec.Body.String())
	list, err := keys.ListAPIKeysByUser(context.Background(), appID, userID)
	require.NoError(t, err)
	assert.Empty(t, list)
}
