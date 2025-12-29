package introspector

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/xraph/authsome/internal/clients/manifest"
)

// Introspector analyzes Go source code to generate manifests
type Introspector struct {
	projectRoot string
	fset        *token.FileSet
}

// NewIntrospector creates a new introspector
func NewIntrospector(projectRoot string) *Introspector {
	return &Introspector{
		projectRoot: projectRoot,
		fset:        token.NewFileSet(),
	}
}

// IntrospectHandlers analyzes handler files to extract route information
func (i *Introspector) IntrospectHandlers(handlerPath string) (*RouteInfo, error) {
	files, err := os.ReadDir(handlerPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read handler directory: %w", err)
	}

	routeInfo := &RouteInfo{
		Routes:  make([]Route, 0),
		Types:   make(map[string]*TypeInfo),
		Handler: filepath.Base(handlerPath),
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".go") {
			continue
		}

		fullPath := filepath.Join(handlerPath, file.Name())
		if err := i.parseHandlerFile(fullPath, routeInfo); err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", file.Name(), err)
		}
	}

	return routeInfo, nil
}

// parseHandlerFile parses a single handler file
func (i *Introspector) parseHandlerFile(filePath string, routeInfo *RouteInfo) error {
	node, err := parser.ParseFile(i.fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return err
	}

	// Extract handler methods
	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncDecl:
			i.extractHandlerMethod(x, routeInfo)
		case *ast.TypeSpec:
			i.extractTypeInfo(x, routeInfo)
		}
		return true
	})

	return nil
}

// extractHandlerMethod extracts route information from handler methods
func (i *Introspector) extractHandlerMethod(fn *ast.FuncDecl, routeInfo *RouteInfo) {
	// Check if it's a handler method (has *forge.Context parameter)
	if !i.isHandlerMethod(fn) {
		return
	}

	route := Route{
		Name:        fn.Name.Name,
		Description: i.extractComment(fn.Doc),
	}

	// Track variable declarations for type inference
	varTypes := make(map[string]*TypeInfo)

	// Extract request/response types from function body
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.DeclStmt:
			// Look for variable declarations like: var reqBody struct { ... }
			if genDecl, ok := x.Decl.(*ast.GenDecl); ok && genDecl.Tok == token.VAR {
				for _, spec := range genDecl.Specs {
					if valueSpec, ok := spec.(*ast.ValueSpec); ok {
						for idx, name := range valueSpec.Names {
							varName := name.Name
							// Extract type from inline struct or named type
							if idx < len(valueSpec.Values) {
								// Has initializer
								continue
							}
							if valueSpec.Type != nil {
								if structType, ok := valueSpec.Type.(*ast.StructType); ok {
									// Inline struct definition - use handler name to make unique
									uniqueName := fn.Name.Name + "_" + varName
									typeInfo := i.extractInlineStruct(uniqueName, structType)
									varTypes[varName] = typeInfo
									routeInfo.Types[uniqueName] = typeInfo
								} else {
									// Named type reference
									typeName := i.exprToString(valueSpec.Type)
									varTypes[varName] = &TypeInfo{Name: typeName}
								}
							}
						}
					}
				}
			}
		case *ast.CallExpr:
			// Look for c.BindJSON(&req) to find request type
			if i.isBindCall(x) {
				if reqVar := i.extractVarFromCall(x); reqVar != "" {
					if typeInfo, ok := varTypes[reqVar]; ok {
						route.RequestType = typeInfo.Name
					}
				}
			}
			// Look for json.NewDecoder().Decode(&req) to find request type
			if i.isDecodeCall(x) {
				if reqVar := i.extractVarFromCall(x); reqVar != "" {
					if typeInfo, ok := varTypes[reqVar]; ok {
						route.RequestType = typeInfo.Name
					}
				}
			}
			// Look for c.JSON(status, response) to find response type
			if i.isJSONCall(x) {
				if respVar := i.extractVarFromJSONCall(x); respVar != "" {
					if typeInfo, ok := varTypes[respVar]; ok {
						route.ResponseType = typeInfo.Name
					}
				}
				// Also check for inline struct literals like &CreateAPIKeyResponse{...}
				if respType := i.extractTypeFromJSONCall(x); respType != "" {
					route.ResponseType = respType
				}
			}
		}
		return true
	})

	// Always add handler methods, even if we couldn't determine request/response types
	// The route registration will provide the path and method
	routeInfo.Routes = append(routeInfo.Routes, route)
}

