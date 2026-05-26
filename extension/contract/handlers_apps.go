// handlers_apps.go: Phase C.2 — Apps dashboard.
//
// Wraps engine.{List,Get,Create,Update,Delete}App. Apps are the
// top-level multi-tenancy unit in authsome; the dashboard surfaces
// them so platform admins can manage tenants without dropping into
// SQL. Pattern mirrors handlers_users.go.
package contract

import (
	"context"
	"strings"
	"time"

	"github.com/xraph/authsome/app"
	"github.com/xraph/authsome/id"

	"github.com/xraph/forge/extensions/dashboard/contract"
)

type AppSummary struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Slug       string `json:"slug"`
	IsPlatform bool   `json:"isPlatform"`
	CreatedAt  string `json:"createdAt"`
}

type AppDetail struct {
	AppSummary
	Logo           string            `json:"logo,omitempty"`
	PublishableKey string            `json:"publishableKey,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	UpdatedAt      string            `json:"updatedAt"`
}

type AppListResponse struct {
	Apps []AppSummary `json:"apps"`
}

type GetAppInput struct {
	ID string `json:"id"`
}

type CreateAppInput struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
	Logo string `json:"logo,omitempty"`
}

type UpdateAppInput struct {
	ID   string  `json:"id"`
	Name *string `json:"name,omitempty"`
	Slug *string `json:"slug,omitempty"`
	Logo *string `json:"logo,omitempty"`
}

type DeleteAppInput struct{ ID string `json:"id"` }

func appsListHandler(deps Deps) func(ctx context.Context, _ struct{}, _ contract.Principal) (AppListResponse, error) {
	return func(ctx context.Context, _ struct{}, _ contract.Principal) (AppListResponse, error) {
		if deps.Engine == nil {
			return AppListResponse{}, &contract.Error{Code: contract.CodeUnavailable, Message: "auth engine not configured"}
		}
		list, err := deps.Engine.ListApps(ctx)
		if err != nil {
			return AppListResponse{}, mapEngineError(err)
		}
		out := AppListResponse{Apps: make([]AppSummary, 0, len(list))}
		for _, a := range list {
			out.Apps = append(out.Apps, projectAppSummary(a))
		}
		return out, nil
	}
}

func appsDetailHandler(deps Deps) func(ctx context.Context, in GetAppInput, _ contract.Principal) (AppDetail, error) {
	return func(ctx context.Context, in GetAppInput, _ contract.Principal) (AppDetail, error) {
		if deps.Engine == nil {
			return AppDetail{}, &contract.Error{Code: contract.CodeUnavailable, Message: "auth engine not configured"}
		}
		aid, err := parseAppID(in.ID)
		if err != nil {
			return AppDetail{}, err
		}
		a, err := deps.Engine.GetApp(ctx, aid)
		if err != nil {
			return AppDetail{}, mapEngineError(err)
		}
		return projectAppDetail(a), nil
	}
}

func appsCreateHandler(deps Deps) func(ctx context.Context, in CreateAppInput, _ contract.Principal) (AckResponse, error) {
	return func(ctx context.Context, in CreateAppInput, _ contract.Principal) (AckResponse, error) {
		if deps.Engine == nil {
			return AckResponse{}, &contract.Error{Code: contract.CodeUnavailable, Message: "auth engine not configured"}
		}
		name := strings.TrimSpace(in.Name)
		slug := strings.TrimSpace(in.Slug)
		if name == "" || slug == "" {
			return AckResponse{}, &contract.Error{Code: contract.CodeBadRequest, Message: "name and slug are required"}
		}
		a := &app.App{Name: name, Slug: slug, Logo: in.Logo}
		if err := deps.Engine.CreateApp(ctx, a); err != nil {
			return AckResponse{}, mapEngineError(err)
		}
		return AckResponse{OK: true, ID: a.ID.String()}, nil
	}
}

func appsUpdateHandler(deps Deps) func(ctx context.Context, in UpdateAppInput, _ contract.Principal) (AckResponse, error) {
	return func(ctx context.Context, in UpdateAppInput, _ contract.Principal) (AckResponse, error) {
		if deps.Engine == nil {
			return AckResponse{}, &contract.Error{Code: contract.CodeUnavailable, Message: "auth engine not configured"}
		}
		aid, err := parseAppID(in.ID)
		if err != nil {
			return AckResponse{}, err
		}
		// Engine.UpdateApp takes the full struct; merge user edits onto
		// the current value so unset fields stay intact.
		current, err := deps.Engine.GetApp(ctx, aid)
		if err != nil {
			return AckResponse{}, mapEngineError(err)
		}
		if in.Name != nil {
			current.Name = *in.Name
		}
		if in.Slug != nil {
			current.Slug = *in.Slug
		}
		if in.Logo != nil {
			current.Logo = *in.Logo
		}
		if err := deps.Engine.UpdateApp(ctx, current); err != nil {
			return AckResponse{}, mapEngineError(err)
		}
		return AckResponse{OK: true, ID: aid.String()}, nil
	}
}

func appsDeleteHandler(deps Deps) func(ctx context.Context, in DeleteAppInput, _ contract.Principal) (AckResponse, error) {
	return func(ctx context.Context, in DeleteAppInput, _ contract.Principal) (AckResponse, error) {
		if deps.Engine == nil {
			return AckResponse{}, &contract.Error{Code: contract.CodeUnavailable, Message: "auth engine not configured"}
		}
		aid, err := parseAppID(in.ID)
		if err != nil {
			return AckResponse{}, err
		}
		if err := deps.Engine.DeleteApp(ctx, aid); err != nil {
			return AckResponse{}, mapEngineError(err)
		}
		return AckResponse{OK: true, ID: aid.String()}, nil
	}
}

func projectAppSummary(a *app.App) AppSummary {
	if a == nil {
		return AppSummary{}
	}
	return AppSummary{
		ID: a.ID.String(), Name: a.Name, Slug: a.Slug,
		IsPlatform: a.IsPlatform,
		CreatedAt:  a.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func projectAppDetail(a *app.App) AppDetail {
	if a == nil {
		return AppDetail{}
	}
	return AppDetail{
		AppSummary:     projectAppSummary(a),
		Logo:           a.Logo,
		PublishableKey: a.PublishableKey,
		Metadata:       a.Metadata,
		UpdatedAt:      a.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func parseAppID(s string) (id.AppID, error) {
	if strings.TrimSpace(s) == "" {
		return id.AppID{}, &contract.Error{Code: contract.CodeBadRequest, Message: "id is required"}
	}
	aid, err := id.ParseAppID(s)
	if err != nil {
		return id.AppID{}, &contract.Error{Code: contract.CodeBadRequest, Message: "invalid app id: " + err.Error()}
	}
	return aid, nil
}
