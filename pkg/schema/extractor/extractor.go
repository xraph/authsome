package extractor

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"

	"github.com/xraph/authsome/pkg/schema/definition"
)

// Extractor extracts schema definitions from Go source files.
type Extractor struct {
	packagePath string
	schema      *definition.Schema
}

// NewExtractor creates a new schema extractor.
func NewExtractor(packagePath string) *Extractor {
	return &Extractor{
		packagePath: packagePath,
		schema: &definition.Schema{
			Version:     "1.0",
			Description: "AuthSome Database Schema",
			Models:      make(map[string]definition.Model),
		},
	}
}

// Extract extracts schema from Go files in the package.
func (e *Extractor) Extract() (*definition.Schema, error) {
	// Parse all Go files in the directory
	files, err := filepath.Glob(filepath.Join(e.packagePath, "*.go"))
	if err != nil {
		return nil, fmt.Errorf("failed to list Go files: %w", err)
	}

	for _, file := range files {
		// Skip base.go as it contains base models
		if filepath.Base(file) == "base.go" {
			continue
		}

		if err := e.extractFromFile(file); err != nil {
			return nil, fmt.Errorf("failed to extract from %s: %w", file, err)
		}
	}

	return e.schema, nil
}

// extractFromFile extracts schema from a single Go file.
func (e *Extractor) extractFromFile(filename string) error {
	fset := token.NewFileSet()

	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse file: %w", err)
	}

	// Iterate through declarations
	for _, decl := range node.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}

			// Extract model from struct
			model, err := e.extractModel(typeSpec.Name.Name, structType, genDecl.Doc)
			if err != nil {
				return fmt.Errorf("failed to extract model %s: %w", typeSpec.Name.Name, err)
			}

			if model != nil {
				e.schema.AddModel(*model)
			}
		}
	}

	return nil
}

// extractModel extracts a model from a struct type.
func (e *Extractor) extractModel(name string, structType *ast.StructType, doc *ast.CommentGroup) (*definition.Model, error) {
	model := definition.Model{
		Name:   name,
		Table:  toTableName(name),
		Fields: []definition.Field{},
	}

	// Extract description from comments
	if doc != nil {
		model.Description = strings.TrimSpace(doc.Text())
	}

	// Check if this is a table struct (has bun.BaseModel)
	hasBaseModel := false

	// First pass: check for embedded models and collect explicit field names
	explicitFields := make(map[string]bool)

	for _, field := range structType.Fields.List {
		if len(field.Names) > 0 {
			for _, fieldName := range field.Names {
				if ast.IsExported(fieldName.Name) {
					explicitFields[fieldName.Name] = true
				}
			}
		}
	}

	// Second pass: extract fields
	for _, field := range structType.Fields.List {
		// Skip embedded fields like bun.BaseModel
		if len(field.Names) == 0 {
			// Check for bun.BaseModel
			if sel, ok := field.Type.(*ast.SelectorExpr); ok {
				if x, ok := sel.X.(*ast.Ident); ok {
					if x.Name == "bun" && sel.Sel.Name == "BaseModel" {
						hasBaseModel = true
						// Extract table name from tag if present
						if field.Tag != nil {
							tagValue := strings.Trim(field.Tag.Value, "`")

							bunTag := extractTag(tagValue, "bun")
							if strings.HasPrefix(bunTag, "table:") {
								parts := strings.Split(bunTag, ",")
								if len(parts) > 0 {
									tableName := strings.TrimPrefix(parts[0], "table:")
									if tableName != "" {
										model.Table = tableName
									}
								}
							}
						}
					}
				}
			}

			// Check if it's AuditableModel or similar
			if ident, ok := field.Type.(*ast.Ident); ok {
				if ident.Name == "AuditableModel" {
					// Add auditable fields only if not explicitly overridden
					for _, auditField := range getAuditableFields() {
						if !explicitFields[auditField.Name] {
							model.Fields = append(model.Fields, auditField)
						}
					}

					continue
				}
			}

			continue
		}

		for _, fieldName := range field.Names {
			// Skip unexported fields
			if !ast.IsExported(fieldName.Name) {
				continue
			}

			extractedField, err := e.extractField(fieldName.Name, field)
			if err != nil {
				return nil, fmt.Errorf("failed to extract field %s: %w", fieldName.Name, err)
			}

			if extractedField != nil {
				model.Fields = append(model.Fields, *extractedField)
			}
		}
	}

	// Only return model if it has fields AND is a table struct
	if len(model.Fields) == 0 || !hasBaseModel {
		return nil, nil
	}

	return &model, nil
}

// extractField extracts a field from an AST field.
func (e *Extractor) extractField(name string, astField *ast.Field) (*definition.Field, error) {
	field := &definition.Field{
		Name:   name,
		Column: toColumnName(name),
	}

	// Extract type
	fieldType, nullable := e.extractType(astField.Type)
	field.Type = fieldType
	field.Nullable = nullable

	// Parse struct tags
	if astField.Tag != nil {
		tagValue := strings.Trim(astField.Tag.Value, "`")
		e.parseStructTags(field, tagValue)
	}

	// Extract description from comments
	if astField.Doc != nil {
		field.Description = strings.TrimSpace(astField.Doc.Text())
	}

	return field, nil
}