// isHandlerMethod checks if a function is a handler method
func (i *Introspector) isHandlerMethod(fn *ast.FuncDecl) bool {
	if fn.Recv == nil || fn.Type.Params == nil {
		return false
	}

	for _, param := range fn.Type.Params.List {
		if i.isForgeContext(param.Type) {
			return true
		}
	}

	return false
}

// isForgeContext checks if a type is forge.Context or *forge.Context
func (i *Introspector) isForgeContext(expr ast.Expr) bool {
	// Check for *forge.Context (pointer)
	if star, ok := expr.(*ast.StarExpr); ok {
		expr = star.X
	}

	// Check for forge.Context (value)
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}

	return ident.Name == "forge" && sel.Sel.Name == "Context"
}

// extractTypeInfo extracts type definitions
func (i *Introspector) extractTypeInfo(spec *ast.TypeSpec, routeInfo *RouteInfo) {
	structType, ok := spec.Type.(*ast.StructType)
	if !ok {
		return
	}

	typeInfo := &TypeInfo{
		Name:        spec.Name.Name,
		Description: i.extractComment(spec.Doc),
		Fields:      make(map[string]FieldInfo),
	}

	for _, field := range structType.Fields.List {
		if len(field.Names) == 0 {
			continue
		}

		fieldName := field.Names[0].Name
		fieldInfo := FieldInfo{
			Name:        fieldName,
			Type:        i.exprToString(field.Type),
			JSONTag:     i.extractJSONTag(field.Tag),
			Description: i.extractComment(field.Doc),
		}

		// Determine if required from JSON tag
		if fieldInfo.JSONTag != "" && !strings.Contains(fieldInfo.JSONTag, "omitempty") {
			fieldInfo.Required = true
		}

		typeInfo.Fields[fieldName] = fieldInfo
	}

	routeInfo.Types[typeInfo.Name] = typeInfo
}

// IntrospectRoutes analyzes route registration to extract HTTP methods and paths
func (i *Introspector) IntrospectRoutes(routesPath string) ([]RouteRegistration, error) {
	files, err := os.ReadDir(routesPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read routes directory: %w", err)
	}

	var registrations []RouteRegistration

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".go") {
			continue
		}

		fullPath := filepath.Join(routesPath, file.Name())
		regs, err := i.parseRoutesFile(fullPath)
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", file.Name(), err)
		}

		registrations = append(registrations, regs...)
	}

	return registrations, nil
}

