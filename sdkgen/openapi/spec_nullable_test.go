package openapi_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/sdkgen/openapi"
)

// forge v1.12.9 wraps a pointer element in oneOf [$ref, null]. The loader
// must hand the generators the plain $ref, or every such list loses its
// element type.
func TestSchema_FoldsNullableRefWrapper(t *testing.T) {
	const doc = `{
	  "type": "object",
	  "properties": {
	    "connections": {
	      "type": "array",
	      "items": {
	        "oneOf": [
	          {"$ref": "#/components/schemas/Connection"},
	          {"enum": [null], "nullable": true, "type": "object"}
	        ]
	      }
	    }
	  }
	}`
	var s openapi.Schema
	require.NoError(t, json.Unmarshal([]byte(doc), &s))
	items := s.Properties["connections"].Items
	require.NotNil(t, items)
	assert.Equal(t, "#/components/schemas/Connection", items.Ref)

	// The order of the two branches carries no meaning.
	var flipped openapi.Schema
	require.NoError(t, json.Unmarshal([]byte(`{"oneOf": [
		{"enum": [null], "nullable": true, "type": "object"},
		{"$ref": "#/components/schemas/Connection"}
	]}`), &flipped))
	assert.Equal(t, "#/components/schemas/Connection", flipped.Ref)
}

// A oneOf that is not exactly that pair is left alone, so a real union is
// never silently narrowed to its first member.
func TestSchema_LeavesOtherOneOfAlone(t *testing.T) {
	cases := map[string]string{
		"two refs": `{"oneOf": [
			{"$ref": "#/components/schemas/A"},
			{"$ref": "#/components/schemas/B"}
		]}`,
		"ref and object": `{"oneOf": [
			{"$ref": "#/components/schemas/A"},
			{"type": "object"}
		]}`,
		"non-null enum": `{"oneOf": [
			{"$ref": "#/components/schemas/A"},
			{"enum": ["x"], "nullable": true, "type": "string"}
		]}`,
	}
	for name, doc := range cases {
		t.Run(name, func(t *testing.T) {
			var s openapi.Schema
			require.NoError(t, json.Unmarshal([]byte(doc), &s))
			assert.Empty(t, s.Ref)
		})
	}
}
