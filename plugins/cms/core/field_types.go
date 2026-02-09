package core

// FieldType defines the type of data a field holds.
type FieldType string

const (
	// FieldTypeText is a short text field (varchar).
	FieldTypeText FieldType = "text"
	// FieldTypeRichText is a long formatted text field.
	FieldTypeRichText FieldType = "richText"
	// FieldTypeNumber is a numeric field (integer or float).
	FieldTypeNumber FieldType = "number"
	// FieldTypeBoolean is a true/false field.
	FieldTypeBoolean FieldType = "boolean"
	// FieldTypeDate is a date-only field.
	FieldTypeDate FieldType = "date"
	// FieldTypeDateTime is a date and time field.
	FieldTypeDateTime FieldType = "datetime"
	// FieldTypeTime is a time-only field.
	FieldTypeTime FieldType = "time"
	// FieldTypeEmail is an email field with validation.
	FieldTypeEmail FieldType = "email"
	// FieldTypeURL is a URL field with validation.
	FieldTypeURL FieldType = "url"
	// FieldTypeJSON is an arbitrary JSON object/array field.
	FieldTypeJSON FieldType = "json"
	// FieldTypeSelect is a single select from options.
	FieldTypeSelect FieldType = "select"
	// FieldTypeMultiSelect is a multi-select from options.
	FieldTypeMultiSelect FieldType = "multiSelect"
	// FieldTypeRelation is a reference to another content type.
	FieldTypeRelation FieldType = "relation"
	// FieldTypeMedia is a file/image reference.
	FieldTypeMedia FieldType = "media"
	// FieldTypeSlug is a URL-friendly slug (auto-generated).
	FieldTypeSlug FieldType = "slug"
	// FieldTypeUUID is a UUID field.
	FieldTypeUUID FieldType = "uuid"
	// FieldTypeColor is a color picker field.
	FieldTypeColor FieldType = "color"
	// FieldTypePassword is a password field (hashed).
	FieldTypePassword FieldType = "password"
	// FieldTypePhone is a phone number field.
	FieldTypePhone FieldType = "phone"
	// FieldTypeTextarea is a multiline text field.
	FieldTypeTextarea FieldType = "textarea"
	// FieldTypeMarkdown is a markdown text field.
	FieldTypeMarkdown FieldType = "markdown"
	// FieldTypeEnumeration is an enumeration field.
	FieldTypeEnumeration FieldType = "enumeration"
	// FieldTypeInteger is an integer-only number field.
	FieldTypeInteger FieldType = "integer"
	// FieldTypeFloat is a float/decimal number field.
	FieldTypeFloat FieldType = "float"
	// FieldTypeBigInteger is a big integer field.
	FieldTypeBigInteger FieldType = "bigInteger"
	// FieldTypeDecimal is a decimal field with precision.
	FieldTypeDecimal FieldType = "decimal"
	// FieldTypeObject is a nested object with sub-fields.
	FieldTypeObject FieldType = "object"
	// FieldTypeArray is an array of objects with sub-fields.
	FieldTypeArray FieldType = "array"
	// FieldTypeOneOf is a discriminated union - schema determined by another field's value.
	FieldTypeOneOf FieldType = "oneOf"
)

// String returns the string representation of the field type.
func (t FieldType) String() string {
	return string(t)
}

// IsValid checks if the field type is valid.
func (t FieldType) IsValid() bool {
	switch t {
	case FieldTypeText, FieldTypeRichText, FieldTypeNumber, FieldTypeBoolean,
		FieldTypeDate, FieldTypeDateTime, FieldTypeTime, FieldTypeEmail,
		FieldTypeURL, FieldTypeJSON, FieldTypeSelect, FieldTypeMultiSelect,
		FieldTypeRelation, FieldTypeMedia, FieldTypeSlug, FieldTypeUUID,
		FieldTypeColor, FieldTypePassword, FieldTypePhone, FieldTypeTextarea,
		FieldTypeMarkdown, FieldTypeEnumeration, FieldTypeInteger, FieldTypeFloat,
		FieldTypeBigInteger, FieldTypeDecimal, FieldTypeObject, FieldTypeArray,
		FieldTypeOneOf:
		return true
	default:
		return false
	}
}

// ParseFieldType parses a string into a FieldType.
func ParseFieldType(s string) (FieldType, bool) {
	t := FieldType(s)
	if t.IsValid() {
		return t, true
	}

	return FieldTypeText, false
}

// IsNumeric returns true if the field type is numeric.
func (t FieldType) IsNumeric() bool {
	switch t {
	case FieldTypeNumber, FieldTypeInteger, FieldTypeFloat, FieldTypeBigInteger, FieldTypeDecimal:
		return true
	default:
		return false
	}
}

