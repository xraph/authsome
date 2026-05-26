// handlers.go: API key intent handlers, owned by the apikey plugin.
//
// Phase D.3-batch4 moved these from the auth contributor's
// handlers_apikeys.go into this package. The /apikeys list+detail
// route stays declared on the auth contributor's manifest (the route
// belongs to the platform-level navigation, not to the apikey
// plugin's own surface); only the intent registrations move. Cross-
// contributor intent invocations work because the dashboard's
// contract dispatcher routes by intent name globally — manifest-level
// declarations are partitioned by contributor for organisation, but
// the wire dispatch is name-only.
package contract

import (
	"context"
	"strings"
	"time"

	"github.com/xraph/authsome/apikey"
	"github.com/xraph/authsome/id"

	"github.com/xraph/forge/extensions/dashboard/contract"

	authcontract "github.com/xraph/authsome/extension/contract"
)

// ────────────────────────────────────────────────────────────────────
// Wire shapes
// ────────────────────────────────────────────────────────────────────

type APIKeySummary struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	KeyPrefix  string   `json:"keyPrefix"`
	Scopes     []string `json:"scopes,omitempty"`
	Revoked    bool     `json:"revoked"`
	ExpiresAt  string   `json:"expiresAt,omitempty"`
	LastUsedAt string   `json:"lastUsedAt,omitempty"`
	CreatedAt  string   `json:"createdAt"`
}

type APIKeyDetail struct {
	APIKeySummary
	AppID            string `json:"appId,omitempty"`
	EnvID            string `json:"envId,omitempty"`
	UserID           string `json:"userId,omitempty"`
	ServiceAccountID string `json:"serviceAccountId,omitempty"`
	PublicKey        string `json:"publicKey,omitempty"`
	UpdatedAt        string `json:"updatedAt"`
}

type GetAPIKeyInput struct {
	ID string `json:"id"`
}

type APIKeyListResponse struct {
	APIKeys []APIKeySummary `json:"apiKeys"`
}

type CreateAPIKeyInput struct {
	Name   string   `json:"name"`
	UserID string   `json:"userId"`
	Scopes []string `json:"scopes,omitempty"`
}

type RevokeAPIKeyInput struct {
	ID string `json:"id"`
}

// CreateAPIKeyResponse carries the plaintext key the user must copy
// immediately — it's NEVER readable again (the server stores only the
// hash). The shell shows this in a one-time modal after creation.
type CreateAPIKeyResponse struct {
	OK        bool   `json:"ok"`
	ID        string `json:"id"`
	KeyPrefix string `json:"keyPrefix"`
	// Secret is the plaintext API key (`ask_<hex>...`). Present only on
	// the create response; subsequent reads omit it.
	Secret string `json:"secret"`
}

type ackResponse struct {
	OK bool   `json:"ok"`
	ID string `json:"id,omitempty"`
}

// ────────────────────────────────────────────────────────────────────
// Handlers
// ────────────────────────────────────────────────────────────────────

func apikeysListHandler(deps Deps) func(ctx context.Context, _ struct{}, p contract.Principal) (APIKeyListResponse, error) {
	return func(ctx context.Context, _ struct{}, p contract.Principal) (APIKeyListResponse, error) {
		if deps.Engine == nil {
			return APIKeyListResponse{}, unavailable("auth engine not configured")
		}
		store := deps.Engine.APIKeyStore()
		if store == nil {
			return APIKeyListResponse{}, unavailable("api key store not configured")
		}
		list, err := store.ListAPIKeysByApp(ctx, authcontract.AppIDFromPrincipal(p, deps.Engine))
		if err != nil {
			return APIKeyListResponse{}, mapErr(err)
		}
		out := APIKeyListResponse{APIKeys: make([]APIKeySummary, 0, len(list))}
		for _, k := range list {
			out.APIKeys = append(out.APIKeys, projectAPIKey(k))
		}
		return out, nil
	}
}

func apikeysCreateHandler(deps Deps) func(ctx context.Context, in CreateAPIKeyInput, p contract.Principal) (CreateAPIKeyResponse, error) {
	return func(ctx context.Context, in CreateAPIKeyInput, p contract.Principal) (CreateAPIKeyResponse, error) {
		if deps.Engine == nil {
			return CreateAPIKeyResponse{}, unavailable("auth engine not configured")
		}
		store := deps.Engine.APIKeyStore()
		if store == nil {
			return CreateAPIKeyResponse{}, unavailable("api key store not configured")
		}
		name := strings.TrimSpace(in.Name)
		if name == "" {
			return CreateAPIKeyResponse{}, badReq("name is required")
		}
		uid, err := parseUserID(in.UserID)
		if err != nil {
			return CreateAPIKeyResponse{}, err
		}
		// apikey.GenerateKey produces (raw, hash, prefix). We persist
		// the hash + prefix and hand `raw` back to the caller — they
		// have one chance to copy it.
		raw, hash, prefix, genErr := apikey.GenerateKey()
		if genErr != nil {
			return CreateAPIKeyResponse{}, &contract.Error{Code: contract.CodeInternal, Message: "generate key: " + genErr.Error()}
		}
		k := &apikey.APIKey{
			ID:        id.NewAPIKeyID(),
			AppID:     authcontract.AppIDFromPrincipal(p, deps.Engine),
			UserID:    uid,
			Name:      name,
			KeyHash:   hash,
			KeyPrefix: prefix,
			Scopes:    in.Scopes,
		}
		if err := store.CreateAPIKey(ctx, k); err != nil {
			return CreateAPIKeyResponse{}, mapErr(err)
		}
		return CreateAPIKeyResponse{
			OK: true, ID: k.ID.String(),
			KeyPrefix: prefix, Secret: raw,
		}, nil
	}
}