// extractType extracts the field type from an AST expression.
func (e *Extractor) extractType(expr ast.Expr) (definition.FieldType, bool) {
	nullable := false

	// Handle pointer types
	if starExpr, ok := expr.(*ast.StarExpr); ok {
		nullable = true
		expr = starExpr.X
	}

	// Handle basic types
	if ident, ok := expr.(*ast.Ident); ok {
		return goTypeToFieldType(ident.Name), nullable
	}

	// Handle selector types (e.g., time.Time, xid.ID)
	if sel, ok := expr.(*ast.SelectorExpr); ok {
		if x, ok := sel.X.(*ast.Ident); ok {
			fullType := x.Name + "." + sel.Sel.Name

			return goTypeToFieldType(fullType), nullable
		}
	}

	return definition.FieldTypeString, nullable
}

// parseStructTags parses Bun struct tags.
func (e *Extractor) parseStructTags(field *definition.Field, tag string) {
	// Extract bun tag
	bunTag := extractTag(tag, "bun")
	if bunTag == "" {
		return
	}

	parts := strings.Split(bunTag, ",")
	for i, part := range parts {
		part = strings.TrimSpace(part)

		// First part is usually the column name
		if i == 0 && !strings.Contains(part, ":") {
			if part != "" && part != "-" {
				field.Column = part
			}

			continue
		}

		// Parse constraints
		switch {
		case part == "pk":
			field.Primary = true
			field.Required = true
		case part == "notnull":
			field.Required = true
		case part == "unique":
			field.Unique = true
		case strings.HasPrefix(part, "type:"):
			// Type override from tag
			typeStr := strings.TrimPrefix(part, "type:")
			if strings.Contains(typeStr, "varchar") {
				field.Type = definition.FieldTypeString
				// Extract length
				if strings.Contains(typeStr, "(") {
					_, _ = fmt.Sscanf(typeStr, "varchar(%d)", &field.Length)
				}
			}
		case strings.HasPrefix(part, "default:"):
			defaultVal := strings.TrimPrefix(part, "default:")
			field.Default = defaultVal
		case part == "nullzero":
			// Bun-specific, means write zero values as null
			// Don't set nullable here, check notnull separately
		case strings.HasPrefix(part, "rel:"):
			// Relation tag - we'll handle this separately
		}
	}
}

// Helper functions

func extractTag(tags, key string) string {
	// Simple tag extraction (would need a proper parser for production)
	parts := strings.SplitSeq(tags, " ")
	for part := range parts {
		if after, ok := strings.CutPrefix(part, key+":"); ok {
			value := after
			value = strings.Trim(value, "\"")

			return value
		}
	}

	return ""
}

func toTableName(name string) string {
	// Convert CamelCase to snake_case and pluralize
	snake := toSnakeCase(name)
	// Simple pluralization
	if !strings.HasSuffix(snake, "s") {
		snake += "s"
	}

	return snake
}

func toColumnName(name string) string {
	return toSnakeCase(name)
}

func toSnakeCase(s string) string {
	var result strings.Builder

	for i, c := range s {
		if c >= 'A' && c <= 'Z' {
			if i > 0 {
				result.WriteRune('_')
			}

			result.WriteRune(c + 32) // Convert to lowercase
		} else {
			result.WriteRune(c)
		}
	}

	return result.String()
}

func goTypeToFieldType(goType string) definition.FieldType {
	switch goType {
	case "string":
		return definition.FieldTypeString
	case "int", "int32", "int64", "uint", "uint32", "uint64":
		return definition.FieldTypeInteger
	case "float32", "float64":
		return definition.FieldTypeFloat
	case "bool":
		return definition.FieldTypeBoolean
	case "time.Time":
		return definition.FieldTypeTimestamp
	case "xid.ID":
		return definition.FieldTypeString
	case "bun.NullTime":
		return definition.FieldTypeTimestamp
	default:
		return definition.FieldTypeString
	}
}

func getAuditableFields() []definition.Field {
	return []definition.Field{
		{
			Name:     "ID",
			Column:   "id",
			Type:     definition.FieldTypeString,
			Primary:  true,
			Required: true,
			Length:   20,
		},
		{
			Name:     "CreatedAt",
			Column:   "created_at",
			Type:     definition.FieldTypeTimestamp,
			Required: true,
			Default:  "current_timestamp",
			AutoGen:  true,
		},
		{
			Name:     "CreatedBy",
			Column:   "created_by",
			Type:     definition.FieldTypeString,
			Required: true,
			Length:   20,
		},
		{
			Name:     "UpdatedAt",
			Column:   "updated_at",
			Type:     definition.FieldTypeTimestamp,
			Required: true,
			Default:  "current_timestamp",
			AutoGen:  true,
		},
		{
			Name:     "UpdatedBy",
			Column:   "updated_by",
			Type:     definition.FieldTypeString,
			Required: true,
			Length:   20,
		},
		{
			Name:     "Version",
			Column:   "version",
			Type:     definition.FieldTypeInteger,
			Required: true,
			Default:  1,
		},
	}
}
