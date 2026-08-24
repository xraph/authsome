package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// formBodySpec builds a document with one request body carried under the given
// content type, holding the given schema.
func formBodySpec(contentType string, schema map[string]any) map[string]any {
	return map[string]any{
		"paths": map[string]any{
			"/v1/oauth/token": map[string]any{
				"post": map[string]any{
					"operationId": "oauth2Token",
					"requestBody": map[string]any{
						"required": true,
						"content": map[string]any{
							contentType: map[string]any{"schema": schema},
						},
					},
				},
			},
		},
	}
}

func bodyContent(t *testing.T, spec map[string]any) map[string]any {
	t.Helper()
	paths := spec["paths"].(map[string]any)
	item := paths["/v1/oauth/token"].(map[string]any)
	post := item["post"].(map[string]any)
	body := post["requestBody"].(map[string]any)
	content, ok := body["content"].(map[string]any)
	require.True(t, ok, "request body should still have content")
	return content
}

// Forge describes any form-tagged struct as multipart/form-data. For the OAuth2
// endpoints that is the wrong half of the form family: RFC 6749 §4.1.3 requires
// application/x-www-form-urlencoded, and multipart is what a conformant client
// will never send.
func TestNormalizeFormContentTypes_RewritesScalarMultipartToURLEncoded(t *testing.T) {
	spec := formBodySpec("multipart/form-data", map[string]any{
		"type":     "object",
		"required": []any{"grant_type"},
		"properties": map[string]any{
			"grant_type": map[string]any{"type": "string"},
			"code":       map[string]any{"type": "string"},
		},
	})

	normalizeFormContentTypes(spec)

	content := bodyContent(t, spec)
	assert.Contains(t, content, "application/x-www-form-urlencoded")
	assert.NotContains(t, content, "multipart/form-data")
}

// The schema has to survive the move, otherwise the generators go back to
// having a body they know is there and no fields to put in it.
func TestNormalizeFormContentTypes_CarriesTheSchemaAcross(t *testing.T) {
	spec := formBodySpec("multipart/form-data", map[string]any{
		"type": "object",
		"properties": map[string]any{
			"grant_type": map[string]any{"type": "string"},
		},
	})

	normalizeFormContentTypes(spec)

	content := bodyContent(t, spec)
	media := content["application/x-www-form-urlencoded"].(map[string]any)
	schema := media["schema"].(map[string]any)
	props := schema["properties"].(map[string]any)
	assert.Contains(t, props, "grant_type")
}

// A genuine upload is multipart for a reason. Rewriting it would describe a file
// as a urlencoded scalar, so anything carrying binary is left alone.
func TestNormalizeFormContentTypes_LeavesFileUploadsAlone(t *testing.T) {
	spec := formBodySpec("multipart/form-data", map[string]any{
		"type": "object",
		"properties": map[string]any{
			"avatar": map[string]any{"type": "string", "format": "binary"},
		},
	})

	normalizeFormContentTypes(spec)

	content := bodyContent(t, spec)
	assert.Contains(t, content, "multipart/form-data")
	assert.NotContains(t, content, "application/x-www-form-urlencoded")
}

// A body that already says JSON is not a form body and must not be touched.
func TestNormalizeFormContentTypes_LeavesJSONAlone(t *testing.T) {
	spec := formBodySpec("application/json", map[string]any{
		"$ref": "#/components/schemas/SignInRequest",
	})

	normalizeFormContentTypes(spec)

	content := bodyContent(t, spec)
	assert.Contains(t, content, "application/json")
	assert.Len(t, content, 1)
}
