package schema

import (
	"context"
	"fmt"
	"maps"
	"sort"
	"sync"

	"github.com/xraph/authsome/internal/errs"
)

// SettingsProvider is the interface for plugins that provide settings sections.
type SettingsProvider interface {
	// ProviderID returns the unique identifier for this provider
	ProviderID() string
	// GetSettingsSections returns the settings sections provided by this plugin
	GetSettingsSections() []*Section
	// GetSettingsSchema returns the complete schema (optional, can return nil)
	GetSettingsSchema() *Schema
}

// DynamicSettingsProvider is an extended interface for providers that generate
// schemas dynamically based on context (e.g., per-app configuration).
type DynamicSettingsProvider interface {
	SettingsProvider
	// GetDynamicSections returns sections that may vary based on context
	GetDynamicSections(ctx context.Context, appID string) ([]*Section, error)
}

// SettingsValidator is an optional interface for providers that need async validation.
type SettingsValidator interface {
	// ValidateSettings validates settings data for this provider
	ValidateSettings(ctx context.Context, appID string, sectionID string, data map[string]any) *ValidationResult
}

// SettingsMigrator is an optional interface for providers that need to migrate settings.
type SettingsMigrator interface {
	// MigrateSettings migrates settings from one version to another
	MigrateSettings(ctx context.Context, appID string, data map[string]any, fromVersion, toVersion int) (map[string]any, error)
}

// Registry manages schema sections and providers.
type Registry struct {
	mu        sync.RWMutex
	sections  map[string]*Section
	providers map[string]SettingsProvider
	order     []string // Maintains insertion order for sections
}

// NewRegistry creates a new schema registry.
func NewRegistry() *Registry {
	return &Registry{
		sections:  make(map[string]*Section),
		providers: make(map[string]SettingsProvider),
		order:     make([]string, 0),
	}
}

// RegisterSection registers a standalone section.
func (r *Registry) RegisterSection(section *Section) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if section == nil {
		return errs.RequiredField("section")
	}

	if section.ID == "" {
		return errs.RequiredField("section.id")
	}

	if _, exists := r.sections[section.ID]; exists {
		return fmt.Errorf("section with ID '%s' already registered", section.ID)
	}

	r.sections[section.ID] = section
	r.order = append(r.order, section.ID)

	return nil
}

// RegisterSections registers multiple sections at once.
func (r *Registry) RegisterSections(sections ...*Section) error {
	for _, section := range sections {
		if err := r.RegisterSection(section); err != nil {
			return err
		}
	}

	return nil
}

// UnregisterSection removes a section from the registry.
func (r *Registry) UnregisterSection(sectionID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.sections, sectionID)
	// Remove from order
	for i, id := range r.order {
		if id == sectionID {
			r.order = append(r.order[:i], r.order[i+1:]...)

			break
		}
	}
}

// RegisterProvider registers a settings provider.
func (r *Registry) RegisterProvider(provider SettingsProvider) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if provider == nil {
		return errs.RequiredField("provider")
	}

	providerID := provider.ProviderID()
	if providerID == "" {
		return errs.RequiredField("provider.id")
	}

	if _, exists := r.providers[providerID]; exists {
		return fmt.Errorf("provider with ID '%s' already registered", providerID)
	}

	r.providers[providerID] = provider

	// Register all sections from the provider
	for _, section := range provider.GetSettingsSections() {
		// Prefix section ID with provider ID to avoid conflicts
		if section.Metadata == nil {
			section.Metadata = make(map[string]any)
		}

		section.Metadata["providerID"] = providerID

		// Use the section ID as-is (providers should use unique IDs)
		r.sections[section.ID] = section
		r.order = append(r.order, section.ID)
	}

	return nil
}

// UnregisterProvider removes a provider and its sections from the registry.
func (r *Registry) UnregisterProvider(providerID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.providers, providerID)

	// Remove all sections from this provider
	var remainingOrder []string

	for id, section := range r.sections {
		if section.Metadata != nil {
			if pid, ok := section.Metadata["providerID"].(string); ok && pid == providerID {
				delete(r.sections, id)

				continue
			}
		}

		remainingOrder = append(remainingOrder, id)
	}

	r.order = remainingOrder
}

// GetSection returns a section by ID.
func (r *Registry) GetSection(sectionID string) *Section {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.sections[sectionID]
}

// GetProvider returns a provider by ID.
func (r *Registry) GetProvider(providerID string) SettingsProvider {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.providers[providerID]
}

// ListSections returns all registered sections.
func (r *Registry) ListSections() []*Section {
	r.mu.RLock()
	defer r.mu.RUnlock()

	sections := make([]*Section, 0, len(r.sections))
	for _, id := range r.order {
		if section, ok := r.sections[id]; ok {
			sections = append(sections, section)
		}
	}

	return sections
}

// GetSortedSections returns sections sorted by order.
func (r *Registry) GetSortedSections() []*Section {
	sections := r.ListSections()
	sort.Slice(sections, func(i, j int) bool {
		return sections[i].Order < sections[j].Order
	})

	return sections
}

// ListProviders returns all registered providers.
func (r *Registry) ListProviders() []SettingsProvider {
	r.mu.RLock()
	defer r.mu.RUnlock()

	providers := make([]SettingsProvider, 0, len(r.providers))
	for _, provider := range r.providers {
		providers = append(providers, provider)
	}

	return providers
}

// GetSchema builds a complete schema from all registered sections.
func (r *Registry) GetSchema(schemaID, schemaName string) *Schema {
	r.mu.RLock()
	defer r.mu.RUnlock()

	schema := NewSchema(schemaID, schemaName)
	schema.Sections = r.GetSortedSections()

	return schema
}

