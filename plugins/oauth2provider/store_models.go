package oauth2provider

import (
	"encoding/json"
	"time"

	"github.com/xraph/grove"

	"github.com/xraph/authsome/id"
)

// ──────────────────────────────────────────────────
// OAuth2 client model (shared across SQL stores)
// ──────────────────────────────────────────────────

type oauth2ClientModel struct {
	grove.BaseModel `grove:"table:authsome_oauth2_clients,alias:oc"`

	ID           string          `grove:"id,pk"`
	AppID        string          `grove:"app_id,notnull"`
	Name         string          `grove:"name,notnull"`
	ClientID     string          `grove:"client_id,notnull"`
	ClientSecret string          `grove:"client_secret,notnull"`
	RedirectURIs json.RawMessage `grove:"redirect_uris,type:jsonb"`
	Scopes       json.RawMessage `grove:"scopes,type:jsonb"`
	GrantTypes   json.RawMessage `grove:"grant_types,type:jsonb"`
	Public       bool            `grove:"public,notnull"`
	CreatedAt    time.Time       `grove:"created_at,notnull,default:now()"`
	UpdatedAt    time.Time       `grove:"updated_at,notnull,default:now()"`
}

// ──────────────────────────────────────────────────
// Authorization code model (shared across SQL stores)
// ──────────────────────────────────────────────────

type authCodeModel struct {
	grove.BaseModel `grove:"table:authsome_oauth2_auth_codes,alias:ac"`

	ID                  string          `grove:"id,pk"`
	Code                string          `grove:"code,notnull"`
	ClientID            string          `grove:"client_id,notnull"`
	UserID              string          `grove:"user_id,notnull"`
	AppID               string          `grove:"app_id,notnull"`
	RedirectURI         string          `grove:"redirect_uri,notnull"`
	Scopes              json.RawMessage `grove:"scopes,type:jsonb"`
	CodeChallenge       string          `grove:"code_challenge,notnull"`
	CodeChallengeMethod string          `grove:"code_challenge_method,notnull"`
	ExpiresAt           time.Time       `grove:"expires_at,notnull"`
	Consumed            bool            `grove:"consumed,notnull"`
	CreatedAt           time.Time       `grove:"created_at,notnull,default:now()"`
}

// ──────────────────────────────────────────────────
// Device code model (shared across SQL stores)
// ──────────────────────────────────────────────────

type deviceCodeModel struct {
	grove.BaseModel `grove:"table:authsome_oauth2_device_codes,alias:dc"`

	ID              string          `grove:"id,pk"`
	DeviceCode      string          `grove:"device_code,notnull"`
	UserCode        string          `grove:"user_code,notnull"`
	ClientID        string          `grove:"client_id,notnull"`
	AppID           string          `grove:"app_id,notnull"`
	Scopes          json.RawMessage `grove:"scopes,type:jsonb"`
	VerificationURI string          `grove:"verification_uri,notnull"`
	ExpiresAt       time.Time       `grove:"expires_at,notnull"`
	Interval        int             `grove:"interval,notnull"`
	Status          string          `grove:"status,notnull"`
	UserID          string          `grove:"user_id"`
	LastPolledAt    time.Time       `grove:"last_polled_at"`
	CreatedAt       time.Time       `grove:"created_at,notnull,default:now()"`
}

// ──────────────────────────────────────────────────
// OAuth2 client converters
// ──────────────────────────────────────────────────

func toOAuth2Client(m *oauth2ClientModel) (*OAuth2Client, error) {
	clientID, err := id.ParseOAuth2ClientID(m.ID)
	if err != nil {
		return nil, err
	}
	appID, err := id.ParseAppID(m.AppID)
	if err != nil {
		return nil, err
	}

	var redirectURIs []string
	if len(m.RedirectURIs) > 0 {
		_ = json.Unmarshal(m.RedirectURIs, &redirectURIs)
	}
	var scopes []string
	if len(m.Scopes) > 0 {
		_ = json.Unmarshal(m.Scopes, &scopes)
	}
	var grantTypes []string
	if len(m.GrantTypes) > 0 {
		_ = json.Unmarshal(m.GrantTypes, &grantTypes)
	}

	return &OAuth2Client{
		ID:           clientID,
		AppID:        appID,
		Name:         m.Name,
		ClientID:     m.ClientID,
		ClientSecret: m.ClientSecret,
		RedirectURIs: redirectURIs,
		Scopes:       scopes,
		GrantTypes:   grantTypes,
		Public:       m.Public,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}, nil
}

