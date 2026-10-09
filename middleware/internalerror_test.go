package middleware_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"
	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/middleware"
)

// The client sees a generic 500; the cause reaches the log under the request
// id and stays on the error for errors.Is.
func TestInternalError_GenericBodyLoggedCause(t *testing.T) {
	tl, ok := log.NewTestLogger().(*log.TestLogger)
	require.True(t, ok)
	middleware.SetInternalErrorLogger(tl)
	t.Cleanup(func() { middleware.SetInternalErrorLogger(log.NewNoopLogger()) })

	cause := errors.New("pg: relation authsome_users does not exist")
	router := forge.NewRouter()
	router.GET("/boom", func(ctx forge.Context) error { return middleware.InternalError(ctx, cause) })
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/boom", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.NotContains(t, rec.Body.String(), "authsome_users", "the cause never reaches the client")
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, http.StatusText(http.StatusInternalServerError), body["error"], "forge masks every 5xx body to its status text")
	assert.EqualValues(t, 500, body["code"])

	logs := tl.GetLogsByLevel("ERROR")
	require.NotEmpty(t, logs, "the cause is logged")
	v, found := logs[len(logs)-1].Field("error")
	assert.True(t, found)
	assert.Contains(t, v, "authsome_users")

	err := middleware.InternalErrorCtx(context.Background(), cause)
	assert.ErrorIs(t, err, cause, "the cause stays reachable in code")
	var httpErr *middleware.InternalHTTPError
	assert.ErrorAs(t, err, &httpErr)
	assert.Equal(t, http.StatusInternalServerError, httpErr.StatusCode())
}
