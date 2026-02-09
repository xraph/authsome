package introspector

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestExtractGroupDeclaration(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected *RouterGroup
	}{
		{
			name: "simple group from router",
			code: `package test
func test() {
	grp := router.Group("/oauth2")
}`,
			expected: &RouterGroup{
				VarName:   "grp",
				Path:      "/oauth2",
				ParentVar: "router",
			},
		},
		{
			name: "nested group",
			code: `package test
func test() {
	deviceGroup := grp.Group("/device")
}`,
			expected: &RouterGroup{
				VarName:   "deviceGroup",
				Path:      "/device",
				ParentVar: "grp",
			},
		},
		{
			name: "group with empty path",
			code: `package test
func test() {
	api := router.Group("")
}`,
			expected: &RouterGroup{
				VarName:   "api",
				Path:      "",
				ParentVar: "router",
			},
		},
		{
			name: "not a group call - different method",
			code: `package test
func test() {
	result := router.GET("/path")
}`,
			expected: nil,
		},
		{
			name: "group with path without leading slash",
			code: `package test
func test() {
	sub := grp.Group("sub")
}`,
			expected: &RouterGroup{
				VarName:   "sub",
				Path:      "sub",
				ParentVar: "grp",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()

			node, err := parser.ParseFile(fset, "test.go", tt.code, 0)
			if err != nil {
				t.Fatalf("failed to parse code: %v", err)
			}

			i := &Introspector{fset: fset}

			var result *RouterGroup

			ast.Inspect(node, func(n ast.Node) bool {
				if stmt, ok := n.(*ast.AssignStmt); ok {
					if group := i.extractGroupDeclaration(stmt); group != nil {
						result = group

						return false
					}
				}

				return true
			})

			if tt.expected == nil {
				if result != nil {
					t.Errorf("expected nil, got %+v", result)
				}

				return
			}

			if result == nil {
				t.Fatalf("expected %+v, got nil", tt.expected)
			}

			if result.VarName != tt.expected.VarName {
				t.Errorf("VarName: expected %q, got %q", tt.expected.VarName, result.VarName)
			}

			if result.Path != tt.expected.Path {
				t.Errorf("Path: expected %q, got %q", tt.expected.Path, result.Path)
			}

			if result.ParentVar != tt.expected.ParentVar {
				t.Errorf("ParentVar: expected %q, got %q", tt.expected.ParentVar, result.ParentVar)
			}
		})
	}
}

