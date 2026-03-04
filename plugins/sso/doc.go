// Package sso provides enterprise single sign-on authentication for AuthSome.
//
// The SSO plugin enables organizations to authenticate users through SAML 2.0
// and OpenID Connect (OIDC) identity providers. SSO connections are scoped per
// organization, allowing each tenant to configure its own identity provider.
// The plugin manages the full SSO flow including metadata exchange, assertion
// validation, and automatic user provisioning.
//
// Usage:
//
//	eng, _ := authsome.NewEngine(
//	    authsome.WithStore(store),
//	    authsome.WithPlugin(sso.New(sso.Config{
//	        DefaultRedirectURL: "https://example.com/dashboard",
//	    })),
//	)
package sso
