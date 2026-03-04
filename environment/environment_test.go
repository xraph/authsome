package environment

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ──────────────────────────────────────────────────
// Type.IsValid tests
// ──────────────────────────────────────────────────

func TestType_IsValid_Development(t *testing.T) {
	assert.True(t, TypeDevelopment.IsValid())
}

func TestType_IsValid_Staging(t *testing.T) {
	assert.True(t, TypeStaging.IsValid())
}

func TestType_IsValid_Production(t *testing.T) {
	assert.True(t, TypeProduction.IsValid())
}

func TestType_IsValid_Unknown(t *testing.T) {
	assert.False(t, Type("unknown").IsValid())
}

func TestType_IsValid_Empty(t *testing.T) {
	assert.False(t, Type("").IsValid())
}

// ──────────────────────────────────────────────────
// Type.String tests
// ──────────────────────────────────────────────────

func TestType_String_Development(t *testing.T) {
	assert.Equal(t, "development", TypeDevelopment.String())
}

func TestType_String_Staging(t *testing.T) {
	assert.Equal(t, "staging", TypeStaging.String())
}

func TestType_String_Production(t *testing.T) {
	assert.Equal(t, "production", TypeProduction.String())
}

// ──────────────────────────────────────────────────
// Type.DefaultColor tests
// ──────────────────────────────────────────────────

func TestType_DefaultColor_Development(t *testing.T) {
	assert.Equal(t, "#22c55e", TypeDevelopment.DefaultColor())
}

func TestType_DefaultColor_Staging(t *testing.T) {
	assert.Equal(t, "#eab308", TypeStaging.DefaultColor())
}

func TestType_DefaultColor_Production(t *testing.T) {
	assert.Equal(t, "#ef4444", TypeProduction.DefaultColor())
}

func TestType_DefaultColor_Unknown(t *testing.T) {
	assert.Equal(t, "#6b7280", Type("unknown").DefaultColor())
}

// ──────────────────────────────────────────────────
// Type.DefaultName tests
// ──────────────────────────────────────────────────

func TestType_DefaultName_Development(t *testing.T) {
	assert.Equal(t, "Development", TypeDevelopment.DefaultName())
}

func TestType_DefaultName_Staging(t *testing.T) {
	assert.Equal(t, "Staging", TypeStaging.DefaultName())
}

func TestType_DefaultName_Production(t *testing.T) {
	assert.Equal(t, "Production", TypeProduction.DefaultName())
}

func TestType_DefaultName_Unknown(t *testing.T) {
	assert.Equal(t, "custom", Type("custom").DefaultName())
}
