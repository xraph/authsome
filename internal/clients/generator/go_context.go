package generator

import (
	"strings"
)

// generateContextFile generates context management utilities for the Go client
func (g *GoGenerator) generateContextFile() error {
	var sb strings.Builder

	sb.WriteString("package authsome\n\n")
	sb.WriteString("import (\n")
	sb.WriteString("\t\"context\"\n")
	sb.WriteString(")\n\n")
	sb.WriteString("// Auto-generated context management utilities\n\n")

	// Context key type
	sb.WriteString("type authContextKey string\n\n")
	sb.WriteString("const (\n")
	sb.WriteString("\tappContextKey    authContextKey = \"authsome_app_id\"\n")
	sb.WriteString("\tenvContextKey    authContextKey = \"authsome_env_id\"\n")
	sb.WriteString("\tuserIDContextKey authContextKey = \"authsome_user_id\"\n")
	sb.WriteString(")\n\n")

	// App context functions
	sb.WriteString("// WithAppID adds app ID to context\n")
	sb.WriteString("func WithAppID(ctx context.Context, appID string) context.Context {\n")
	sb.WriteString("\treturn context.WithValue(ctx, appContextKey, appID)\n")
	sb.WriteString("}\n\n")

	sb.WriteString("// GetAppID retrieves app ID from context\n")
	sb.WriteString("func GetAppID(ctx context.Context) (string, bool) {\n")
	sb.WriteString("\tappID, ok := ctx.Value(appContextKey).(string)\n")
	sb.WriteString("\treturn appID, ok\n")
	sb.WriteString("}\n\n")

	// Environment context functions
	sb.WriteString("// WithEnvironmentID adds environment ID to context\n")
	sb.WriteString("func WithEnvironmentID(ctx context.Context, envID string) context.Context {\n")
	sb.WriteString("\treturn context.WithValue(ctx, envContextKey, envID)\n")
	sb.WriteString("}\n\n")

	sb.WriteString("// GetEnvironmentID retrieves environment ID from context\n")
	sb.WriteString("func GetEnvironmentID(ctx context.Context) (string, bool) {\n")
	sb.WriteString("\tenvID, ok := ctx.Value(envContextKey).(string)\n")
	sb.WriteString("\treturn envID, ok\n")
	sb.WriteString("}\n\n")

	// User ID context functions
	sb.WriteString("// WithUserID adds user ID to context\n")
	sb.WriteString("func WithUserID(ctx context.Context, userID string) context.Context {\n")
	sb.WriteString("\treturn context.WithValue(ctx, userIDContextKey, userID)\n")
	sb.WriteString("}\n\n")

	sb.WriteString("// GetUserID retrieves user ID from context\n")
	sb.WriteString("func GetUserID(ctx context.Context) (string, bool) {\n")
	sb.WriteString("\tuserID, ok := ctx.Value(userIDContextKey).(string)\n")
	sb.WriteString("\treturn userID, ok\n")
	sb.WriteString("}\n\n")

	// Composite context function
	sb.WriteString("// SetContextAppAndEnvironment adds both app and environment IDs to context\n")
	sb.WriteString("func SetContextAppAndEnvironment(ctx context.Context, appID, envID string) context.Context {\n")
	sb.WriteString("\tctx = context.WithValue(ctx, appContextKey, appID)\n")
	sb.WriteString("\tctx = context.WithValue(ctx, envContextKey, envID)\n")
	sb.WriteString("\treturn ctx\n")
	sb.WriteString("}\n")

	return g.writeFile("context.go", sb.String())
}

