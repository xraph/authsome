package generator

import (
	"strings"
)

// generateForgeMiddleware generates Forge-specific middleware for the Go client
func (g *GoGenerator) generateForgeMiddleware() error {
	var sb strings.Builder

	sb.WriteString("package authsome\n\n")
	sb.WriteString("import (\n")
	sb.WriteString("\t\"context\"\n")
	sb.WriteString("\t\"net/http\"\n")
	sb.WriteString("\n")
	sb.WriteString("\t\"github.com/xraph/forge\"\n")
	sb.WriteString(")\n\n")
	sb.WriteString("// Auto-generated Forge middleware\n\n")

	// ForgeMiddleware method
	sb.WriteString("// ForgeMiddleware returns a Forge middleware that injects auth into context\n")
	sb.WriteString("// This middleware verifies the session with the AuthSome backend and populates\n")
	sb.WriteString("// the request context with user and session information\n")
	sb.WriteString("func (c *Client) ForgeMiddleware() forge.Middleware {\n")
	sb.WriteString("\treturn func(next forge.Handler) forge.Handler {\n")
	sb.WriteString("\t\treturn func(ctx forge.Context) error {\n")
	sb.WriteString("\t\t\t// Try to verify session with AuthSome backend\n")
	sb.WriteString("\t\t\tsession, err := c.GetCurrentSession(ctx.Request().Context())\n")
	sb.WriteString("\t\t\tif err == nil && session != nil {\n")
	sb.WriteString("\t\t\t\t// Inject user/session into request context\n")
	sb.WriteString("\t\t\t\tnewCtx := withAuthContext(ctx.Request().Context(), session)\n")
	sb.WriteString("\t\t\t\t*ctx.Request() = *ctx.Request().WithContext(newCtx)\n")
	sb.WriteString("\t\t\t}\n")
	sb.WriteString("\t\t\treturn next(ctx)\n")
	sb.WriteString("\t\t}\n")
	sb.WriteString("\t}\n")
	sb.WriteString("}\n\n")

	// RequireAuth method
	sb.WriteString("// RequireAuth returns Forge middleware that requires authentication\n")
	sb.WriteString("// Requests without valid authentication will receive a 401 response\n")
	sb.WriteString("func (c *Client) RequireAuth() forge.Middleware {\n")
	sb.WriteString("\treturn func(next forge.Handler) forge.Handler {\n")
	sb.WriteString("\t\treturn func(ctx forge.Context) error {\n")
	sb.WriteString("\t\t\tauthCtx := getAuthContext(ctx.Request().Context())\n")
	sb.WriteString("\t\t\tif authCtx == nil || authCtx.Session == nil {\n")
	sb.WriteString("\t\t\t\treturn ctx.JSON(http.StatusUnauthorized, map[string]string{\n")
	sb.WriteString("\t\t\t\t\t\"error\": \"authentication required\",\n")
	sb.WriteString("\t\t\t\t\t\"code\":  \"AUTHENTICATION_REQUIRED\",\n")
	sb.WriteString("\t\t\t\t})\n")
	sb.WriteString("\t\t\t}\n")
	sb.WriteString("\t\t\treturn next(ctx)\n")
	sb.WriteString("\t\t}\n")
	sb.WriteString("\t}\n")
	sb.WriteString("}\n\n")

	// OptionalAuth method
	sb.WriteString("// OptionalAuth returns Forge middleware that optionally loads auth if present\n")
	sb.WriteString("// Unlike RequireAuth, this does not block unauthenticated requests\n")
	sb.WriteString("func (c *Client) OptionalAuth() forge.Middleware {\n")
	sb.WriteString("\treturn c.ForgeMiddleware()\n")
	sb.WriteString("}\n\n")

	// Context helper functions
	sb.WriteString("// Context management for Forge middleware\n")
	sb.WriteString("type contextKey string\n\n")
	sb.WriteString("const (\n")
	sb.WriteString("\tsessionContextKey contextKey = \"authsome_session\"\n")
	sb.WriteString("\tuserContextKey    contextKey = \"authsome_user\"\n")
	sb.WriteString(")\n\n")

	sb.WriteString("type authContext struct {\n")
	sb.WriteString("\tSession *Session\n")
	sb.WriteString("\tUser    *User\n")
	sb.WriteString("}\n\n")

	sb.WriteString("func withAuthContext(ctx context.Context, session *Session) context.Context {\n")
	sb.WriteString("\tctx = context.WithValue(ctx, sessionContextKey, session)\n")
	sb.WriteString("\treturn ctx\n")
	sb.WriteString("}\n\n")

	sb.WriteString("func getAuthContext(ctx context.Context) *authContext {\n")
	sb.WriteString("\tsession, _ := ctx.Value(sessionContextKey).(*Session)\n")
	sb.WriteString("\tif session == nil {\n")
	sb.WriteString("\t\treturn nil\n")
	sb.WriteString("\t}\n")
	sb.WriteString("\treturn &authContext{\n")
	sb.WriteString("\t\tSession: session,\n")
	sb.WriteString("\t}\n")
	sb.WriteString("}\n\n")

	// Exported context retrieval functions
	sb.WriteString("// GetUserFromContext retrieves user ID from Forge context\n")
	sb.WriteString("func GetUserFromContext(ctx context.Context) (*Session, bool) {\n")
	sb.WriteString("\tsession, ok := ctx.Value(sessionContextKey).(*Session)\n")
	sb.WriteString("\treturn session, ok\n")
	sb.WriteString("}\n\n")

	sb.WriteString("// GetSessionFromContext retrieves session from Forge context\n")
	sb.WriteString("func GetSessionFromContext(ctx context.Context) (*Session, bool) {\n")
	sb.WriteString("\tsession, ok := ctx.Value(sessionContextKey).(*Session)\n")
	sb.WriteString("\treturn session, ok\n")
	sb.WriteString("}\n")

	return g.writeFile("middleware_forge.go", sb.String())
}

