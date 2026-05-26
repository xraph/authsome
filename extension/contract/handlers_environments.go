// handlers_environments.go: Phase C.7 — Environments dashboard.
package contract

import (
	"context"
	"strings"
	"time"

	"github.com/xraph/authsome/environment"
	"github.com/xraph/authsome/id"

	"github.com/xraph/forge/extensions/dashboard/contract"
)

type EnvSummary struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	Type      string `json:"type"`
	IsDefault bool   `json:"isDefault"`
	CreatedAt string `json:"createdAt"`
}

type EnvDetail struct {
	EnvSummary
	AppID       string            `json:"appId,omitempty"`
	Color       string            `json:"color,omitempty"`
	Description string            `json:"description,omitempty"`
	ClonedFrom  string            `json:"clonedFrom,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	UpdatedAt   string            `json:"updatedAt"`
}

type EnvListResponse struct{ Environments []EnvSummary `json:"environments"` }
type GetEnvInput struct{ ID string `json:"id"` }
type CreateEnvInput struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Type        string `json:"type,omitempty"`
	Description string `json:"description,omitempty"`
	Color       string `json:"color,omitempty"`
}
type UpdateEnvInput struct {
	ID          string  `json:"id"`
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Color       *string `json:"color,omitempty"`
}
type DeleteEnvInput struct{ ID string `json:"id"` }
type CloneEnvInput struct {
	SourceID string `json:"sourceId"`
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	Type     string `json:"type,omitempty"`
}
type SetDefaultEnvInput struct{ ID string `json:"id"` }

func envsListHandler(deps Deps) func(ctx context.Context, _ struct{}, _ contract.Principal) (EnvListResponse, error) {
	return func(ctx context.Context, _ struct{}, _ contract.Principal) (EnvListResponse, error) {
		if deps.Engine == nil {
			return EnvListResponse{}, &contract.Error{Code: contract.CodeUnavailable, Message: "auth engine not configured"}
		}
		list, err := deps.Engine.ListEnvironments(ctx, defaultAppID(deps.Engine))
		if err != nil {
			return EnvListResponse{}, mapEngineError(err)
		}
		out := EnvListResponse{Environments: make([]EnvSummary, 0, len(list))}
		for _, e := range list {
			out.Environments = append(out.Environments, projectEnvSummary(e))
		}
		return out, nil
	}
}

func envsDetailHandler(deps Deps) func(ctx context.Context, in GetEnvInput, _ contract.Principal) (EnvDetail, error) {
	return func(ctx context.Context, in GetEnvInput, _ contract.Principal) (EnvDetail, error) {
		if deps.Engine == nil {
			return EnvDetail{}, &contract.Error{Code: contract.CodeUnavailable, Message: "auth engine not configured"}
		}
		eid, err := parseEnvID(in.ID)
		if err != nil {
			return EnvDetail{}, err
		}
		e, err := deps.Engine.GetEnvironment(ctx, eid)
		if err != nil {
			return EnvDetail{}, mapEngineError(err)
		}
		return projectEnvDetail(e), nil
	}
}

func envsCreateHandler(deps Deps) func(ctx context.Context, in CreateEnvInput, _ contract.Principal) (AckResponse, error) {
	return func(ctx context.Context, in CreateEnvInput, _ contract.Principal) (AckResponse, error) {
		if deps.Engine == nil {
			return AckResponse{}, &contract.Error{Code: contract.CodeUnavailable, Message: "auth engine not configured"}
		}
		name := strings.TrimSpace(in.Name)
		slug := strings.TrimSpace(in.Slug)
		if name == "" || slug == "" {
			return AckResponse{}, &contract.Error{Code: contract.CodeBadRequest, Message: "name and slug are required"}
		}
		t := environment.Type(in.Type)
		if t == "" {
			t = environment.TypeDevelopment
		}
		e := &environment.Environment{
			AppID: defaultAppID(deps.Engine),
			Name:  name, Slug: slug, Type: t,
			Description: in.Description, Color: in.Color,
		}
		if err := deps.Engine.CreateEnvironment(ctx, e); err != nil {
			return AckResponse{}, mapEngineError(err)
		}
		return AckResponse{OK: true, ID: e.ID.String()}, nil
	}
}

func envsUpdateHandler(deps Deps) func(ctx context.Context, in UpdateEnvInput, _ contract.Principal) (AckResponse, error) {
	return func(ctx context.Context, in UpdateEnvInput, _ contract.Principal) (AckResponse, error) {
		if deps.Engine == nil {
			return AckResponse{}, &contract.Error{Code: contract.CodeUnavailable, Message: "auth engine not configured"}
		}
		eid, err := parseEnvID(in.ID)
		if err != nil {
			return AckResponse{}, err
		}
		current, err := deps.Engine.GetEnvironment(ctx, eid)
		if err != nil {
			return AckResponse{}, mapEngineError(err)
		}
		if in.Name != nil {
			current.Name = *in.Name
		}
		if in.Description != nil {
			current.Description = *in.Description
		}
		if in.Color != nil {
			current.Color = *in.Color
		}
		if err := deps.Engine.UpdateEnvironment(ctx, current); err != nil {
			return AckResponse{}, mapEngineError(err)
		}
		return AckResponse{OK: true, ID: eid.String()}, nil
	}
}

func envsDeleteHandler(deps Deps) func(ctx context.Context, in DeleteEnvInput, _ contract.Principal) (AckResponse, error) {
	return func(ctx context.Context, in DeleteEnvInput, _ contract.Principal) (AckResponse, error) {
		if deps.Engine == nil {
			return AckResponse{}, &contract.Error{Code: contract.CodeUnavailable, Message: "auth engine not configured"}
		}
		eid, err := parseEnvID(in.ID)
		if err != nil {
			return AckResponse{}, err
		}
		if err := deps.Engine.DeleteEnvironment(ctx, eid); err != nil {
			return AckResponse{}, mapEngineError(err)
		}
		return AckResponse{OK: true, ID: eid.String()}, nil
	}
}

func envsCloneHandler(deps Deps) func(ctx context.Context, in CloneEnvInput, _ contract.Principal) (AckResponse, error) {
	return func(ctx context.Context, in CloneEnvInput, _ contract.Principal) (AckResponse, error) {
		if deps.Engine == nil {
			return AckResponse{}, &contract.Error{Code: contract.CodeUnavailable, Message: "auth engine not configured"}
		}
		src, err := parseEnvID(in.SourceID)
		if err != nil {
			return AckResponse{}, err
		}
		t := environment.Type(in.Type)
		if t == "" {
			t = environment.TypeDevelopment
		}
		res, err := deps.Engine.CloneEnvironment(ctx, environment.CloneRequest{
			SourceEnvID: src, Name: in.Name, Slug: in.Slug, Type: t,
		})
		if err != nil {
			return AckResponse{}, mapEngineError(err)
		}
		return AckResponse{OK: true, ID: res.Environment.ID.String()}, nil
	}
}

func envsSetDefaultHandler(deps Deps) func(ctx context.Context, in SetDefaultEnvInput, _ contract.Principal) (AckResponse, error) {
	return func(ctx context.Context, in SetDefaultEnvInput, _ contract.Principal) (AckResponse, error) {
		if deps.Engine == nil {
			return AckResponse{}, &contract.Error{Code: contract.CodeUnavailable, Message: "auth engine not configured"}
		}
		eid, err := parseEnvID(in.ID)
		if err != nil {
			return AckResponse{}, err
		}
		if err := deps.Engine.SetDefaultEnvironment(ctx, defaultAppID(deps.Engine), eid); err != nil {
			return AckResponse{}, mapEngineError(err)
		}
		return AckResponse{OK: true, ID: eid.String()}, nil
	}
}

func projectEnvSummary(e *environment.Environment) EnvSummary {
	if e == nil {
		return EnvSummary{}
	}
	return EnvSummary{
		ID: e.ID.String(), Name: e.Name, Slug: e.Slug,
		Type: string(e.Type), IsDefault: e.IsDefault,
		CreatedAt: e.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func projectEnvDetail(e *environment.Environment) EnvDetail {
	if e == nil {
		return EnvDetail{}
	}
	d := EnvDetail{
		EnvSummary:  projectEnvSummary(e),
		AppID:       e.AppID.String(),
		Color:       e.Color,
		Description: e.Description,
		Metadata:    e.Metadata,
		UpdatedAt:   e.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if !e.ClonedFrom.IsNil() {
		d.ClonedFrom = e.ClonedFrom.String()
	}
	return d
}

func parseEnvID(s string) (id.EnvironmentID, error) {
	if strings.TrimSpace(s) == "" {
		return id.EnvironmentID{}, &contract.Error{Code: contract.CodeBadRequest, Message: "id is required"}
	}
	eid, err := id.ParseEnvironmentID(s)
	if err != nil {
		return id.EnvironmentID{}, &contract.Error{Code: contract.CodeBadRequest, Message: "invalid environment id: " + err.Error()}
	}
	return eid, nil
}
