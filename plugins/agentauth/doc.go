// Package agentauth adds delegated agent identity to authsome: a way for a
// user to hand an AI agent scoped, expiring, revocable access, enforced as
// the intersection of what the agent was granted and what the delegating
// human may themselves do. See docs/superpowers/specs/2026-08-24-agentauth-delegation-design.md
// for the full design.
//
// # Host integration contract
//
// This package is a set of primitives, not a wired feature. Registering the
// plugin alone enforces nothing. A host application must do all five of the
// following, in this order, before an agent session carries any real
// restriction:
//
//  1. Call oauth2provider.SetConsentGate(agentauthPlugin), or set
//     oauth2provider.Config.ConsentGate to the agentauth plugin, so that
//     org policy is consulted at the moment a user consents to an agent.
//     agentauth's own OnInit does this automatically when an "oauth2provider"
//     plugin is registered on the same engine — see Plugin.OnInit — but a
//     host that constructs and wires oauth2provider independently of the
//     plugin registry (or that registers oauth2provider after agentauth has
//     already run its own OnInit) must still call one of these itself.
//
//  2. Call agentauth.CreateGrant from the application's own consent
//     handler — the screen or endpoint where a user approves an agent for a
//     set of scopes — and persist the returned AgentGrant.ID somewhere the
//     application can find again (its own consent record, typically). There
//     is no consent UI or handler in this package; only the primitive that
//     backs one.
//
//  3. Call agentauth.IssueAgentSession to mint the agent's credential,
//     rather than relying on oauth2provider's own token endpoint. The
//     standard OAuth2 token endpoint mints an ordinary human session: no
//     PrincipalKind, no grant, the application's normal session TTL, a
//     long-lived refresh token, and the delegating human's full role set.
//     Nothing about the OAuth2 authorization-code exchange knows this
//     package exists. IssueAgentSession is the only path that produces a
//     session carrying PrincipalKind "agent", AgentID and GrantID.
//
//  4. Mount agentauth.Guard(action, resource) on every route an agent may
//     reach — IN ADDITION TO middleware.RequirePermission or an equivalent
//     RBAC check, not instead of it. Guard enforces nothing for a session
//     that is not agent-principal; it exists purely to add the scope
//     intersection on top of whatever a human-facing permission check
//     already does. middleware.RequirePermission and
//     middleware.RequireAnyRole are not principal-kind aware: they resolve
//     the permission or role of sess.UserID, which for an agent session is
//     the delegating human, so an unguarded route grants the agent that
//     human's full permission, not the scoped subset the grant named. A host
//     that mounts RequirePermission alone, or Guard alone, has not
//     implemented the intersection this package exists to provide.
//
//  5. Re-issue, don't refresh. Agent sessions cannot be refreshed:
//     Engine.Refresh (service.go) refuses to rotate a session whose
//     PrincipalKind is "agent", by design, so that a session can never
//     outlive the grant that authorized it through the generic refresh
//     path. An agent session's TTL is short — 15 minutes, see
//     agentSessionTTL in issue.go — and a host must call
//     IssueAgentSession again before it expires to keep the agent working.
//     There is no long-lived agent credential; the grant is the long-lived
//     thing, the session is not.
//
// Until an application has wired all five of these, agentauth enforces
// nothing: an agent that authenticates through the ordinary OAuth2 flow
// receives a plain human session with that user's full access, and every
// route not explicitly wrapped in Guard is reachable with it regardless of
// what any grant says.
package agentauth
