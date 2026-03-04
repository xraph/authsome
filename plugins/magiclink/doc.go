// Package magiclink provides passwordless magic link authentication for AuthSome.
//
// The magic link plugin allows users to sign in by clicking a one-time link
// sent to their email. It uses the core verification entity (type="magic_link")
// for token storage, so no additional database tables are required.
//
// Usage:
//
//	mailer := &MyMailer{} // implements magiclink.Mailer
//	eng, _ := authsome.NewEngine(
//	    authsome.WithStore(store),
//	    authsome.WithPlugin(magiclink.New(magiclink.Config{
//	        Mailer: mailer,
//	    })),
//	)
package magiclink
