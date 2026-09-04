package retention

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/xraph/grove"

	"github.com/xraph/authsome/id"
)

// contactRefModel and jobModel are shared between the SQL-backed stores
// (SQLite and Postgres); each store constructs its own query but maps rows
// through the same fromX/toX pair here.

type contactRefModel struct {
	grove.BaseModel `grove:"table:authsome_retention_contact_ref,alias:rtr"`

	ID               string    `grove:"id,pk"`
	AppID            string    `grove:"app_id,notnull"`
	EnvID            string    `grove:"env_id,notnull"`
	UserID           string    `grove:"user_id,notnull"`
	Provider         string    `grove:"provider,notnull"`
	RemoteObjectType string    `grove:"remote_object_type,notnull"`
	RemoteID         string    `grove:"remote_id,notnull"`
	SyncedAt         time.Time `grove:"synced_at,notnull"`
}

type jobModel struct {
	grove.BaseModel `grove:"table:authsome_retention_outbox,alias:rto"`

	ID             string       `grove:"id,pk"`
	AppID          string       `grove:"app_id,notnull"`
	EnvID          string       `grove:"env_id,notnull"`
	UserID         string       `grove:"user_id,notnull"`
	Provider       string       `grove:"provider,notnull"`
	Kind           string       `grove:"kind,notnull"`
	Payload        string       `grove:"payload,notnull"`
	IdempotencyKey string       `grove:"idempotency_key,notnull"`
	State          string       `grove:"state,notnull"`
	Attempts       int          `grove:"attempts,notnull"`
	NextAttemptAt  time.Time    `grove:"next_attempt_at,notnull"`
	InFlightUntil  sql.NullTime `grove:"in_flight_until"`
	LastError      string       `grove:"last_error,notnull"`
	CreatedAt      time.Time    `grove:"created_at,notnull"`
}

func fromJob(j *Job) *jobModel {
	payload, _ := json.Marshal(j.Payload)
	m := &jobModel{
		ID: j.ID.String(), AppID: j.AppID.String(), EnvID: j.EnvID.String(),
		UserID: j.UserID.String(), Provider: j.Provider, Kind: j.Kind,
		Payload: string(payload), IdempotencyKey: j.IdempotencyKey,
		State: j.State, Attempts: j.Attempts,
		// .UTC(): SQLite has no timestamp type, so these columns are TEXT and
		// a later WHERE compares them as strings (see SqliteStore.ClaimDue).
		// A value written with a local offset would sort wrong against one
		// written in UTC. Postgres columns are TIMESTAMPTZ and normalise
		// regardless, so calling .UTC() here is harmless there too, and
		// keeps both backends persisting the same representation since this
		// model is shared between them.
		NextAttemptAt: j.NextAttemptAt.UTC(), LastError: j.LastError, CreatedAt: j.CreatedAt.UTC(),
	}
	if !j.InFlightUntil.IsZero() {
		m.InFlightUntil = sql.NullTime{Time: j.InFlightUntil.UTC(), Valid: true}
	}
	return m
}

func toJob(m *jobModel) (*Job, error) {
	jobID, err := id.ParseRetentionJobID(m.ID)
	if err != nil {
		return nil, err
	}
	appID, err := id.ParseAppID(m.AppID)
	if err != nil {
		return nil, err
	}
	userID, err := id.ParseUserID(m.UserID)
	if err != nil {
		return nil, err
	}
	j := &Job{
		ID: jobID, AppID: appID, UserID: userID, Provider: m.Provider, Kind: m.Kind,
		IdempotencyKey: m.IdempotencyKey, State: m.State, Attempts: m.Attempts,
		NextAttemptAt: m.NextAttemptAt, LastError: m.LastError, CreatedAt: m.CreatedAt,
	}
	if m.InFlightUntil.Valid {
		j.InFlightUntil = m.InFlightUntil.Time
	}
	// The empty environment is the zero id.EnvironmentID, whose String() is
	// "". ParseEnvironmentID("") would fail, so only parse a non-empty column.
	if m.EnvID != "" {
		envID, err := id.ParseEnvironmentID(m.EnvID)
		if err != nil {
			return nil, err
		}
		j.EnvID = envID
	}
	j.Payload = make(map[string]string)
	if m.Payload != "" {
		if err := json.Unmarshal([]byte(m.Payload), &j.Payload); err != nil {
			return nil, err
		}
	}
	return j, nil
}

func fromRef(r *ContactRef) *contactRefModel {
	return &contactRefModel{
		ID: r.ID.String(), AppID: r.AppID.String(), EnvID: r.EnvID.String(),
		UserID: r.UserID.String(), Provider: r.Provider,
		RemoteObjectType: r.RemoteObjectType, RemoteID: r.RemoteID,
		// .UTC(): see fromJob's comment.
		SyncedAt: r.SyncedAt.UTC(),
	}
}

func toRef(m *contactRefModel) (*ContactRef, error) {
	refID, err := id.ParseRetentionRefID(m.ID)
	if err != nil {
		return nil, err
	}
	appID, err := id.ParseAppID(m.AppID)
	if err != nil {
		return nil, err
	}
	userID, err := id.ParseUserID(m.UserID)
	if err != nil {
		return nil, err
	}
	r := &ContactRef{
		ID: refID, AppID: appID, UserID: userID, Provider: m.Provider,
		RemoteObjectType: m.RemoteObjectType, RemoteID: m.RemoteID, SyncedAt: m.SyncedAt,
	}
	// Same empty-environment guard as toJob: ParseEnvironmentID("") fails, so
	// only parse a non-empty column.
	if m.EnvID != "" {
		envID, err := id.ParseEnvironmentID(m.EnvID)
		if err != nil {
			return nil, err
		}
		r.EnvID = envID
	}
	return r, nil
}
