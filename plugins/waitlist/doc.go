// Package waitlist provides a pre-launch waitlist gate for AuthSome.
//
// The waitlist plugin allows administrators to gate sign-ups behind an
// approval process. Users join the waitlist with their email address and
// are assigned a "pending" status. Administrators can approve or reject
// entries. Only approved users are allowed to complete the sign-up flow.
//
// Usage:
//
//	eng, _ := authsome.NewEngine(
//	    authsome.WithStore(store),
//	    authsome.WithPlugin(waitlist.New()),
//	)
package waitlist
