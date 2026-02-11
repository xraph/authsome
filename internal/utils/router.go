package utils

import "github.com/xraph/forge"

// ExtensionRouter wraps the base router and auto-applies exclusion.
type ExtensionRouter struct {
	forge.Router

	ext forge.Extension
}

func (r *ExtensionRouter) POST(path string, handler any, opts ...forge.RouteOption) error {
	// Prepend extension exclusion options
	allOpts := forge.ExtensionRoutes(r.ext, opts...)

	return r.Router.POST(path, handler, allOpts...)
}

func (r *ExtensionRouter) GET(path string, handler any, opts ...forge.RouteOption) error {
	allOpts := forge.ExtensionRoutes(r.ext, opts...)

	return r.Router.GET(path, handler, allOpts...)
}

func (r *ExtensionRouter) PUT(path string, handler any, opts ...forge.RouteOption) error {
	allOpts := forge.ExtensionRoutes(r.ext, opts...)

	return r.Router.PUT(path, handler, allOpts...)
}

func (r *ExtensionRouter) DELETE(path string, handler any, opts ...forge.RouteOption) error {
	allOpts := forge.ExtensionRoutes(r.ext, opts...)

	return r.Router.DELETE(path, handler, allOpts...)
}

func (r *ExtensionRouter) PATCH(path string, handler any, opts ...forge.RouteOption) error {
	allOpts := forge.ExtensionRoutes(r.ext, opts...)

	return r.Router.PATCH(path, handler, allOpts...)
}
