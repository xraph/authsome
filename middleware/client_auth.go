package middleware

import (
	"context"
	"sync"
	"time"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/id"
	authclient "github.com/xraph/authsome/sdk/go"
	"github.com/xraph/authsome/user"
)

type rawTokenKey struct{}

// WithRawToken stores the raw bearer token in context so downstream services
// can forward it to other services (e.g. TwinOS → Portal for user-scoped ops).
func WithRawToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, rawTokenKey{}, token)
}

// RawTokenFrom extracts the raw bearer token from context.
func RawTokenFrom(ctx context.Context) (string, bool) {
	token, ok := ctx.Value(rawTokenKey{}).(string)
	return token, ok && token != ""
}

// introspectCacheEntry holds a cached introspection result.
type introspectCacheEntry struct {
	resp      *authclient.IntrospectResponse
	expiresAt time.Time
}

// ClientAuthMiddleware validates tokens by calling the remote authsome server.
// Opaque tokens are validated via the /v1/introspect endpoint. Results are
// cached briefly to avoid per-request HTTP calls.
//
// This middleware sets the same context keys as the engine-based AuthMiddleware
// (WithUser, WithUserID, WithAppID, WithOrgID, WithSessionID, WithAuthMethod)
// so downstream code works identically regardless of which middleware is used.
func ClientAuthMiddleware(client *authclient.Client, logger log.Logger) forge.Middleware {
	cacheMu := &sync.RWMutex{}
	cache := make(map[string]*introspectCacheEntry)
	cacheTTL := 30 * time.Second

	return func(next forge.Handler) forge.Handler {
		return func(ctx forge.Context) error {
			token := extractBearerToken(ctx.Request())
			if token == "" {
				return next(ctx)
			}

			goCtx := ctx.Context()
			goCtx = WithRawToken(goCtx, token)

			// Check cache first
			cacheMu.RLock()
			entry, cached := cache[token]
			cacheMu.RUnlock()

			var resp *authclient.IntrospectResponse

			if cached && time.Now().Before(entry.expiresAt) {
				resp = entry.resp
			} else {
				// Call introspect endpoint
				var err error
				resp, err = client.Introspect(goCtx, token)
				if err != nil {
					logger.Debug("client auth: introspect failed",
						log.String("error", err.Error()),
					)
					ctx.WithContext(goCtx)
					return next(ctx)
				}

				// Cache the result
				cacheMu.Lock()
				cache[token] = &introspectCacheEntry{
					resp:      resp,
					expiresAt: time.Now().Add(cacheTTL),
				}
				// Evict stale entries periodically (simple approach)
				if len(cache) > 1000 {
					now := time.Now()
					for k, v := range cache {
						if now.After(v.expiresAt) {
							delete(cache, k)
						}
					}
				}
				cacheMu.Unlock()
			}

			if !resp.Active {
				ctx.WithContext(goCtx)
				return next(ctx)
			}

			// Set context from introspect response
			goCtx = setContextFromIntrospect(goCtx, resp)
			goCtx = WithAuthMethod(goCtx, "session")
			ctx.WithContext(goCtx)
			return next(ctx)
		}
	}
}

// setContextFromIntrospect populates the middleware context from an introspect response.
func setContextFromIntrospect(ctx context.Context, resp *authclient.IntrospectResponse) context.Context {
	if resp.UserID != "" {
		if parsed, err := id.ParseUserID(resp.UserID); err == nil {
			ctx = WithUserID(ctx, parsed)
		}
	}

	if resp.AppID != "" {
		if parsed, err := id.ParseAppID(resp.AppID); err == nil {
			ctx = WithAppID(ctx, parsed)
		}
	}

	if resp.OrgID != "" {
		if parsed, err := id.ParseOrgID(resp.OrgID); err == nil {
			ctx = WithOrgID(ctx, parsed)
			if resp.AppID != "" {
				ctx = forge.WithScope(ctx, forge.NewOrgScope(resp.AppID, resp.OrgID))
			}
		}
	} else if resp.AppID != "" {
		ctx = forge.WithScope(ctx, forge.NewAppScope(resp.AppID))
	}

	if resp.SessionID != "" {
		if parsed, err := id.ParseSessionID(resp.SessionID); err == nil {
			ctx = WithSessionID(ctx, parsed)
		}
	}

	if resp.EnvID != "" {
		if parsed, err := id.ParseEnvironmentID(resp.EnvID); err == nil {
			ctx = WithEnvID(ctx, parsed)
		}
	}

	// Set user if available
	if resp.User != nil {
		if parsed, err := id.ParseUserID(resp.User.ID); err == nil {
			u := &user.User{
				ID:        parsed,
				Email:     resp.User.Email,
				FirstName: resp.User.FirstName,
				LastName:  resp.User.LastName,
				Username:  resp.User.Username,
			}
			ctx = WithUser(ctx, u)
		}
	}

	return ctx
}
