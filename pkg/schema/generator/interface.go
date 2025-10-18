package generator

import "github.com/xraph/authsome/pkg/schema/definition"

// Generator is the interface that all schema generators must implement
type Generator interface {
	// Generate generates files from the schema
	Generate(schema *definition.Schema, opts Options) error

	// Name returns the generator name
	Name() string

	// Description returns a description of what this generator produces
	Description() string
}

// Options contains common options for all generators
type Options struct {
	// OutputDir is the directory where generated files will be written
	OutputDir string

	// Dialect specifies the database dialect (postgres, mysql, sqlite, etc.)
	Dialect string

	// PackageName is the Go package name for generated code
	PackageName string

	// Overwrite determines if existing files should be overwritten
	Overwrite bool

	// Verbose enables verbose output
	Verbose bool

	// Custom contains generator-specific options
	Custom map[string]interface{}
}

// MigrationFile represents a generated migration file
type MigrationFile struct {
	Name    string
	Path    string
	Content string
}

// Result represents the result of a generation operation
type Result struct {
	Files   []MigrationFile
	Message string
	Errors  []error
}
