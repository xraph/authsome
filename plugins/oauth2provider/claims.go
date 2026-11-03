package oauth2provider

import "github.com/xraph/authsome/user"

// StandardClaims are the OpenID Connect Core 5.1 claims authsome can fill from
// a user. Userinfo and id_tokens share this one shape, so a claim cannot be
// present in one and missing from the other. The verified flags are pointers
// so a false value is still written when its scope is granted. They point
// at copies, never into the user.
type StandardClaims struct {
	Name                string `json:"name,omitempty"`
	GivenName           string `json:"given_name,omitempty"`
	FamilyName          string `json:"family_name,omitempty"`
	Picture             string `json:"picture,omitempty"`
	PreferredUsername   string `json:"preferred_username,omitempty"`
	UpdatedAt           int64  `json:"updated_at,omitempty"`
	Email               string `json:"email,omitempty"`
	EmailVerified       *bool  `json:"email_verified,omitempty"`
	PhoneNumber         string `json:"phone_number,omitempty"`
	PhoneNumberVerified *bool  `json:"phone_number_verified,omitempty"`
}

// standardClaims returns the claims u's granted scopes cover (OpenID Connect
// Core 5.4). has reports whether a scope is granted.
func standardClaims(u *user.User, has func(scope string) bool) StandardClaims {
	var c StandardClaims
	if has("profile") {
		c.Name = u.Name()
		c.GivenName = u.FirstName
		c.FamilyName = u.LastName
		c.Picture = u.Image
		c.PreferredUsername = u.Username
		if !u.UpdatedAt.IsZero() {
			c.UpdatedAt = u.UpdatedAt.Unix()
		}
	}
	if has("email") {
		c.Email = u.Email
		verified := u.EmailVerified
		c.EmailVerified = &verified
	}
	if has("phone") {
		c.PhoneNumber = u.Phone
		verified := u.PhoneVerified
		c.PhoneNumberVerified = &verified
	}
	return c
}

// supportedClaims lists the claims discovery advertises: the id_token
// envelope plus every StandardClaims field.
func supportedClaims() []string {
	return []string{
		"sub", "iss", "aud", "exp", "iat", "auth_time", "nonce", "azp", "at_hash",
		"name", "given_name", "family_name", "picture", "preferred_username", "updated_at",
		"email", "email_verified", "phone_number", "phone_number_verified",
	}
}
