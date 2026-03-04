// Package environment defines the environment domain entity and its store interface.
// Environments partition an App into isolated contexts (development, staging, production).
// Each App has one or more environments; all user-facing entities (users, sessions,
// organizations, etc.) are scoped to an environment.
package environment

import (
	"time"

	"github.com/xraph/authsome/id"
)

// Type identifies the kind of environment.
type Type string

const (
	TypeDevelopment Type = "development"
	TypeStaging     Type = "staging"
	TypeProduction  Type = "production"
)

// ValidTypes is the set of allowed environment types.
var ValidTypes = []Type{TypeDevelopment, TypeStaging, TypeProduction}

// IsValid reports whether t is a recognized environment type.
func (t Type) IsValid() bool {
	switch t {
	case TypeDevelopment, TypeStaging, TypeProduction:
		return true
	default:
		return false
	}
}

// String returns the string representation of the environment type.
func (t Type) String() string { return string(t) }

// DefaultColor returns the default UI badge color for the environment type.
func (t Type) DefaultColor() string {
	switch t {
	case TypeDevelopment:
		return "#22c55e" // green
	case TypeStaging:
		return "#eab308" // yellow
	case TypeProduction:
		return "#ef4444" // red
	default:
		return "#6b7280" // gray
	}
}

// DefaultName returns the default display name for the environment type.
func (t Type) DefaultName() string {
	switch t {
	case TypeDevelopment:
		return "Development"
	case TypeStaging:
		return "Staging"
	case TypeProduction:
		return "Production"
	default:
		return string(t)
	}
}

// Environment represents an isolated context within an App.
// Each App has one or more environments. All user-facing entities
// (users, sessions, organizations, etc.) are scoped to an environment.
type Environment struct {
	ID          id.EnvironmentID `json:"id"`
	AppID       id.AppID         `json:"app_id"`
	Name        string           `json:"name"`
	Slug        string           `json:"slug"`
	Type        Type             `json:"type"`
	IsDefault   bool             `json:"is_default"`
	Color       string           `json:"color,omitempty"`
	Description string           `json:"description,omitempty"`
	Settings    *Settings        `json:"settings,omitempty"`
	ClonedFrom  id.EnvironmentID `json:"cloned_from,omitempty"`
	Metadata    Metadata         `json:"metadata,omitempty"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
}

// Metadata holds arbitrary environment metadata as typed key-value pairs.
type Metadata map[string]string

// EnvironmentQuery holds query parameters for listing environments.
type EnvironmentQuery struct {
	AppID id.AppID `json:"app_id"`
	Type  Type     `json:"type,omitempty"`
}

// EnvironmentList is a list of environments for an app.
type EnvironmentList struct {
	Environments []*Environment `json:"environments"`
	Total        int            `json:"total"`
}
