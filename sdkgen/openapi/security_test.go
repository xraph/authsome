package openapi_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/xraph/authsome/sdkgen/openapi"
)

// schemes mirrors what specgen declares. "session" is the one that matters:
// routes reach for it via forge.WithGroupAuth("session", "session-cookie"),
// and because it was never declared the generators saw no scheme to inspect
// and dropped the token parameter from every operation using it.
func schemes() map[string]*openapi.SecurityScheme {
	return map[string]*openapi.SecurityScheme{
		"bearerAuth":     {Type: "http", Scheme: "bearer", BearerFormat: "JWT"},
		"session":        {Type: "http", Scheme: "bearer", BearerFormat: "JWT"},
		"session-cookie": {Type: "apiKey", In: "cookie", Name: "auth_token"},
	}
}

func TestOperationTakesToken(t *testing.T) {
	tests := []struct {
		name     string
		security []openapi.SecurityRequirement
		want     bool
	}{
		{
			name:     "no security is a public operation",
			security: nil,
			want:     false,
		},
		{
			name:     "bearerAuth",
			security: []openapi.SecurityRequirement{{"bearerAuth": {}}},
			want:     true,
		},
		{
			// The regression this whole change exists for.
			name:     "session alongside its cookie fallback",
			security: []openapi.SecurityRequirement{{"session": {}}, {"session-cookie": {}}},
			want:     true,
		},
		{
			// Cookie travels on its own, so there is no token to pass.
			name:     "cookie only",
			security: []openapi.SecurityRequirement{{"session-cookie": {}}},
			want:     false,
		},
		{
			// A name nobody declared contributes nothing rather than
			// quietly deciding the answer either way.
			name:     "undeclared scheme",
			security: []openapi.SecurityRequirement{{"mystery": {}}},
			want:     false,
		},
		{
			name:     "undeclared scheme next to a real one",
			security: []openapi.SecurityRequirement{{"mystery": {}}, {"session": {}}},
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			op := &openapi.Operation{Security: tt.security}
			assert.Equal(t, tt.want, openapi.OperationTakesToken(op, schemes()))
		})
	}
}

func TestOperationTakesToken_NilSafety(t *testing.T) {
	assert.False(t, openapi.OperationTakesToken(nil, schemes()))

	// A spec with no components block at all still has to answer, and the
	// honest answer is "nothing is declared, so nothing takes a token".
	op := &openapi.Operation{Security: []openapi.SecurityRequirement{{"session": {}}}}
	assert.False(t, openapi.OperationTakesToken(op, nil))

	var spec *openapi.Spec
	assert.Nil(t, spec.SecuritySchemes())
	assert.Nil(t, (&openapi.Spec{}).SecuritySchemes())
}
