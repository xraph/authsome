// Package apitypes holds the request and response types that have no fields of
// their own and are shared across the whole HTTP surface.
//
// They exist because an anonymous struct{} has no name, and a route signature
// is where the OpenAPI generator gets its names from. Every unnamed type in a
// spec comes out as "Object", so a handler written as
//
//	func(ctx forge.Context, _ *struct{}) (*struct{}, error)
//
// contributes nothing a client generator can turn into a type, and two of them
// in one document contend for the same component name. Naming the type once,
// here, gives every such route the same component and keeps the generated SDKs
// honest about what the endpoint takes and returns.
//
// The package deliberately imports nothing, so any part of the tree can use it.
package apitypes

// Empty is a body with no fields. Use it for a request that carries no body
// and for a response that answers with nothing but a status code.
//
// It marshals to {}, exactly as an anonymous struct{} does, so swapping one for
// the other changes the spec without touching the wire.
type Empty struct{}
