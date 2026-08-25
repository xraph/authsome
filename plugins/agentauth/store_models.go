package agentauth

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/xraph/grove"

	"github.com/xraph/authsome/id"
)

// ──────────────────────────────────────────────────
// Agent model (shared across SQL stores)
// ──────────────────────────────────────────────────

type agentModel struct {
	grove.BaseModel `grove:"table:authsome_agents,alias:ag"`

	ID          string    `grove:"id,pk"`
	AppID       string    `grove:"app_id,notnull"`
	OrgID       string    `grove:"org_id,notnull"`
	ClientID    string    `grove:"client_id,notnull"`
	Name        string    `grove:"name,notnull"`
	Description string    `grove:"description,notnull"`
	LogoURI     string    `grove:"logo_uri,notnull"`
	Origin      string    `grove:"origin,notnull"`
	Status      string    `grove:"status,notnull"`
	CreatedBy   string    `grove:"created_by,notnull"`
	CreatedAt   time.Time `grove:"created_at,notnull,default:now()"`
	UpdatedAt   time.Time `grove:"updated_at,notnull,default:now()"`
}

// ──────────────────────────────────────────────────
// Agent grant model (shared across SQL stores)
// ──────────────────────────────────────────────────

type agentGrantModel struct {
	grove.BaseModel `grove:"table:authsome_agent_grants,alias:agr"`

	ID      string `grove:"id,pk"`
	AppID   string `grove:"app_id,notnull"`
	AgentID string `grove:"agent_id,notnull"`
	UserID  string `grove:"user_id,notnull"`
	OrgID   string `grove:"org_id,notnull"`
	// Scopes is stored as a raw JSON array (TEXT), not a jsonb-typed column,
	// so the same model works unmodified against both Postgres and SQLite —
	// mirrors plugins/sso/store_models.go's AttributeMappings rather than
	// oauth2provider's json.RawMessage+jsonb approach, which is Postgres-only.
	Scopes    string    `grove:"scopes,notnull"`
	ConsentID string    `grove:"consent_id,notnull"`
	ExpiresAt time.Time `grove:"expires_at,notnull"`
	// LastUsedAt/RevokedAt are *time.Time, not sql.NullTime: grove's scan
	// layer (grove/scan/convert.go) only wraps *time.Time/**time.Time
	// destinations with its TEXT-aware adapter. A sql.NullTime destination
	// falls through to its own Scan method, which cannot parse the TEXT
	// timestamp strings SQLite returns — confirmed by hand, this fails with
	// "unsupported Scan, storing driver.Value type string into type
	// *time.Time" against SQLite even though the identical column reads
	// fine against Postgres, which returns a native time.Time already.
	LastUsedAt *time.Time `grove:"last_used_at"`
	RevokedAt  *time.Time `grove:"revoked_at"`
	CreatedAt  time.Time  `grove:"created_at,notnull,default:now()"`
	UpdatedAt  time.Time  `grove:"updated_at,notnull,default:now()"`
}

// ──────────────────────────────────────────────────
// Org agent policy model (shared across SQL stores)
// ──────────────────────────────────────────────────

type orgPolicyModel struct {
	grove.BaseModel `grove:"table:authsome_agent_policies,alias:ap"`

	OrgID string `grove:"org_id,pk"`
	Mode  string `grove:"mode,notnull"`
	// MaxGrantTTL is persisted as an integer count of nanoseconds so the
	// time.Duration round-trips exactly regardless of backend.
	MaxGrantTTL   int64     `grove:"max_grant_ttl,notnull"`
	AllowedScopes string    `grove:"allowed_scopes,notnull"`
	CreatedAt     time.Time `grove:"created_at,notnull,default:now()"`
	UpdatedAt     time.Time `grove:"updated_at,notnull,default:now()"`
}

// ──────────────────────────────────────────────────
// Agent converters
// ──────────────────────────────────────────────────

