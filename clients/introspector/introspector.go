package introspector

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/xraph/authsome/clients/manifest"
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

	// Extract request/response types from function body
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.CallExpr:
			// Look for c.BindJSON(&req) to find request type
			if i.isBindCall(x) {
				if reqType := i.extractTypeFromCall(x); reqType != "" {
					route.RequestType = reqType
				}
			}
			// Look for c.JSON(status, response) to find response type
			if i.isJSONCall(x) {
				if respType := i.extractTypeFromCall(x); respType != "" {
					route.ResponseType = respType
				}
			}
		}
		return true
	})

	if route.RequestType != "" || route.ResponseType != "" {
		routeInfo.Routes = append(routeInfo.Routes, route)
	}
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

// isForgeContext checks if a type is *forge.Context
func (i *Introspector) isForgeContext(expr ast.Expr) bool {
	star, ok := expr.(*ast.StarExpr)
	if !ok {
		return false
	}

	sel, ok := star.X.(*ast.SelectorExpr)
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

	return &RouteRegistration{
		Method:      method,
		Path:        path,
		HandlerName: handlerName,
	}
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

	// Get handler info
	handlerPath := pluginPath
	routeInfo, err := i.IntrospectHandlers(handlerPath)
	if err != nil {
		return nil, err
	}

	// Get route registrations
	routesPath := filepath.Join(i.projectRoot, "routes")
	registrations, err := i.IntrospectRoutes(routesPath)
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

		manifestRoute := manifest.Route{
			Name:        route.Name,
			Description: route.Description,
			Method:      reg.Method,
			Path:        reg.Path,
			Request:     i.convertTypeToFields(route.RequestType, routeInfo),
			Response:    i.convertTypeToFields(route.ResponseType, routeInfo),
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
	Method      string
	Path        string
	HandlerName string
}

type PluginInfo struct {
	ID          string
	Name        string
	Version     string
	Description string
}
