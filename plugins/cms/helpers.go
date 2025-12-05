package cms

import (
	"fmt"

	"github.com/xraph/authsome/plugins/cms/service"
	"github.com/xraph/forge"
)

// Service name constants for DI container registration
const (
	ServiceNamePlugin                 = "cms.plugin"
	ServiceNameContentTypeService     = "cms.content_type_service"
	ServiceNameFieldService           = "cms.field_service"
	ServiceNameEntryService           = "cms.entry_service"
	ServiceNameRevisionService        = "cms.revision_service"
	ServiceNameComponentSchemaService = "cms.component_schema_service"
)

// ResolveCMSPlugin resolves the CMS plugin from the container
func ResolveCMSPlugin(container forge.Container) (*Plugin, error) {
	resolved, err := container.Resolve(ServiceNamePlugin)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve CMS plugin: %w", err)
	}
	plugin, ok := resolved.(*Plugin)
	if !ok {
		return nil, fmt.Errorf("invalid CMS plugin type")
	}
	return plugin, nil
}

// ResolveContentTypeService resolves the content type service from the container
func ResolveContentTypeService(container forge.Container) (*service.ContentTypeService, error) {
	resolved, err := container.Resolve(ServiceNameContentTypeService)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve content type service: %w", err)
	}
	svc, ok := resolved.(*service.ContentTypeService)
	if !ok {
		return nil, fmt.Errorf("invalid content type service type")
	}
	return svc, nil
}

// ResolveFieldService resolves the content field service from the container
func ResolveFieldService(container forge.Container) (*service.ContentFieldService, error) {
	resolved, err := container.Resolve(ServiceNameFieldService)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve field service: %w", err)
	}
	svc, ok := resolved.(*service.ContentFieldService)
	if !ok {
		return nil, fmt.Errorf("invalid field service type")
	}
	return svc, nil
}

// ResolveEntryService resolves the content entry service from the container
func ResolveEntryService(container forge.Container) (*service.ContentEntryService, error) {
	resolved, err := container.Resolve(ServiceNameEntryService)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve entry service: %w", err)
	}
	svc, ok := resolved.(*service.ContentEntryService)
	if !ok {
		return nil, fmt.Errorf("invalid entry service type")
	}
	return svc, nil
}

// ResolveRevisionService resolves the revision service from the container
func ResolveRevisionService(container forge.Container) (*service.RevisionService, error) {
	resolved, err := container.Resolve(ServiceNameRevisionService)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve revision service: %w", err)
	}
	svc, ok := resolved.(*service.RevisionService)
	if !ok {
		return nil, fmt.Errorf("invalid revision service type")
	}
	return svc, nil
}

// ResolveComponentSchemaService resolves the component schema service from the container
func ResolveComponentSchemaService(container forge.Container) (*service.ComponentSchemaService, error) {
	resolved, err := container.Resolve(ServiceNameComponentSchemaService)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve component schema service: %w", err)
	}
	svc, ok := resolved.(*service.ComponentSchemaService)
	if !ok {
		return nil, fmt.Errorf("invalid component schema service type")
	}
	return svc, nil
}

// RegisterServices registers all CMS services in the DI container
func (p *Plugin) RegisterServices(container forge.Container) error {
	// Register plugin itself
	if err := container.Register(ServiceNamePlugin, func(_ forge.Container) (any, error) {
		return p, nil
	}); err != nil {
		return fmt.Errorf("failed to register CMS plugin: %w", err)
	}

	// Register content type service
	if err := container.Register(ServiceNameContentTypeService, func(_ forge.Container) (any, error) {
		if p.contentTypeSvc == nil {
			return nil, fmt.Errorf("content type service not initialized")
		}
		return p.contentTypeSvc, nil
	}); err != nil {
		return fmt.Errorf("failed to register content type service: %w", err)
	}

	// Register field service
	if err := container.Register(ServiceNameFieldService, func(_ forge.Container) (any, error) {
		if p.fieldSvc == nil {
			return nil, fmt.Errorf("field service not initialized")
		}
		return p.fieldSvc, nil
	}); err != nil {
		return fmt.Errorf("failed to register field service: %w", err)
	}

	// Register entry service
	if err := container.Register(ServiceNameEntryService, func(_ forge.Container) (any, error) {
		if p.entrySvc == nil {
			return nil, fmt.Errorf("entry service not initialized")
		}
		return p.entrySvc, nil
	}); err != nil {
		return fmt.Errorf("failed to register entry service: %w", err)
	}

	// Register revision service
	if err := container.Register(ServiceNameRevisionService, func(_ forge.Container) (any, error) {
		if p.revisionSvc == nil {
			return nil, fmt.Errorf("revision service not initialized")
		}
		return p.revisionSvc, nil
	}); err != nil {
		return fmt.Errorf("failed to register revision service: %w", err)
	}

	// Register component schema service
	if err := container.Register(ServiceNameComponentSchemaService, func(_ forge.Container) (any, error) {
		if p.componentSchemaSvc == nil {
			return nil, fmt.Errorf("component schema service not initialized")
		}
		return p.componentSchemaSvc, nil
	}); err != nil {
		return fmt.Errorf("failed to register component schema service: %w", err)
	}

	return nil
}

// GetServices returns a map of all available services for inspection
func (p *Plugin) GetServices() map[string]interface{} {
	return map[string]interface{}{
		"contentTypeService":     p.contentTypeSvc,
		"fieldService":           p.fieldSvc,
		"entryService":           p.entrySvc,
		"revisionService":        p.revisionSvc,
		"componentSchemaService": p.componentSchemaSvc,
	}
}

// GetContentTypeService returns the content type service directly
func (p *Plugin) GetContentTypeService() *service.ContentTypeService {
	return p.contentTypeSvc
}

// GetFieldService returns the content field service directly
func (p *Plugin) GetFieldService() *service.ContentFieldService {
	return p.fieldSvc
}

// GetEntryService returns the content entry service directly
func (p *Plugin) GetEntryService() *service.ContentEntryService {
	return p.entrySvc
}

// GetRevisionService returns the revision service directly
func (p *Plugin) GetRevisionService() *service.RevisionService {
	return p.revisionSvc
}

// GetComponentSchemaService returns the component schema service directly
func (p *Plugin) GetComponentSchemaService() *service.ComponentSchemaService {
	return p.componentSchemaSvc
}

