// Package apikey provides an API key authentication plugin for AuthSome.
//
// The apikey plugin enables machine-to-machine authentication. API keys are
// created per-user, hashed with SHA-256, and identified by a visible prefix
// during lookup. The plugin registers HTTP routes for key lifecycle management
// and an authentication strategy that checks the Authorization or X-API-Key headers.
package apikey