// IsText returns true if the field type is text-based.
func (t FieldType) IsText() bool {
	switch t {
	case FieldTypeText, FieldTypeRichText, FieldTypeTextarea, FieldTypeMarkdown,
		FieldTypeEmail, FieldTypeURL, FieldTypeSlug, FieldTypePhone, FieldTypeColor:
		return true
	default:
		return false
	}
}

// IsDate returns true if the field type is date-related.
func (t FieldType) IsDate() bool {
	switch t {
	case FieldTypeDate, FieldTypeDateTime, FieldTypeTime:
		return true
	default:
		return false
	}
}

// IsSelectable returns true if the field type has selectable options.
func (t FieldType) IsSelectable() bool {
	switch t {
	case FieldTypeSelect, FieldTypeMultiSelect, FieldTypeEnumeration:
		return true
	default:
		return false
	}
}

// RequiresOptions returns true if the field type requires options configuration.
func (t FieldType) RequiresOptions() bool {
	switch t {
	case FieldTypeSelect, FieldTypeMultiSelect, FieldTypeEnumeration, FieldTypeRelation,
		FieldTypeObject, FieldTypeArray, FieldTypeOneOf:
		return true
	default:
		return false
	}
}

// IsNested returns true if the field type supports nested sub-fields.
func (t FieldType) IsNested() bool {
	switch t {
	case FieldTypeObject, FieldTypeArray, FieldTypeOneOf:
		return true
	default:
		return false
	}
}

// IsOneOf returns true if the field type is a discriminated union.
func (t FieldType) IsOneOf() bool {
	return t == FieldTypeOneOf
}

// IsObject returns true if the field type is an object.
func (t FieldType) IsObject() bool {
	return t == FieldTypeObject
}

// IsArray returns true if the field type is an array.
func (t FieldType) IsArray() bool {
	return t == FieldTypeArray
}

// SupportsUnique returns true if the field type supports unique constraint.
func (t FieldType) SupportsUnique() bool {
	switch t {
	case FieldTypeText, FieldTypeEmail, FieldTypeURL, FieldTypeSlug, FieldTypeUUID,
		FieldTypePhone, FieldTypeNumber, FieldTypeInteger, FieldTypeFloat,
		FieldTypeBigInteger, FieldTypeDecimal:
		return true
	default:
		return false
	}
}

// SupportsIndex returns true if the field type supports indexing.
func (t FieldType) SupportsIndex() bool {
	switch t {
	case FieldTypeText, FieldTypeEmail, FieldTypeURL, FieldTypeSlug, FieldTypeUUID,
		FieldTypePhone, FieldTypeNumber, FieldTypeInteger, FieldTypeFloat,
		FieldTypeBigInteger, FieldTypeDecimal, FieldTypeDate, FieldTypeDateTime,
		FieldTypeBoolean, FieldTypeSelect, FieldTypeEnumeration:
		return true
	default:
		return false
	}
}

// SupportsSearch returns true if the field type supports full-text search.
func (t FieldType) SupportsSearch() bool {
	switch t {
	case FieldTypeText, FieldTypeRichText, FieldTypeTextarea, FieldTypeMarkdown:
		return true
	default:
		return false
	}
}

