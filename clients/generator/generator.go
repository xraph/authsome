package generator

import (
	"fmt"
	"path/filepath"

	"github.com/xraph/authsome/clients/manifest"
)

// Language represents a target language for code generation
type Language string

const (
	LanguageGo         Language = "go"
	LanguageTypeScript Language = "typescript"
	LanguageRust       Language = "rust"
)

// Generator coordinates code generation for all languages
type Generator struct {
	parser    *manifest.Parser
	outputDir string
}

// NewGenerator creates a new generator
func NewGenerator(manifestDir, outputDir string) (*Generator, error) {
	parser := manifest.NewParser()
	if err := parser.LoadDirectory(manifestDir); err != nil {
		return nil, fmt.Errorf("failed to load manifests: %w", err)
	}

	return &Generator{
		parser:    parser,
		outputDir: outputDir,
	}, nil
}

// GenerateAll generates clients for all languages
func (g *Generator) GenerateAll() error {
	languages := []Language{LanguageGo, LanguageTypeScript, LanguageRust}
	for _, lang := range languages {
		if err := g.Generate(lang, nil); err != nil {
			return fmt.Errorf("failed to generate %s client: %w", lang, err)
		}
	}
	return nil
}

// Generate generates a client for a specific language
func (g *Generator) Generate(lang Language, pluginIDs []string) error {
	// Get manifests to generate
	var manifests []*manifest.Manifest
	if len(pluginIDs) > 0 {
		manifests = g.parser.Filter(pluginIDs)
	} else {
		manifests = g.parser.List()
	}

	if len(manifests) == 0 {
		return fmt.Errorf("no manifests found")
	}

	// Ensure core is included
	hasCore := false
	for _, m := range manifests {
		if m.PluginID == "core" {
			hasCore = true
			break
		}
	}
	if !hasCore {
		core, err := g.parser.GetCore()
		if err != nil {
			return err
		}
		manifests = append([]*manifest.Manifest{core}, manifests...)
	}

	// Generate based on language
	outputDir := filepath.Join(g.outputDir, string(lang))

	switch lang {
	case LanguageGo:
		gen := NewGoGenerator(outputDir, manifests)
		return gen.Generate()

	case LanguageTypeScript:
		gen := NewTypeScriptGenerator(outputDir, manifests)
		return gen.Generate()

	case LanguageRust:
		gen := NewRustGenerator(outputDir, manifests)
		return gen.Generate()

	default:
		return fmt.Errorf("unsupported language: %s", lang)
	}
}

// ListPlugins returns all available plugin IDs
func (g *Generator) ListPlugins() []string {
	manifests := g.parser.List()
	plugins := make([]string, 0, len(manifests))
	for _, m := range manifests {
		plugins = append(plugins, m.PluginID)
	}
	return plugins
}

// ValidateManifests validates all loaded manifests
func (g *Generator) ValidateManifests() error {
	for _, m := range g.parser.List() {
		if err := m.Validate(); err != nil {
			return fmt.Errorf("manifest %s is invalid: %w", m.PluginID, err)
		}
	}
	return nil
}
