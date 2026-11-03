package oauth2provider

import (
	"context"
	"net/http"

	"github.com/xraph/forge"
	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome"
	"github.com/xraph/authsome/apitypes"
	"github.com/xraph/authsome/internal/jwkutil"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/tokenformat"
)

// jwksCacheControl lets relying parties and CDNs hold the key set for the
// same five minutes the signing key cache is held for.
const jwksCacheControl = "public, max-age=300"

// JWKSResponse is the JSON Web Key Set the provider publishes.
type JWKSResponse struct {
	Keys []jwkutil.JWK `json:"keys"`
}

// jwksFor assembles the key set for one issuer scope: the id_token keys the
// key manager publishes, and the asymmetric access token keys the engine
// verifies with. The deployment issuer ("") gets every JWT format's key; an
// app scope gets only its own app's, and nothing from the deployment scope.
// HMAC keys are never included. Entries are deduplicated by kid.
func (p *Plugin) jwksFor(ctx context.Context, scope string) (*JWKSResponse, error) {
	out := &JWKSResponse{Keys: []jwkutil.JWK{}}
	seen := map[string]bool{}
	add := func(j jwkutil.JWK) {
		if j.KID != "" {
			if seen[j.KID] {
				return
			}
			seen[j.KID] = true
		}
		out.Keys = append(out.Keys, j)
	}

	if p.keys != nil {
		keys, err := p.keys.Published(ctx, scope)
		if err != nil {
			return nil, err
		}
		for i := range keys {
			add(keys[i])
		}
	}
	for _, j := range p.accessTokenJWKs(scope) {
		add(j)
	}
	return out, nil
}

// accessTokenJWKs returns the public halves of the engine's asymmetric JWT
// access token formats: all of them, and the default format, for the
// deployment scope (""), or just the format registered for the app for an
// app scope. The plugin.Engine interface does not expose them, so this
// asserts the concrete engine, as the refresh grant does.
func (p *Plugin) accessTokenJWKs(scope string) []jwkutil.JWK {
	eng, ok := p.engine.(*authsome.Engine)
	if !ok || eng == nil {
		return nil
	}
	var out []jwkutil.JWK
	add := func(f *tokenformat.JWT) {
		pub := f.PublicKey()
		if pub == nil {
			return // HMAC: never published
		}
		j, err := jwkutil.Encode(pub, f.KeyID(), f.Algorithm())
		if err != nil {
			p.logger.Warn("oauth2: skip unencodable access token key in JWKS", log.Error(err))
			return
		}
		out = append(out, *j)
	}
	if scope != "" {
		if f := eng.JWTFormats()[scope]; f != nil {
			add(f)
		}
		return out
	}
	for _, f := range eng.JWTFormats() {
		add(f)
	}
	if f, ok := eng.DefaultTokenFormat().(*tokenformat.JWT); ok && f != nil {
		add(f)
	}
	return out
}

// idTokenAlgs lists the signing algorithms scope publishes keys for. It
// reads the key manager without generating anything. A scope with no key
// advertises RS256, what a managed key would use: an empty list makes many
// OIDC clients refuse the discovery document.
func (p *Plugin) idTokenAlgs(ctx context.Context, scope string) []string {
	var algs []string
	if p.keys != nil {
		info, err := p.keys.Describe(ctx, scope)
		if err != nil {
			p.logger.Warn("oauth2: describe signing keys for discovery",
				log.String("scope", scope), log.Error(err))
		}
		seen := map[string]bool{}
		for _, k := range info {
			if k.Alg != "" && !seen[k.Alg] {
				seen[k.Alg] = true
				algs = append(algs, k.Alg)
			}
		}
	}
	if len(algs) == 0 {
		return []string{"RS256"}
	}
	return algs
}

func (p *Plugin) handleJWKS(ctx forge.Context, _ *apitypes.Empty) (*JWKSResponse, error) {
	doc, err := p.jwksFor(ctx.Context(), "")
	if err != nil {
		return nil, middleware.InternalError(ctx, err)
	}
	ctx.SetHeader("Cache-Control", jwksCacheControl)
	return doc, nil
}

func (p *Plugin) registerJWKSRoute(g forge.Router) error {
	return g.GET("/jwks", p.handleJWKS,
		forge.WithSummary("JSON Web Key Set"),
		forge.WithDescription("Public keys that verify id_tokens and asymmetric access tokens issued by this provider."),
		forge.WithOperationID("oauth2JWKS"),
		forge.WithResponseSchema(http.StatusOK, "JWKS", JWKSResponse{}),
		forge.WithErrorResponses(),
	)
}