func toAgent(m *agentModel) (*Agent, error) {
	agentID, err := id.ParseAgentID(m.ID)
	if err != nil {
		return nil, err
	}
	appID, err := id.ParseAppID(m.AppID)
	if err != nil {
		return nil, err
	}

	a := &Agent{
		ID:          agentID,
		AppID:       appID,
		ClientID:    m.ClientID,
		Name:        m.Name,
		Description: m.Description,
		LogoURI:     m.LogoURI,
		Origin:      AgentOrigin(m.Origin),
		Status:      AgentStatus(m.Status),
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
	if m.OrgID != "" {
		orgID, err := id.ParseOrgID(m.OrgID)
		if err != nil {
			return nil, err
		}
		a.OrgID = orgID
	}
	if m.CreatedBy != "" {
		createdBy, err := id.ParseUserID(m.CreatedBy)
		if err != nil {
			return nil, err
		}
		a.CreatedBy = createdBy
	}
	return a, nil
}

func fromAgent(a *Agent) *agentModel {
	m := &agentModel{
		ID:          a.ID.String(),
		AppID:       a.AppID.String(),
		ClientID:    a.ClientID,
		Name:        a.Name,
		Description: a.Description,
		LogoURI:     a.LogoURI,
		Origin:      string(a.Origin),
		Status:      string(a.Status),
		CreatedAt:   utc(a.CreatedAt),
		UpdatedAt:   utc(a.UpdatedAt),
	}
	if !a.OrgID.IsNil() {
		m.OrgID = a.OrgID.String()
	}
	if !a.CreatedBy.IsNil() {
		m.CreatedBy = a.CreatedBy.String()
	}
	return m
}

// ──────────────────────────────────────────────────
// Agent grant converters
// ──────────────────────────────────────────────────

func toAgentGrant(m *agentGrantModel) (*AgentGrant, error) {
	grantID, err := id.ParseAgentGrantID(m.ID)
	if err != nil {
		return nil, err
	}
	appID, err := id.ParseAppID(m.AppID)
	if err != nil {
		return nil, err
	}
	agentID, err := id.ParseAgentID(m.AgentID)
	if err != nil {
		return nil, err
	}
	userID, err := id.ParseUserID(m.UserID)
	if err != nil {
		return nil, err
	}

	g := &AgentGrant{
		ID:        grantID,
		AppID:     appID,
		AgentID:   agentID,
		UserID:    userID,
		ExpiresAt: m.ExpiresAt,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
	if m.OrgID != "" {
		orgID, err := id.ParseOrgID(m.OrgID)
		if err != nil {
			return nil, err
		}
		g.OrgID = orgID
	}
	if m.ConsentID != "" {
		consentID, err := id.ParseConsentID(m.ConsentID)
		if err != nil {
			return nil, err
		}
		g.ConsentID = consentID
	}
	// Only decode when non-empty: matches MemoryStore, where a nil or empty
	// Scopes slice round-trips as nil (append(nil) with no elements is nil),
	// never as a distinguishable empty-vs-nil state callers must special-case.
	if m.Scopes != "" && m.Scopes != "[]" {
		var scopes []string
		if err := json.Unmarshal([]byte(m.Scopes), &scopes); err != nil {
			return nil, err
		}
		g.Scopes = scopes
	}
	if m.LastUsedAt != nil {
		t := *m.LastUsedAt
		g.LastUsedAt = &t
	}
	if m.RevokedAt != nil {
		t := *m.RevokedAt
		g.RevokedAt = &t
	}
	return g, nil
}

func fromAgentGrant(g *AgentGrant) (*agentGrantModel, error) {
	scopesJSON := "[]"
	if len(g.Scopes) > 0 {
		b, err := json.Marshal(g.Scopes)
		if err != nil {
			return nil, err
		}
		scopesJSON = string(b)
	}

	m := &agentGrantModel{
		ID:        g.ID.String(),
		AppID:     g.AppID.String(),
		AgentID:   g.AgentID.String(),
		UserID:    g.UserID.String(),
		Scopes:    scopesJSON,
		ExpiresAt: utc(g.ExpiresAt),
		CreatedAt: utc(g.CreatedAt),
		UpdatedAt: utc(g.UpdatedAt),
	}
	if !g.OrgID.IsNil() {
		m.OrgID = g.OrgID.String()
	}
	if !g.ConsentID.IsNil() {
		m.ConsentID = g.ConsentID.String()
	}
	if g.LastUsedAt != nil {
		t := utc(*g.LastUsedAt)
		m.LastUsedAt = &t
	}
	if g.RevokedAt != nil {
		t := utc(*g.RevokedAt)
		m.RevokedAt = &t
	}
	return m, nil
}

// utc strips any monotonic clock reading from t (the same way t.UTC() does)
// so a time.Time built from a bare time.Now() can round-trip through a
// TEXT-affinity SQLite column. modernc/sqlite serializes time.Time
// arguments via time.Time.String() when the value carries a monotonic
// reading, which appends a " m=+..." suffix that grove's own timeLayouts
// can't parse back — see grove/scan/convert.go. Normalizing every
// timestamp at the write boundary, rather than trusting every caller
// across the codebase to remember .UTC(), is the one choke point that
// closes this for good.
func utc(t time.Time) time.Time { return t.UTC() }

// ──────────────────────────────────────────────────
// Org policy converters
// ──────────────────────────────────────────────────

func toOrgPolicy(m *orgPolicyModel) (*OrgAgentPolicy, error) {
	orgID, err := id.ParseOrgID(m.OrgID)
	if err != nil {
		return nil, err
	}
	p := &OrgAgentPolicy{
		OrgID:       orgID,
		Mode:        PolicyMode(m.Mode),
		MaxGrantTTL: time.Duration(m.MaxGrantTTL),
	}
	if m.AllowedScopes != "" && m.AllowedScopes != "[]" {
		var scopes []string
		if err := json.Unmarshal([]byte(m.AllowedScopes), &scopes); err != nil {
			return nil, err
		}
		p.AllowedScopes = scopes
	}
	return p, nil
}

func fromOrgPolicy(p *OrgAgentPolicy) (*orgPolicyModel, error) {
	scopesJSON := "[]"
	if len(p.AllowedScopes) > 0 {
		b, err := json.Marshal(p.AllowedScopes)
		if err != nil {
			return nil, err
		}
		scopesJSON = string(b)
	}
	return &orgPolicyModel{
		OrgID:         p.OrgID.String(),
		Mode:          string(p.Mode),
		MaxGrantTTL:   int64(p.MaxGrantTTL),
		AllowedScopes: scopesJSON,
	}, nil
}

// ──────────────────────────────────────────────────
// Shared validation
// ──────────────────────────────────────────────────

// validatePolicyMode mirrors MemoryStore.PutOrgPolicy's mode check so every
// backend refuses the same malformed writes identically — a policy with an
// unrecognized mode must never reach any store, SQL or otherwise, because
// Evaluate and CreateGrant both treat that case as a deny and the safer
// invariant is that bad data can't exist to be misread in the first place.
func validatePolicyMode(mode PolicyMode) error {
	switch mode {
	case ModeOpen, ModeAllowlist, ModeBlocked:
		return nil
	default:
		return fmt.Errorf("agentauth: invalid policy mode %q", mode)
	}
}
