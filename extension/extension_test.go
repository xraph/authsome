package extension

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/forge"
)

func TestName(t *testing.T) {
	ext := New()
	assert.Equal(t, "authsome", ext.Name())
}

func TestDescription(t *testing.T) {
	ext := New()
	assert.Equal(t, ExtensionDescription, ext.Description())
}

func TestVersion(t *testing.T) {
	ext := New()
	assert.Equal(t, "0.5.0", ext.Version())
}

func TestDependencies(t *testing.T) {
	ext := New()
	assert.Empty(t, ext.Dependencies())
}

func TestEngine_NilBeforeRegister(t *testing.T) {
	ext := New()
	assert.Nil(t, ext.Engine())
}

func TestMiddlewares_NilWhenEngineNotInitialized(t *testing.T) {
	ext := New()
	assert.Nil(t, ext.Middlewares())
}

func TestHealth_NotInitialized(t *testing.T) {
	ext := New()
	err := ext.Health(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not initialized")
}

func TestStart_NotInitialized(t *testing.T) {
	ext := New()
	err := ext.Start(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not initialized")
}

func TestStop_NilEngineIsNoop(t *testing.T) {
	ext := New()
	err := ext.Stop(context.Background())
	assert.NoError(t, err)
}

func TestHandler_NoAPI(t *testing.T) {
	ext := New()
	h := ext.Handler()
	assert.NotNil(t, h, "should return a fallback handler even when apiHandler is nil")
}

// Compile-time check that Extension satisfies forge.MiddlewareExtension.
var _ forge.MiddlewareExtension = (*Extension)(nil)
