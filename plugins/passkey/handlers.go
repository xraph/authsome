package passkey

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/xraph/forge"

	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/hook"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/store"
	"github.com/xraph/authsome/user"
)

// RegisterRoutes registers passkey/WebAuthn HTTP endpoints on a forge.Router.
func (p *Plugin) RegisterRoutes(r any) error {
	router, ok := r.(forge.Router)
	if !ok {
		return fmt.Errorf("passkey: expected forge.Router, got %T", r)
	}

	g := router.Group("/v1/auth/passkeys", forge.WithGroupTags("Passkeys"))

	if err := g.POST("/register/begin", p.handleRegisterBegin,
		forge.WithSummary("Begin passkey registration"),
		forge.WithOperationID("passkeyRegisterBegin"),
		forge.WithResponseSchema(http.StatusOK, "Registration options", RegisterBeginResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.POST("/register/finish", p.handleRegisterFinish,
		forge.WithSummary("Complete passkey registration"),
		forge.WithOperationID("passkeyRegisterFinish"),
		forge.WithResponseSchema(http.StatusOK, "Registered", RegisterFinishResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.POST("/login/begin", p.handleLoginBegin,
		forge.WithSummary("Begin passkey login"),
		forge.WithOperationID("passkeyLoginBegin"),
		forge.WithResponseSchema(http.StatusOK, "Assertion options", LoginBeginResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.POST("/login/finish", p.handleLoginFinish,
		forge.WithSummary("Complete passkey login"),
		forge.WithOperationID("passkeyLoginFinish"),
		forge.WithResponseSchema(http.StatusOK, "Authenticated", LoginFinishResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.GET("/", p.handleList,
		forge.WithSummary("List passkeys"),
		forge.WithOperationID("listPasskeys"),
		forge.WithResponseSchema(http.StatusOK, "Credential list", ListResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	return g.DELETE("/:credentialId", p.handleDelete,
		forge.WithSummary("Delete passkey"),
		forge.WithOperationID("deletePasskey"),
		forge.WithResponseSchema(http.StatusOK, "Deleted", DeleteResponse{}),
		forge.WithErrorResponses(),
	)
}

// ──────────────────────────────────────────────────
// Request / Response types
// ──────────────────────────────────────────────────

// RegisterBeginRequest is the request body for beginning passkey registration.
type RegisterBeginRequest struct {
	DisplayName string `json:"display_name,omitempty"`
}

// RegisterBeginResponse contains the WebAuthn registration options.
type RegisterBeginResponse struct {
	Options any `json:"options"`
}

// RegisterFinishRequest is the raw attestation response from the browser.
type RegisterFinishRequest struct{}

// RegisterFinishResponse confirms credential registration.
type RegisterFinishResponse struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	Status      string `json:"status"`
}

// LoginBeginRequest initiates a WebAuthn authentication ceremony.
type LoginBeginRequest struct {
	Email string `json:"email,omitempty"`
}

// LoginBeginResponse contains the WebAuthn assertion options.
type LoginBeginResponse struct {
	Options any `json:"options"`
}

// LoginFinishRequest is the raw assertion response from the browser.
type LoginFinishRequest struct{}

// LoginFinishResponse confirms successful authentication.
type LoginFinishResponse struct {
	UserID string `json:"user_id"`
	Status string `json:"status"`
}

// ListRequest is an empty request for listing passkeys.
type ListRequest struct{}

// ListResponse contains the user's registered passkeys.
type ListResponse struct {
	Credentials []*CredentialInfo `json:"credentials"`
}

// CredentialInfo is a serializable credential summary.
type CredentialInfo struct {
	ID          string   `json:"id"`
	DisplayName string   `json:"display_name"`
	Transport   []string `json:"transport"`
	CreatedAt   string   `json:"created_at"`
}

// DeleteRequest is the request for deleting a passkey.
type DeleteRequest struct {
	CredentialID string `path:"credentialId"`
}

// DeleteResponse confirms passkey deletion.
type DeleteResponse struct {
	Status string `json:"status"`
}

// ──────────────────────────────────────────────────
// Helper: resolve authenticated user
// ──────────────────────────────────────────────────

func (p *Plugin) resolveUser(ctx forge.Context) (*user.User, error) {
	u, ok := middleware.UserFrom(ctx.Context())
	if ok && u != nil {
		return u, nil
	}
	return nil, forge.Unauthorized("authentication required")
}

// ──────────────────────────────────────────────────
// Handlers
// ──────────────────────────────────────────────────

func (p *Plugin) handleRegisterBegin(ctx forge.Context, req *RegisterBeginRequest) (*RegisterBeginResponse, error) {
	if p.wa == nil {
		return nil, forge.InternalError(fmt.Errorf("passkey: WebAuthn not initialized"))
	}

	u, err := p.resolveUser(ctx)
	if err != nil {
		return nil, err
	}

	wau := p.toWebAuthnUser(ctx.Context(), u)

	options, session, err := p.wa.BeginRegistration(wau)
	if err != nil {
		return nil, forge.InternalError(fmt.Errorf("passkey: begin registration: %w", err))
	}

	// Store session data for the finish step
	key := "passkey:reg:" + u.ID.String()
	sessionJSON, _ := json.Marshal(&ceremonySession{ //nolint:errcheck // marshaling known types
		Data:        session,
		DisplayName: req.DisplayName,
	})
	_ = p.ceremonies.Set(ctx.Context(), key, sessionJSON, p.config.SessionTimeout) //nolint:errcheck // best-effort cache

	return &RegisterBeginResponse{Options: options}, nil
}

func (p *Plugin) handleRegisterFinish(ctx forge.Context, _ *RegisterFinishRequest) (*RegisterFinishResponse, error) {
	if p.wa == nil {
		return nil, forge.InternalError(fmt.Errorf("passkey: WebAuthn not initialized"))
	}

	u, err := p.resolveUser(ctx)
	if err != nil {
		return nil, err
	}

	key := "passkey:reg:" + u.ID.String()
	sessionJSON, err := p.ceremonies.Get(ctx.Context(), key)
	if err != nil {
		return nil, forge.BadRequest("no pending registration ceremony")
	}
	_ = p.ceremonies.Delete(ctx.Context(), key) //nolint:errcheck // best-effort cleanup

	var cs ceremonySession
	if unmarshalErr := json.Unmarshal(sessionJSON, &cs); unmarshalErr != nil {
		return nil, forge.InternalError(fmt.Errorf("failed to parse session: %w", unmarshalErr))
	}

	wau := p.toWebAuthnUser(ctx.Context(), u)

	cred, err := p.wa.FinishRegistration(wau, *cs.Data, ctx.Request())
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("passkey: finish registration: %v", err))
	}

	displayName := cs.DisplayName
	if displayName == "" {
		displayName = "Passkey"
	}

	credential := toCredential(u.ID, u.AppID, cred, displayName)
	if p.store != nil {
		if err := p.store.CreateCredential(ctx.Context(), credential); err != nil {
			return nil, forge.InternalError(fmt.Errorf("passkey: store credential: %w", err))
		}
	}

	credIDStr := hex.EncodeToString(cred.ID)
	userIDStr := u.ID.String()
	p.audit(ctx.Context(), hook.ActionPasskeyRegister, hook.ResourcePasskey, credIDStr, userIDStr, "", bridge.OutcomeSuccess)
	p.relayEvent(ctx.Context(), "auth.passkey.registered", "", map[string]string{"user_id": userIDStr})
	p.emitHook(ctx.Context(), hook.ActionPasskeyRegister, hook.ResourcePasskey, credIDStr, userIDStr, "")

	return &RegisterFinishResponse{
		ID:          credential.ID.String(),
		DisplayName: displayName,
		Status:      "registered",
	}, nil
}

func (p *Plugin) handleLoginBegin(ctx forge.Context, req *LoginBeginRequest) (*LoginBeginResponse, error) {
	if p.wa == nil {
		return nil, forge.InternalError(fmt.Errorf("passkey: WebAuthn not initialized"))
	}

	// For discoverable credentials (passkey), we can start without user identity
	if req.Email == "" {
		options, session, err := p.wa.BeginDiscoverableLogin()
		if err != nil {
			return nil, forge.InternalError(fmt.Errorf("passkey: begin discoverable login: %w", err))
		}
		sessionJSON, _ := json.Marshal(session)                                                           //nolint:errcheck // marshaling known types
		_ = p.ceremonies.Set(ctx.Context(), "passkey:discoverable", sessionJSON, p.config.SessionTimeout) //nolint:errcheck // best-effort cache
		return &LoginBeginResponse{Options: options}, nil
	}

	// With email, resolve user and generate assertion with existing credentials
	// For now, we require the user context to be set (e.g., by prior lookup)
	u, err := p.resolveUser(ctx)
	if err != nil {
		return nil, forge.BadRequest("user not found for passkey login")
	}

	wau := p.toWebAuthnUser(ctx.Context(), u)

	options, session, err := p.wa.BeginLogin(wau)
	if err != nil {
		return nil, forge.InternalError(fmt.Errorf("passkey: begin login: %w", err))
	}

	key := "passkey:login:" + u.ID.String()
	sessionJSON, _ := json.Marshal(session)                                        //nolint:errcheck // marshaling known types
	_ = p.ceremonies.Set(ctx.Context(), key, sessionJSON, p.config.SessionTimeout) //nolint:errcheck // best-effort cache

	return &LoginBeginResponse{Options: options}, nil
}

func (p *Plugin) handleLoginFinish(ctx forge.Context, _ *LoginFinishRequest) (*LoginFinishResponse, error) {
	if p.wa == nil {
		return nil, forge.InternalError(fmt.Errorf("passkey: WebAuthn not initialized"))
	}

	u, err := p.resolveUser(ctx)
	if err != nil {
		return nil, err
	}

	key := "passkey:login:" + u.ID.String()
	sessionJSON, err := p.ceremonies.Get(ctx.Context(), key)
	if err != nil {
		return nil, forge.BadRequest("no pending login ceremony")
	}
	_ = p.ceremonies.Delete(ctx.Context(), key) //nolint:errcheck // best-effort cleanup

	var session webauthn.SessionData
	if unmarshalErr := json.Unmarshal(sessionJSON, &session); unmarshalErr != nil {
		return nil, forge.InternalError(fmt.Errorf("failed to parse session: %w", unmarshalErr))
	}

	wau := p.toWebAuthnUser(ctx.Context(), u)

	cred, err := p.wa.FinishLogin(wau, session, ctx.Request())
	if err != nil {
		return nil, forge.Unauthorized(fmt.Sprintf("passkey: finish login: %v", err))
	}

	// Update sign count
	if p.store != nil {
		_ = p.store.UpdateSignCount(ctx.Context(), cred.ID, cred.Authenticator.SignCount) //nolint:errcheck // best-effort update
	}

	userIDStr := u.ID.String()
	p.audit(ctx.Context(), hook.ActionPasskeyLogin, hook.ResourcePasskey, "", userIDStr, "", bridge.OutcomeSuccess)
	p.relayEvent(ctx.Context(), "auth.passkey.authenticated", "", map[string]string{"user_id": userIDStr})
	p.emitHook(ctx.Context(), hook.ActionPasskeyLogin, hook.ResourcePasskey, "", userIDStr, "")

	return &LoginFinishResponse{
		UserID: userIDStr,
		Status: "authenticated",
	}, nil
}

func (p *Plugin) handleList(ctx forge.Context, _ *ListRequest) (*ListResponse, error) {
	u, err := p.resolveUser(ctx)
	if err != nil {
		return nil, err
	}

	if p.store == nil {
		return &ListResponse{Credentials: []*CredentialInfo{}}, nil
	}

	creds, err := p.store.ListUserCredentials(ctx.Context(), u.ID)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		return nil, forge.InternalError(fmt.Errorf("passkey: list credentials: %w", err))
	}

	var infos []*CredentialInfo
	for _, c := range creds {
		infos = append(infos, &CredentialInfo{
			ID:          hex.EncodeToString(c.CredentialID),
			DisplayName: c.DisplayName,
			Transport:   c.Transport,
			CreatedAt:   c.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	if infos == nil {
		infos = []*CredentialInfo{}
	}

	return &ListResponse{Credentials: infos}, nil
}

func (p *Plugin) handleDelete(ctx forge.Context, req *DeleteRequest) (*DeleteResponse, error) {
	u, err := p.resolveUser(ctx)
	if err != nil {
		return nil, err
	}

	credentialIDBytes, err := hex.DecodeString(req.CredentialID)
	if err != nil {
		return nil, forge.BadRequest("invalid credential ID")
	}

	if p.store != nil {
		if err := p.store.DeleteCredential(ctx.Context(), credentialIDBytes); err != nil {
			if errors.Is(err, ErrCredentialNotFound) {
				return nil, forge.NotFound("credential not found")
			}
			return nil, forge.InternalError(fmt.Errorf("passkey: delete credential: %w", err))
		}
	}

	userIDStr := u.ID.String()
	p.audit(ctx.Context(), hook.ActionPasskeyDelete, hook.ResourcePasskey, req.CredentialID, userIDStr, "", bridge.OutcomeSuccess)
	p.relayEvent(ctx.Context(), "auth.passkey.deleted", "", map[string]string{"user_id": userIDStr, "credential_id": req.CredentialID})
	p.emitHook(ctx.Context(), hook.ActionPasskeyDelete, hook.ResourcePasskey, req.CredentialID, userIDStr, "")

	return &DeleteResponse{Status: "deleted"}, nil
}

// ──────────────────────────────────────────────────
// Internal: ceremony session wrapper
// ──────────────────────────────────────────────────

type ceremonySession struct {
	Data        *webauthn.SessionData `json:"data"`
	DisplayName string                `json:"display_name"`
}