func apikeysRevokeHandler(deps Deps) func(ctx context.Context, in RevokeAPIKeyInput, _ contract.Principal) (ackResponse, error) {
	return func(ctx context.Context, in RevokeAPIKeyInput, _ contract.Principal) (ackResponse, error) {
		if deps.Engine == nil {
			return ackResponse{}, unavailable("auth engine not configured")
		}
		store := deps.Engine.APIKeyStore()
		if store == nil {
			return ackResponse{}, unavailable("api key store not configured")
		}
		kid, err := parseAPIKeyID(in.ID)
		if err != nil {
			return ackResponse{}, err
		}
		k, err := store.GetAPIKey(ctx, kid)
		if err != nil {
			return ackResponse{}, mapErr(err)
		}
		k.Revoked = true
		if err := store.UpdateAPIKey(ctx, k); err != nil {
			return ackResponse{}, mapErr(err)
		}
		return ackResponse{OK: true, ID: kid.String()}, nil
	}
}

func apikeysDetailHandler(deps Deps) func(ctx context.Context, in GetAPIKeyInput, _ contract.Principal) (APIKeyDetail, error) {
	return func(ctx context.Context, in GetAPIKeyInput, _ contract.Principal) (APIKeyDetail, error) {
		if deps.Engine == nil {
			return APIKeyDetail{}, unavailable("auth engine not configured")
		}
		store := deps.Engine.APIKeyStore()
		if store == nil {
			return APIKeyDetail{}, unavailable("api key store not configured")
		}
		kid, err := parseAPIKeyID(in.ID)
		if err != nil {
			return APIKeyDetail{}, err
		}
		k, err := store.GetAPIKey(ctx, kid)
		if err != nil {
			return APIKeyDetail{}, mapErr(err)
		}
		d := APIKeyDetail{
			APIKeySummary: projectAPIKey(k),
			AppID:         k.AppID.String(),
			EnvID:         k.EnvID.String(),
			UserID:        k.UserID.String(),
			PublicKey:     k.PublicKey,
			UpdatedAt:     k.UpdatedAt.UTC().Format(time.RFC3339),
		}
		if !k.ServiceAccountID.IsNil() {
			d.ServiceAccountID = k.ServiceAccountID.String()
		}
		return d, nil
	}
}

// ────────────────────────────────────────────────────────────────────
// Helpers
// ────────────────────────────────────────────────────────────────────

func projectAPIKey(k *apikey.APIKey) APIKeySummary {
	if k == nil {
		return APIKeySummary{}
	}
	out := APIKeySummary{
		ID: k.ID.String(), Name: k.Name, KeyPrefix: k.KeyPrefix,
		Scopes: k.Scopes, Revoked: k.Revoked,
		CreatedAt: k.CreatedAt.UTC().Format(time.RFC3339),
	}
	if k.ExpiresAt != nil {
		out.ExpiresAt = k.ExpiresAt.UTC().Format(time.RFC3339)
	}
	if k.LastUsedAt != nil {
		out.LastUsedAt = k.LastUsedAt.UTC().Format(time.RFC3339)
	}
	return out
}

func parseAPIKeyID(s string) (id.APIKeyID, error) {
	if strings.TrimSpace(s) == "" {
		return id.APIKeyID{}, badReq("id is required")
	}
	kid, err := id.ParseAPIKeyID(s)
	if err != nil {
		return id.APIKeyID{}, badReq("invalid api key id: " + err.Error())
	}
	return kid, nil
}

func parseUserID(s string) (id.UserID, error) {
	if strings.TrimSpace(s) == "" {
		return id.UserID{}, badReq("user id is required")
	}
	uid, err := id.ParseUserID(s)
	if err != nil {
		return id.UserID{}, badReq("invalid user id: " + err.Error())
	}
	return uid, nil
}

func badReq(msg string) error {
	return &contract.Error{Code: contract.CodeBadRequest, Message: msg}
}

func unavailable(msg string) error {
	return &contract.Error{Code: contract.CodeUnavailable, Message: msg}
}

// mapErr is a tiny pass-through that wraps engine errors as contract
// CodeInternal. The auth contributor's mapEngineError has a richer
// switch over typed engine sentinels; for now plugins replicate the
// minimum needed surface. Pull error-mapping into a shared package
// when the duplication starts to bite.
func mapErr(err error) error {
	if err == nil {
		return nil
	}
	if ce, ok := err.(*contract.Error); ok {
		return ce
	}
	return &contract.Error{Code: contract.CodeInternal, Message: err.Error()}
}
