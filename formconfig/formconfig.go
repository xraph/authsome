// Package formconfig defines dynamic form configuration types for authsome.
//
// It supports per-app signup form schemas with custom fields, validation rules,
// and field rendering hints. Custom field values are stored in User.Metadata.
package formconfig

import (
	"time"

	"github.com/xraph/authsome/id"
)

// FieldType represents the type of a form field.
type FieldType string

// Supported field types.
const (
	FieldText     FieldType = "text"
	FieldEmail    FieldType = "email"
	FieldNumber   FieldType = "number"
	FieldTel      FieldType = "tel"
	FieldURL      FieldType = "url"
	FieldDate     FieldType = "date"
	FieldTextarea FieldType = "textarea"
	FieldSelect   FieldType = "select"
	FieldCheckbox FieldType = "checkbox"
	FieldRadio       FieldType = "radio"
	FieldSwitch      FieldType = "switch"
	FieldObjectArray FieldType = "object_array"
)

// SelectOption represents a single option for select, radio, or checkbox fields.
type SelectOption struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// Validation defines server-side and client-side validation rules for a form field.
type Validation struct {
	Required bool   `json:"required,omitempty"`
	MinLen   int    `json:"min_len,omitempty"`
	MaxLen   int    `json:"max_len,omitempty"`
	Pattern  string `json:"pattern,omitempty"` // regex pattern
	Min      *int   `json:"min,omitempty"`     // for number fields
	Max      *int   `json:"max,omitempty"`     // for number fields
}

// FormField defines a single custom field in a form configuration.
type FormField struct {
	Key         string         `json:"key"`                   // stored as key in User.Metadata
	Label       string         `json:"label"`
	Type        FieldType      `json:"type"`
	Placeholder string         `json:"placeholder,omitempty"`
	Description string         `json:"description,omitempty"`
	Options     []SelectOption `json:"options,omitempty"` // for select/radio/checkbox
	Default     string         `json:"default,omitempty"`
	Validation  Validation     `json:"validation,omitempty"`
	Order       int            `json:"order"`
}

// FormConfig defines a dynamic form schema for a specific app and form type.
type FormConfig struct {
	ID        id.FormConfigID `json:"id"`
	AppID     id.AppID        `json:"app_id"`
	FormType  string          `json:"form_type"` // "signup", future: "profile_edit"
	Fields    []FormField     `json:"fields"`
	Active    bool            `json:"active"`
	Version   int             `json:"version"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// FormTypeSignup is the form type for signup forms.
const FormTypeSignup = "signup"