// parseRoutesFile parses route registration code
func (i *Introspector) parseRoutesFile(filePath string) ([]RouteRegistration, error) {
	node, err := parser.ParseFile(i.fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var registrations []RouteRegistration

	ast.Inspect(node, func(n ast.Node) bool {
		// Look for app.POST("/path", handler) calls
		if call, ok := n.(*ast.CallExpr); ok {
			if reg := i.extractRouteRegistration(call); reg != nil {
				registrations = append(registrations, *reg)
			}
		}
		return true
	})

	return registrations, nil
}

// parseSchemaAnnotation extracts type name from forge.WithXxxSchema() call
func (i *Introspector) parseSchemaAnnotation(call *ast.CallExpr) string {
	// Handle: forge.WithRequestSchema(TypeName{})
	// or: forge.WithResponseSchema(200, "desc", TypeName{})
	if len(call.Args) == 0 {
		return ""
	}

	// Get the last argument which should be the type composite literal
	lastArg := call.Args[len(call.Args)-1]

	if compLit, ok := lastArg.(*ast.CompositeLit); ok {
		// Handle simple type: TypeName{}
		if ident, ok := compLit.Type.(*ast.Ident); ok {
			return ident.Name
		}
		// Handle qualified type: package.TypeName{}
		if sel, ok := compLit.Type.(*ast.SelectorExpr); ok {
			return sel.Sel.Name // Strip package qualifier
		}
	}

	return ""
}

// extractRouteRegistration extracts route registration from a call expression
func (i *Introspector) extractRouteRegistration(call *ast.CallExpr) *RouteRegistration {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil
	}

	// Check if it's a HTTP method (GET, POST, PUT, PATCH, DELETE)
	method := sel.Sel.Name
	validMethods := map[string]bool{
		"GET": true, "POST": true, "PUT": true, "PATCH": true, "DELETE": true,
	}
	if !validMethods[method] {
		return nil
	}

	// Extract path (first argument)
	if len(call.Args) < 2 {
		return nil
	}

	pathLit, ok := call.Args[0].(*ast.BasicLit)
	if !ok || pathLit.Kind != token.STRING {
		return nil
	}

	path := strings.Trim(pathLit.Value, `"`)

	// Extract handler name (second argument)
	var handlerName string
	switch handler := call.Args[1].(type) {
	case *ast.SelectorExpr:
		handlerName = handler.Sel.Name
	case *ast.Ident:
		handlerName = handler.Name
	}

	reg := &RouteRegistration{
		Method:      method,
		Path:        path,
		HandlerName: handlerName,
	}

	// Process forge.WithXXX options (args 2+)
	for idx := 2; idx < len(call.Args); idx++ {
		if optCall, ok := call.Args[idx].(*ast.CallExpr); ok {
			if sel, ok := optCall.Fun.(*ast.SelectorExpr); ok {
				switch sel.Sel.Name {
				case "WithRequestSchema":
					reg.RequestType = i.parseSchemaAnnotation(optCall)
				case "WithResponseSchema":
					// Handle: WithResponseSchema(200, "desc", Type{})
					// Only extract success responses (200-299 status codes)
					if len(optCall.Args) >= 3 {
						// First arg is status code
						if statusLit, ok := optCall.Args[0].(*ast.BasicLit); ok {
							if statusCode := statusLit.Value; statusCode >= "200" && statusCode < "300" {
								// Only use 2xx success responses, ignore error responses (4xx, 5xx)
								if responseType := i.parseSchemaAnnotation(optCall); responseType != "" {
									reg.ResponseType = responseType
								}
							}
						}
					}
				}
			}
		}
	}

	return reg
}