func fromOAuth2Client(c *OAuth2Client) *oauth2ClientModel {
	redirectURIs, _ := json.Marshal(c.RedirectURIs)
	if len(redirectURIs) == 0 {
		redirectURIs = []byte("[]")
	}
	scopes, _ := json.Marshal(c.Scopes)
	if len(scopes) == 0 {
		scopes = []byte("[]")
	}
	grantTypes, _ := json.Marshal(c.GrantTypes)
	if len(grantTypes) == 0 {
		grantTypes = []byte("[]")
	}

	return &oauth2ClientModel{
		ID:           c.ID.String(),
		AppID:        c.AppID.String(),
		Name:         c.Name,
		ClientID:     c.ClientID,
		ClientSecret: c.ClientSecret,
		RedirectURIs: redirectURIs,
		Scopes:       scopes,
		GrantTypes:   grantTypes,
		Public:       c.Public,
		CreatedAt:    c.CreatedAt,
		UpdatedAt:    c.UpdatedAt,
	}
}

// ──────────────────────────────────────────────────
// Authorization code converters
// ──────────────────────────────────────────────────

func toAuthCode(m *authCodeModel) (*AuthorizationCode, error) {
	codeID, err := id.ParseAuthCodeID(m.ID)
	if err != nil {
		return nil, err
	}
	userID, err := id.ParseUserID(m.UserID)
	if err != nil {
		return nil, err
	}
	appID, err := id.ParseAppID(m.AppID)
	if err != nil {
		return nil, err
	}

	var scopes []string
	if len(m.Scopes) > 0 {
		_ = json.Unmarshal(m.Scopes, &scopes)
	}

	return &AuthorizationCode{
		ID:                  codeID,
		Code:                m.Code,
		ClientID:            m.ClientID,
		UserID:              userID,
		AppID:               appID,
		RedirectURI:         m.RedirectURI,
		Scopes:              scopes,
		CodeChallenge:       m.CodeChallenge,
		CodeChallengeMethod: m.CodeChallengeMethod,
		ExpiresAt:           m.ExpiresAt,
		Consumed:            m.Consumed,
		CreatedAt:           m.CreatedAt,
	}, nil
}

func fromAuthCode(c *AuthorizationCode) *authCodeModel {
	scopes, _ := json.Marshal(c.Scopes)
	if len(scopes) == 0 {
		scopes = []byte("[]")
	}

	return &authCodeModel{
		ID:                  c.ID.String(),
		Code:                c.Code,
		ClientID:            c.ClientID,
		UserID:              c.UserID.String(),
		AppID:               c.AppID.String(),
		RedirectURI:         c.RedirectURI,
		Scopes:              scopes,
		CodeChallenge:       c.CodeChallenge,
		CodeChallengeMethod: c.CodeChallengeMethod,
		ExpiresAt:           c.ExpiresAt,
		Consumed:            c.Consumed,
		CreatedAt:           c.CreatedAt,
	}
}

// ──────────────────────────────────────────────────
// Device code converters
// ──────────────────────────────────────────────────

func toDeviceCode(m *deviceCodeModel) (*DeviceCode, error) {
	dcID, err := id.ParseDeviceCodeID(m.ID)
	if err != nil {
		return nil, err
	}
	appID, err := id.ParseAppID(m.AppID)
	if err != nil {
		return nil, err
	}

	var userID id.UserID
	if m.UserID != "" {
		userID, err = id.ParseUserID(m.UserID)
		if err != nil {
			return nil, err
		}
	}

	var scopes []string
	if len(m.Scopes) > 0 {
		_ = json.Unmarshal(m.Scopes, &scopes)
	}

	return &DeviceCode{
		ID:              dcID,
		DeviceCode:      m.DeviceCode,
		UserCode:        m.UserCode,
		ClientID:        m.ClientID,
		AppID:           appID,
		Scopes:          scopes,
		VerificationURI: m.VerificationURI,
		ExpiresAt:       m.ExpiresAt,
		Interval:        m.Interval,
		Status:          m.Status,
		UserID:          userID,
		LastPolledAt:    m.LastPolledAt,
		CreatedAt:       m.CreatedAt,
	}, nil
}

func fromDeviceCode(dc *DeviceCode) *deviceCodeModel {
	scopes, _ := json.Marshal(dc.Scopes)
	if len(scopes) == 0 {
		scopes = []byte("[]")
	}

	return &deviceCodeModel{
		ID:              dc.ID.String(),
		DeviceCode:      dc.DeviceCode,
		UserCode:        dc.UserCode,
		ClientID:        dc.ClientID,
		AppID:           dc.AppID.String(),
		Scopes:          scopes,
		VerificationURI: dc.VerificationURI,
		ExpiresAt:       dc.ExpiresAt,
		Interval:        dc.Interval,
		Status:          dc.Status,
		UserID:          dc.UserID.String(),
		LastPolledAt:    dc.LastPolledAt,
		CreatedAt:       dc.CreatedAt,
	}
}

