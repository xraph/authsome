package middleware_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"

	"github.com/xraph/authsome/middleware"
)

type recordingCollector struct {
	mu     sync.Mutex
	events []string
}

func (r *recordingCollector) RecordEvent(action, resource, outcome, _ string, d time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if d < 0 {
		panic("negative duration")
	}
	r.events = append(r.events, action+"|"+resource+"|"+outcome)
}
func (*recordingCollector) IncrementGauge(string, string, int) {}
func (*recordingCollector) IncrementCounter(string, string)    {}

func TestRouteLabel(t *testing.T) {
	assert.Equal(t, "/v1/webhooks/:id/rotate-secret", middleware.RouteLabel("/v1/webhooks/awhk_01jf0000000000000000000000/rotate-secret"))
	assert.Equal(t, "/v1/sessions", middleware.RouteLabel("/v1/sessions"))
	assert.Equal(t, "/", middleware.RouteLabel("/"))
}

func TestMetricsMiddlewareRecordsRouteAndStatus(t *testing.T) {
	c := &recordingCollector{}
	router := forge.NewRouter()
	router.Use(middleware.Metrics(c))
	router.GET("/v1/users/:userId", func(ctx forge.Context) error { return ctx.NoContent(http.StatusOK) })
	router.GET("/v1/boom", func(forge.Context) error { return forge.NewHTTPError(http.StatusTeapot, "no") })
	router.GET("/v1/crash", func(forge.Context) error { return errors.New("unexpected") })

	for _, p := range []string{"/v1/users/ausr_01jf0000000000000000000000", "/v1/boom", "/v1/crash"} {
		req := httptest.NewRequestWithContext(context.Background(), "GET", p, nil)
		router.ServeHTTP(httptest.NewRecorder(), req)
	}
	require.Len(t, c.events, 3)
	assert.Equal(t, "http.request|GET /v1/users/:id|200", c.events[0])
	assert.Equal(t, "http.request|GET /v1/boom|418", c.events[1])
	assert.Equal(t, "http.request|GET /v1/crash|500", c.events[2])
}