func TestResolveGroupPath(t *testing.T) {
	tests := []struct {
		name     string
		groups   map[string]*RouterGroup
		groupVar string
		expected string
	}{
		{
			name: "top-level group",
			groups: map[string]*RouterGroup{
				"grp": {VarName: "grp", Path: "/oauth2", ParentVar: "router"},
			},
			groupVar: "grp",
			expected: "/oauth2",
		},
		{
			name: "nested group - one level",
			groups: map[string]*RouterGroup{
				"grp":         {VarName: "grp", Path: "/oauth2", ParentVar: "router"},
				"deviceGroup": {VarName: "deviceGroup", Path: "/device", ParentVar: "grp"},
			},
			groupVar: "deviceGroup",
			expected: "/oauth2/device",
		},
		{
			name: "nested group - two levels",
			groups: map[string]*RouterGroup{
				"api":  {VarName: "api", Path: "/api", ParentVar: "router"},
				"v1":   {VarName: "v1", Path: "/v1", ParentVar: "api"},
				"auth": {VarName: "auth", Path: "/auth", ParentVar: "v1"},
			},
			groupVar: "auth",
			expected: "/api/v1/auth",
		},
		{
			name: "group with empty path",
			groups: map[string]*RouterGroup{
				"grp": {VarName: "grp", Path: "", ParentVar: "router"},
			},
			groupVar: "grp",
			expected: "",
		},
		{
			name: "nested group with empty parent path",
			groups: map[string]*RouterGroup{
				"grp": {VarName: "grp", Path: "", ParentVar: "router"},
				"sub": {VarName: "sub", Path: "/sub", ParentVar: "grp"},
			},
			groupVar: "sub",
			expected: "/sub",
		},
		{
			name: "path without leading slash",
			groups: map[string]*RouterGroup{
				"grp": {VarName: "grp", Path: "/api", ParentVar: "router"},
				"sub": {VarName: "sub", Path: "sub", ParentVar: "grp"},
			},
			groupVar: "sub",
			expected: "/api/sub",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &Introspector{}
			group := tt.groups[tt.groupVar]
			visited := make(map[string]bool)
			result := i.resolveGroupPath(group, tt.groups, visited)

			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestParseRoutesFileWithGroups(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected []RouteRegistration
	}{
		{
			name: "routes with group prefix",
			code: `package test
func RegisterRoutes(router forge.Router) {
	grp := router.Group("/oauth2")
	grp.POST("/register", h.RegisterClient)
	grp.GET("/jwks", h.JWKS)
}`,
			expected: []RouteRegistration{
				{Method: "POST", Path: "/oauth2/register", HandlerName: "RegisterClient"},
				{Method: "GET", Path: "/oauth2/jwks", HandlerName: "JWKS"},
			},
		},
		{
			name: "nested groups",
			code: `package test
func RegisterRoutes(router forge.Router) {
	grp := router.Group("/oauth2")
	grp.POST("/token", h.Token)
	
	deviceGroup := grp.Group("/device")
	deviceGroup.GET("", h.DeviceCodeEntry)
	deviceGroup.POST("/verify", h.DeviceVerify)
}`,
			expected: []RouteRegistration{
				{Method: "POST", Path: "/oauth2/token", HandlerName: "Token"},
				{Method: "GET", Path: "/oauth2/device", HandlerName: "DeviceCodeEntry"},
				{Method: "POST", Path: "/oauth2/device/verify", HandlerName: "DeviceVerify"},
			},
		},
		{
			name: "routes directly on router - no group",
			code: `package test
func RegisterRoutes(router forge.Router) {
	router.GET("/health", h.Health)
	router.POST("/login", h.Login)
}`,
			expected: []RouteRegistration{
				{Method: "GET", Path: "/health", HandlerName: "Health"},
				{Method: "POST", Path: "/login", HandlerName: "Login"},
			},
		},
		{
			name: "mixed - some with groups, some without",
			code: `package test
func RegisterRoutes(router forge.Router) {
	router.GET("/health", h.Health)
	
	api := router.Group("/api")
	api.GET("/users", h.ListUsers)
	api.POST("/users", h.CreateUser)
}`,
			expected: []RouteRegistration{
				{Method: "GET", Path: "/health", HandlerName: "Health"},
				{Method: "GET", Path: "/api/users", HandlerName: "ListUsers"},
				{Method: "POST", Path: "/api/users", HandlerName: "CreateUser"},
			},
		},
		{
			name: "multiple independent groups",
			code: `package test
func RegisterRoutes(router forge.Router) {
	templates := router.Group("/templates")
	templates.GET("", h.ListTemplates)
	
	notifications := router.Group("/notifications")
	notifications.POST("", h.SendNotification)
}`,
			expected: []RouteRegistration{
				{Method: "GET", Path: "/templates", HandlerName: "ListTemplates"},
				{Method: "POST", Path: "/notifications", HandlerName: "SendNotification"},
			},
		},
		{
			name: "deeply nested groups - three levels",
			code: `package test
func RegisterRoutes(router forge.Router) {
	cms := router.Group("/cms")
	types := cms.Group("/types")
	fields := types.Group("/:slug/fields")
	fields.POST("", h.CreateField)
	fields.DELETE("/:fieldId", h.DeleteField)
}`,
			expected: []RouteRegistration{
				{Method: "POST", Path: "/cms/types/:slug/fields", HandlerName: "CreateField"},
				{Method: "DELETE", Path: "/cms/types/:slug/fields/:fieldId", HandlerName: "DeleteField"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Write code to temp file
			fset := token.NewFileSet()

			node, err := parser.ParseFile(fset, "test.go", tt.code, parser.ParseComments)
			if err != nil {
				t.Fatalf("failed to parse code: %v", err)
			}

			i := &Introspector{fset: fset}

			// First pass: collect groups
			groupMap := make(map[string]*RouterGroup)

			ast.Inspect(node, func(n ast.Node) bool {
				if stmt, ok := n.(*ast.AssignStmt); ok {
					if group := i.extractGroupDeclaration(stmt); group != nil {
						groupMap[group.VarName] = group
					}
				}

				return true
			})

			// Resolve full paths
			for _, group := range groupMap {
				visited := make(map[string]bool)
				group.FullPath = i.resolveGroupPath(group, groupMap, visited)
			}

			// Second pass: extract routes
			var registrations []RouteRegistration

			ast.Inspect(node, func(n ast.Node) bool {
				if call, ok := n.(*ast.CallExpr); ok {
					if reg := i.extractRouteRegistrationWithGroups(call, groupMap); reg != nil {
						registrations = append(registrations, *reg)
					}
				}

				return true
			})

			if len(registrations) != len(tt.expected) {
				t.Fatalf("expected %d routes, got %d: %+v", len(tt.expected), len(registrations), registrations)
			}

			for idx, exp := range tt.expected {
				got := registrations[idx]
				if got.Method != exp.Method {
					t.Errorf("route %d: expected method %q, got %q", idx, exp.Method, got.Method)
				}

				if got.Path != exp.Path {
					t.Errorf("route %d: expected path %q, got %q", idx, exp.Path, got.Path)
				}

				if got.HandlerName != exp.HandlerName {
					t.Errorf("route %d: expected handler %q, got %q", idx, exp.HandlerName, got.HandlerName)
				}
			}
		})
	}
}

func TestGetReceiverName(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected string
	}{
		{
			name: "simple identifier receiver",
			code: `package test
func test() {
	grp.POST("/path", handler)
}`,
			expected: "grp",
		},
		{
			name: "router receiver",
			code: `package test
func test() {
	router.GET("/path", handler)
}`,
			expected: "router",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()

			node, err := parser.ParseFile(fset, "test.go", tt.code, 0)
			if err != nil {
				t.Fatalf("failed to parse code: %v", err)
			}

			i := &Introspector{fset: fset}

			var result string

			ast.Inspect(node, func(n ast.Node) bool {
				if call, ok := n.(*ast.CallExpr); ok {
					if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
						result = i.getReceiverName(sel)

						return false
					}
				}

				return true
			})

			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}
