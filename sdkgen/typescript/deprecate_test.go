package typescript_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/sdkgen/openapi"
	"github.com/xraph/authsome/sdkgen/typescript"
)

// Operations whose tag starts with a deprecated prefix get an @deprecated
// tag in their doc comment; everything else is left alone.
func TestGenerate_DeprecatesTaggedOperations(t *testing.T) {
	spec, err := openapi.LoadSpecFromFile("../spec.json")
	require.NoError(t, err)

	gen := typescript.NewGenerator(typescript.GeneratorConfig{
		OutputMode: "embedded",
		DeprecatedTags: map[string]string{
			"Billing":  "Use @authsome/plugin-subscription instead.",
			"Billing ": "Operator route. Call it from your server with @authsome/client.",
		},
	})
	files, err := gen.Generate(spec)
	require.NoError(t, err)

	var client string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "api-client.ts") {
			client = f.Content
		}
	}
	require.NotEmpty(t, client)

	docFor := func(method string) string {
		i := strings.Index(client, "  async "+method+"(")
		require.Greater(t, i, 0, method)
		start := strings.LastIndex(client[:i], "/**")
		return client[start:i]
	}
	assert.Contains(t, docFor("getCurrentSubscription"), "@deprecated Use @authsome/plugin-subscription instead.")
	assert.Contains(t, docFor("getActiveSubscription"), "@deprecated Operator route. Call it from your server with @authsome/client.")
	assert.Contains(t, docFor("subscribe"), "@deprecated")
	assert.NotContains(t, docFor("getMe"), "@deprecated")
}

// A note sits inside a JSDoc block, so "*/" can't close it early.
func TestGenerate_EscapesDeprecationNotes(t *testing.T) {
	spec, err := openapi.LoadSpecFromFile("../spec.json")
	require.NoError(t, err)

	gen := typescript.NewGenerator(typescript.GeneratorConfig{
		OutputMode:     "embedded",
		DeprecatedTags: map[string]string{"Billing": "Read docs/*/billing */ first"},
	})
	files, err := gen.Generate(spec)
	require.NoError(t, err)

	for _, f := range files {
		if strings.HasSuffix(f.Path, "api-client.ts") {
			assert.Contains(t, f.Content, `@deprecated Read docs/*\/billing *\/ first`)
			assert.NotContains(t, f.Content, "billing */ first")
			return
		}
	}
	t.Fatal("no api-client.ts")
}
