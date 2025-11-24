package authsome

import (
	"context"
)

// Auto-generated context helper methods

// GetCurrentUser retrieves the current user from the session
func (c *Client) GetCurrentUser(ctx context.Context) (*User, error) {
	session, err := c.GetSession(ctx)
	if err != nil {
		return nil, err
	}
	return &session.User, nil
}

// GetCurrentSession retrieves the current session
func (c *Client) GetCurrentSession(ctx context.Context) (*Session, error) {
	session, err := c.GetSession(ctx)
	if err != nil {
		return nil, err
	}
	return &session.Session, nil
}