// IntrospectPlugin analyzes a plugin directory
func (i *Introspector) IntrospectPlugin(pluginPath string) (*PluginInfo, error) {
	pluginFile := filepath.Join(pluginPath, "plugin.go")
	if _, err := os.Stat(pluginFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("plugin.go not found in %s", pluginPath)
	}

	node, err := parser.ParseFile(i.fset, pluginFile, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	pluginInfo := &PluginInfo{
		ID:      filepath.Base(pluginPath),
		Name:    filepath.Base(pluginPath),
		Version: "1.0.0", // Default
	}

	// Extract plugin ID from ID() method
	ast.Inspect(node, func(n ast.Node) bool {
		if fn, ok := n.(*ast.FuncDecl); ok {
			if fn.Name.Name == "ID" && fn.Body != nil {
				ast.Inspect(fn.Body, func(n2 ast.Node) bool {
					if ret, ok := n2.(*ast.ReturnStmt); ok && len(ret.Results) > 0 {
						if lit, ok := ret.Results[0].(*ast.BasicLit); ok {
							pluginInfo.ID = strings.Trim(lit.Value, `"`)
							return false
						}
					}
					return true
				})
			}
		}
		return true
	})

	return pluginInfo, nil
}

// GenerateManifest creates a manifest from introspection data
func (i *Introspector) GenerateManifest(pluginID string) (*manifest.Manifest, error) {
	pluginPath := filepath.Join(i.projectRoot, "plugins", pluginID)

	// Get plugin info
	pluginInfo, err := i.IntrospectPlugin(pluginPath)
	if err != nil {
		return nil, err
	}

	// Get handler info - look in handlers subdirectory if it exists, otherwise plugin root
	handlerPath := filepath.Join(pluginPath, "handlers")
	if _, err := os.Stat(handlerPath); os.IsNotExist(err) {
		handlerPath = pluginPath
	}
	routeInfo, err := i.IntrospectHandlers(handlerPath)
	if err != nil {
		return nil, err
	}

	// Get route registrations from plugin directory (routes.go or plugin.go)
	registrations, err := i.IntrospectRoutes(pluginPath)
	if err != nil {
		return nil, err
	}

	// Match handlers with routes
	m := &manifest.Manifest{
		PluginID:    pluginInfo.ID,
		Version:     pluginInfo.Version,
		Description: pluginInfo.Description,
		Routes:      make([]manifest.Route, 0),
		Types:       make([]manifest.TypeDef, 0),
	}

	// Convert route info to manifest routes
	for _, route := range routeInfo.Routes {
		// Find matching registration
		var reg *RouteRegistration
		for _, r := range registrations {
			if r.HandlerName == route.Name {
				reg = &r
				break
			}
		}

		if reg == nil {
			continue
		}

		// Handle empty paths (from router groups) - use a placeholder
		path := reg.Path
		if path == "" {
			// Try to infer from handler name (e.g., CreateAPIKey -> /api-key)
			path = "/" + strings.ToLower(route.Name)
		}

		// Prefer schema types from route registration (WithRequestSchema/WithResponseSchema)
		// over types extracted from handler code
		requestType := route.RequestType
		if reg.RequestType != "" {
			requestType = reg.RequestType
		}
		responseType := route.ResponseType
		if reg.ResponseType != "" {
			responseType = reg.ResponseType
		}

		manifestRoute := manifest.Route{
			Name:         route.Name,
			Description:  route.Description,
			Method:       reg.Method,
			Path:         path,
			Request:      i.convertTypeToFields(requestType, routeInfo),
			Response:     i.convertTypeToFields(responseType, routeInfo),
			RequestType:  requestType,  // Store the named type
			ResponseType: responseType, // Store the named type
		}

		m.Routes = append(m.Routes, manifestRoute)
	}

	// Convert types
	for _, typeInfo := range routeInfo.Types {
		typeDef := manifest.TypeDef{
			Name:        typeInfo.Name,
			Description: typeInfo.Description,
			Fields:      make(map[string]string),
		}

		for _, field := range typeInfo.Fields {
			fieldType := field.Type
			if field.Required {
				fieldType += "!"
			}
			typeDef.Fields[field.JSONTag] = fieldType
		}

		m.Types = append(m.Types, typeDef)
	}

	return m, nil
}

// Helper methods

func (i *Introspector) extractComment(doc *ast.CommentGroup) string {
	if doc == nil {
		return ""
	}
	return strings.TrimSpace(doc.Text())
}

func (i *Introspector) extractJSONTag(tag *ast.BasicLit) string {
	if tag == nil {
		return ""
	}

	tagValue := strings.Trim(tag.Value, "`")
	for _, part := range strings.Fields(tagValue) {
		if strings.HasPrefix(part, "json:") {
			jsonTag := strings.Trim(part[5:], `"`)
			// Remove omitempty and other options
			if idx := strings.Index(jsonTag, ","); idx > 0 {
				return jsonTag[:idx]
			}
			return jsonTag
		}
	}

	return ""
}

func (i *Introspector) exprToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + i.exprToString(t.X)
	case *ast.ArrayType:
		return "[]" + i.exprToString(t.Elt)
	case *ast.SelectorExpr:
		return i.exprToString(t.X) + "." + t.Sel.Name
	default:
		return ""
	}
}

func (i *Introspector) isBindCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	return ok && sel.Sel.Name == "BindJSON"
}

func (i *Introspector) isJSONCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	return ok && sel.Sel.Name == "JSON"
}

func (i *Introspector) isDecodeCall(call *ast.CallExpr) bool {
	// Look for json.NewDecoder().Decode(&req) pattern
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Decode" {
		return false
	}
	// Check if receiver is a call to NewDecoder
	if innerCall, ok := sel.X.(*ast.CallExpr); ok {
		if innerSel, ok := innerCall.Fun.(*ast.SelectorExpr); ok {
			return innerSel.Sel.Name == "NewDecoder"
		}
	}
	return false
}