// GetDynamicSchema builds a schema including dynamic sections from providers.
func (r *Registry) GetDynamicSchema(ctx context.Context, appID, schemaID, schemaName string) (*Schema, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	schema := NewSchema(schemaID, schemaName)

	// Add static sections first
	schema.Sections = append(schema.Sections, r.GetSortedSections()...)

	// Add dynamic sections from providers
	for _, provider := range r.providers {
		if dynamicProvider, ok := provider.(DynamicSettingsProvider); ok {
			dynamicSections, err := dynamicProvider.GetDynamicSections(ctx, appID)
			if err != nil {
				return nil, fmt.Errorf("failed to get dynamic sections from provider %s: %w", provider.ProviderID(), err)
			}

			schema.Sections = append(schema.Sections, dynamicSections...)
		}
	}

	// Re-sort by order
	sort.Slice(schema.Sections, func(i, j int) bool {
		return schema.Sections[i].Order < schema.Sections[j].Order
	})

	return schema, nil
}

// ValidateSection validates data for a specific section.
func (r *Registry) ValidateSection(ctx context.Context, sectionID string, data map[string]any) *ValidationResult {
	section := r.GetSection(sectionID)
	if section == nil {
		result := NewValidationResult()
		result.AddGlobalError(fmt.Sprintf("section '%s' not found", sectionID))

		return result
	}

	return section.Validate(ctx, data)
}

// ValidateSchema validates data for the entire schema.
func (r *Registry) ValidateSchema(ctx context.Context, data map[string]map[string]any) *ValidationResult {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := NewValidationResult()

	for sectionID, sectionData := range data {
		section, ok := r.sections[sectionID]
		if !ok {
			result.AddGlobalError("unknown section: " + sectionID)

			continue
		}

		sectionResult := section.Validate(ctx, sectionData)
		result.Merge(sectionResult)
	}

	return result
}

// ValidateWithProviders runs provider-specific validation.
func (r *Registry) ValidateWithProviders(ctx context.Context, appID, sectionID string, data map[string]any) *ValidationResult {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// First run standard section validation
	result := r.ValidateSection(ctx, sectionID, data)
	if result.HasErrors() {
		return result
	}

	// Then run provider-specific validation
	section := r.sections[sectionID]
	if section != nil && section.Metadata != nil {
		if providerID, ok := section.Metadata["providerID"].(string); ok {
			if provider, ok := r.providers[providerID]; ok {
				if validator, ok := provider.(SettingsValidator); ok {
					providerResult := validator.ValidateSettings(ctx, appID, sectionID, data)
					result.Merge(providerResult)
				}
			}
		}
	}

	return result
}

// GetDefaults returns default values for all sections.
func (r *Registry) GetDefaults() map[string]map[string]any {
	r.mu.RLock()
	defer r.mu.RUnlock()

	defaults := make(map[string]map[string]any)
	for id, section := range r.sections {
		defaults[id] = section.GetDefaults()
	}

	return defaults
}

// GetSectionDefaults returns default values for a specific section.
func (r *Registry) GetSectionDefaults(sectionID string) map[string]any {
	section := r.GetSection(sectionID)
	if section == nil {
		return nil
	}

	return section.GetDefaults()
}

// MergeWithDefaults merges provided data with default values.
func (r *Registry) MergeWithDefaults(data map[string]map[string]any) map[string]map[string]any {
	defaults := r.GetDefaults()

	result := make(map[string]map[string]any)

	// Start with defaults
	for sectionID, sectionDefaults := range defaults {
		result[sectionID] = make(map[string]any)
		maps.Copy(result[sectionID], sectionDefaults)
	}

	// Overlay provided data
	for sectionID, sectionData := range data {
		if result[sectionID] == nil {
			result[sectionID] = make(map[string]any)
		}

		maps.Copy(result[sectionID], sectionData)
	}

	return result
}

// Clone creates a deep copy of the registry.
func (r *Registry) Clone() *Registry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	clone := NewRegistry()

	for id, section := range r.sections {
		clone.sections[id] = section.Clone()
	}

	clone.order = make([]string, len(r.order))
	copy(clone.order, r.order)

	// Note: Providers are not cloned as they contain business logic
	maps.Copy(clone.providers, r.providers)

	return clone
}

// Clear removes all sections and providers from the registry.
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.sections = make(map[string]*Section)
	r.providers = make(map[string]SettingsProvider)
	r.order = make([]string, 0)
}

// Stats returns registry statistics.
func (r *Registry) Stats() RegistryStats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	totalFields := 0
	for _, section := range r.sections {
		totalFields += len(section.Fields)
	}

	return RegistryStats{
		SectionCount:  len(r.sections),
		ProviderCount: len(r.providers),
		FieldCount:    totalFields,
	}
}

// RegistryStats contains registry statistics.
type RegistryStats struct {
	SectionCount  int `json:"sectionCount"`
	ProviderCount int `json:"providerCount"`
	FieldCount    int `json:"fieldCount"`
}

// Global registry instance.
var globalRegistry = NewRegistry()

// Global returns the global registry instance.
func Global() *Registry {
	return globalRegistry
}

// RegisterSection registers a section in the global registry.
func RegisterSection(section *Section) error {
	return globalRegistry.RegisterSection(section)
}

// RegisterProvider registers a provider in the global registry.
func RegisterProvider(provider SettingsProvider) error {
	return globalRegistry.RegisterProvider(provider)
}

// GetGlobalSection returns a section from the global registry.
func GetGlobalSection(sectionID string) *Section {
	return globalRegistry.GetSection(sectionID)
}

// GetGlobalSchema returns the complete schema from the global registry.
func GetGlobalSchema(schemaID, schemaName string) *Schema {
	return globalRegistry.GetSchema(schemaID, schemaName)
}
