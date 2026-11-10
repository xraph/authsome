package dart_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/sdkgen/dart"
	"github.com/xraph/authsome/sdkgen/openapi"
)

// Operations whose tag starts with a deprecated prefix get @Deprecated;
// everything else is left alone.
func TestGenerate_DeprecatesTaggedOperations(t *testing.T) {
	spec, err := openapi.LoadSpecFromFile("../spec.json")
	require.NoError(t, err)

	gen := dart.NewGenerator(dart.GeneratorConfig{
		OutputMode: "embedded",
		DeprecatedTags: map[string]string{
			"Billing":  "Use the authsome_subscription package instead.",
			"Billing ": "Operator route. Call it from your server.",
		},
	})
	files, err := gen.Generate(spec)
	require.NoError(t, err)

	var client string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "api_client.dart") {
			client = f.Content
		}
	}
	require.NotEmpty(t, client)

	before := func(method string) string {
		i := strings.Index(client, "> "+method+"(")
		require.Greater(t, i, 0, method)
		start := strings.LastIndex(client[:i], "\n\n")
		return client[start:i]
	}
	assert.Contains(t, before("getCurrentSubscription"), "@Deprecated('Use the authsome_subscription package instead.')")
	assert.Contains(t, before("getActiveSubscription"), "@Deprecated('Operator route. Call it from your server.')")
	assert.NotContains(t, before("getMe"), "@Deprecated")
}

// A note is a Dart string literal, so quotes, dollars and backslashes are
// escaped and it can't end the annotation early.
func TestGenerate_EscapesDeprecationNotes(t *testing.T) {
	spec, err := openapi.LoadSpecFromFile("../spec.json")
	require.NoError(t, err)

	gen := dart.NewGenerator(dart.GeneratorConfig{
		OutputMode:     "embedded",
		DeprecatedTags: map[string]string{"Billing": `Don't use $this \ here`},
	})
	files, err := gen.Generate(spec)
	require.NoError(t, err)

	for _, f := range files {
		if strings.HasSuffix(f.Path, "api_client.dart") {
			assert.Contains(t, f.Content, `@Deprecated('Don\'t use \$this \\ here')`)
			return
		}
	}
	t.Fatal("no api_client.dart")
}