func (i *Introspector) extractVarFromCall(call *ast.CallExpr) string {
	if len(call.Args) == 0 {
		return ""
	}

	// For Decode(&req) or BindJSON(&req), extract variable name from unary expression
	if unary, ok := call.Args[0].(*ast.UnaryExpr); ok {
		if ident, ok := unary.X.(*ast.Ident); ok {
			return ident.Name
		}
	}

	return ""
}

func (i *Introspector) extractVarFromJSONCall(call *ast.CallExpr) string {
	// For c.JSON(status, response), extract second argument
	if len(call.Args) < 2 {
		return ""
	}

	if ident, ok := call.Args[1].(*ast.Ident); ok {
		return ident.Name
	}

	return ""
}

func (i *Introspector) extractTypeFromJSONCall(call *ast.CallExpr) string {
	// For c.JSON(status, &ResponseType{...}), extract type from composite literal
	if len(call.Args) < 2 {
		return ""
	}

	// Check for unary expression (address operator &)
	var expr ast.Expr = call.Args[1]
	if unary, ok := expr.(*ast.UnaryExpr); ok {
		expr = unary.X
	}

	// Check for composite literal
	if comp, ok := expr.(*ast.CompositeLit); ok {
		return i.exprToString(comp.Type)
	}

	return ""
}

func (i *Introspector) extractInlineStruct(varName string, structType *ast.StructType) *TypeInfo {
	typeInfo := &TypeInfo{
		Name:   varName,
		Fields: make(map[string]FieldInfo),
	}

	if structType.Fields == nil {
		return typeInfo
	}

	for _, field := range structType.Fields.List {
		// Extract JSON tag
		jsonTag := ""
		if field.Tag != nil {
			jsonTag = i.extractJSONTag(field.Tag)
		}

		// Skip fields without JSON tags or with "-"
		if jsonTag == "" || jsonTag == "-" {
			continue
		}

		// Extract field type
		fieldType := i.exprToString(field.Type)

		// Extract field names
		for _, name := range field.Names {
			fieldInfo := FieldInfo{
				Name:     name.Name,
				Type:     fieldType,
				JSONTag:  jsonTag,
				Required: !strings.Contains(field.Tag.Value, "omitempty"),
			}
			typeInfo.Fields[jsonTag] = fieldInfo
		}
	}

	return typeInfo
}

func (i *Introspector) extractTypeFromCall(call *ast.CallExpr) string {
	if len(call.Args) == 0 {
		return ""
	}

	// For BindJSON(&req), extract type from unary expression
	if unary, ok := call.Args[0].(*ast.UnaryExpr); ok {
		if ident, ok := unary.X.(*ast.Ident); ok {
			return ident.Name
		}
	}

	return ""
}

func (i *Introspector) convertTypeToFields(typeName string, routeInfo *RouteInfo) map[string]string {
	typeInfo, ok := routeInfo.Types[typeName]
	if !ok {
		return nil
	}

	fields := make(map[string]string)
	for _, field := range typeInfo.Fields {
		fieldType := field.Type
		if field.Required {
			fieldType += "!"
		}
		fields[field.JSONTag] = fieldType
	}

	return fields
}

// Data structures

type RouteInfo struct {
	Handler string
	Routes  []Route
	Types   map[string]*TypeInfo
}

type Route struct {
	Name         string
	Description  string
	RequestType  string
	ResponseType string
}

type TypeInfo struct {
	Name        string
	Description string
	Fields      map[string]FieldInfo
}

type FieldInfo struct {
	Name        string
	Type        string
	JSONTag     string
	Required    bool
	Description string
}

type RouteRegistration struct {
	Method       string
	Path         string
	HandlerName  string
	RequestType  string // Type name from WithRequestSchema
	ResponseType string // Type name from WithResponseSchema
}

type PluginInfo struct {
	ID          string
	Name        string
	Version     string
	Description string
}
