package authsome

// Auto-generated plugin interface

// Plugin defines the interface for client plugins
type Plugin interface {
	// ID returns the unique plugin identifier
	ID() string
	
	// Init initializes the plugin with the client
	Init(client *Client) error
}