// FieldTypeInfo provides metadata about a field type.
type FieldTypeInfo struct {
	Type            FieldType `json:"type"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	Category        string    `json:"category"`
	Icon            string    `json:"icon"`
	SupportsUnique  bool      `json:"supportsUnique"`
	SupportsIndex   bool      `json:"supportsIndex"`
	SupportsSearch  bool      `json:"supportsSearch"`
	RequiresOptions bool      `json:"requiresOptions"`
}

// GetAllFieldTypes returns information about all available field types.
func GetAllFieldTypes() []FieldTypeInfo {
	return []FieldTypeInfo{
		{Type: FieldTypeText, Name: "Text", Description: "Short text field", Category: "text", Icon: "Type", SupportsUnique: true, SupportsIndex: true, SupportsSearch: true},
		{Type: FieldTypeTextarea, Name: "Textarea", Description: "Multiline text field", Category: "text", Icon: "AlignLeft", SupportsSearch: true},
		{Type: FieldTypeRichText, Name: "Rich Text", Description: "Formatted text with HTML", Category: "text", Icon: "FileText", SupportsSearch: true},
		{Type: FieldTypeMarkdown, Name: "Markdown", Description: "Markdown formatted text", Category: "text", Icon: "Hash", SupportsSearch: true},
		{Type: FieldTypeNumber, Name: "Number", Description: "Numeric value (integer or decimal)", Category: "number", Icon: "Hash", SupportsUnique: true, SupportsIndex: true},
		{Type: FieldTypeInteger, Name: "Integer", Description: "Whole number", Category: "number", Icon: "Binary", SupportsUnique: true, SupportsIndex: true},
		{Type: FieldTypeFloat, Name: "Float", Description: "Decimal number", Category: "number", Icon: "Percent", SupportsUnique: true, SupportsIndex: true},
		{Type: FieldTypeDecimal, Name: "Decimal", Description: "Precise decimal number", Category: "number", Icon: "DollarSign", SupportsUnique: true, SupportsIndex: true},
		{Type: FieldTypeBigInteger, Name: "Big Integer", Description: "Large whole number", Category: "number", Icon: "Infinity", SupportsUnique: true, SupportsIndex: true},
		{Type: FieldTypeBoolean, Name: "Boolean", Description: "True/False toggle", Category: "boolean", Icon: "ToggleLeft", SupportsIndex: true},
		{Type: FieldTypeDate, Name: "Date", Description: "Date only", Category: "date", Icon: "Calendar", SupportsIndex: true},
		{Type: FieldTypeDateTime, Name: "Date & Time", Description: "Date with time", Category: "date", Icon: "Clock", SupportsIndex: true},
		{Type: FieldTypeTime, Name: "Time", Description: "Time only", Category: "date", Icon: "Timer"},
		{Type: FieldTypeEmail, Name: "Email", Description: "Email address with validation", Category: "text", Icon: "Mail", SupportsUnique: true, SupportsIndex: true},
		{Type: FieldTypeURL, Name: "URL", Description: "Web URL with validation", Category: "text", Icon: "Link", SupportsUnique: true, SupportsIndex: true},
		{Type: FieldTypePhone, Name: "Phone", Description: "Phone number", Category: "text", Icon: "Phone", SupportsUnique: true, SupportsIndex: true},
		{Type: FieldTypeSlug, Name: "Slug", Description: "URL-friendly identifier", Category: "text", Icon: "Link2", SupportsUnique: true, SupportsIndex: true},
		{Type: FieldTypeUUID, Name: "UUID", Description: "Unique identifier", Category: "text", Icon: "Fingerprint", SupportsUnique: true, SupportsIndex: true},
		{Type: FieldTypeColor, Name: "Color", Description: "Color picker", Category: "text", Icon: "Palette"},
		{Type: FieldTypePassword, Name: "Password", Description: "Hashed password", Category: "text", Icon: "Lock"},
		{Type: FieldTypeJSON, Name: "JSON", Description: "Arbitrary JSON data", Category: "advanced", Icon: "Braces"},
		{Type: FieldTypeSelect, Name: "Single Select", Description: "Single choice from options", Category: "selection", Icon: "ChevronDown", SupportsIndex: true, RequiresOptions: true},
		{Type: FieldTypeMultiSelect, Name: "Multi Select", Description: "Multiple choices from options", Category: "selection", Icon: "CheckSquare", RequiresOptions: true},
		{Type: FieldTypeEnumeration, Name: "Enumeration", Description: "Predefined set of values", Category: "selection", Icon: "List", SupportsIndex: true, RequiresOptions: true},
		{Type: FieldTypeRelation, Name: "Relation", Description: "Reference to another content type", Category: "relation", Icon: "GitBranch", RequiresOptions: true},
		{Type: FieldTypeMedia, Name: "Media", Description: "File or image upload", Category: "media", Icon: "Image"},
		{Type: FieldTypeObject, Name: "Object", Description: "Nested object with sub-fields", Category: "nested", Icon: "Braces", RequiresOptions: true},
		{Type: FieldTypeArray, Name: "Array", Description: "Array of objects with sub-fields", Category: "nested", Icon: "List", RequiresOptions: true},
		{Type: FieldTypeOneOf, Name: "OneOf", Description: "Discriminated union - schema based on another field", Category: "nested", Icon: "GitMerge", RequiresOptions: true},
	}
}

// GetFieldTypeInfo returns information about a specific field type.
func GetFieldTypeInfo(t FieldType) *FieldTypeInfo {
	for _, info := range GetAllFieldTypes() {
		if info.Type == t {
			return &info
		}
	}

	return nil
}

// GetFieldTypesByCategory returns field types grouped by category.
func GetFieldTypesByCategory() map[string][]FieldTypeInfo {
	result := make(map[string][]FieldTypeInfo)
	for _, info := range GetAllFieldTypes() {
		result[info.Category] = append(result[info.Category], info)
	}

	return result
}
