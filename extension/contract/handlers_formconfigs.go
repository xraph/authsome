// handlers_formconfigs.go: Phase C.9 — Form Configs dashboard.
//
// Surfaces the signup-form customization the legacy templ dashboard
// exposed via /signup-forms. Today the engine only ships
// SignupFormConfig methods; other form types can layer on later.
package contract

import (
	"context"
	"time"

	"github.com/xraph/authsome/formconfig"

	"github.com/xraph/forge/extensions/dashboard/contract"
)

type FormConfigSummary struct {
	ID        string `json:"id"`
	FormType  string `json:"formType"`
	Version   int    `json:"version"`
	Active    bool   `json:"active"`
	CreatedAt string `json:"createdAt"`
}

type FormConfigDetail struct {
	FormConfigSummary
	AppID     string              `json:"appId,omitempty"`
	Fields    []formconfig.FormField `json:"fields,omitempty"`
	UpdatedAt string              `json:"updatedAt"`
}

type FormConfigListResponse struct {
	FormConfigs []FormConfigSummary `json:"formConfigs"`
}

type SaveSignupFormInput struct {
	Fields []formconfig.FormField `json:"fields"`
	Active bool                   `json:"active"`
}

type DeleteSignupFormInput struct{}

func formConfigsListHandler(deps Deps) func(ctx context.Context, _ struct{}, _ contract.Principal) (FormConfigListResponse, error) {
	return func(ctx context.Context, _ struct{}, _ contract.Principal) (FormConfigListResponse, error) {
		if deps.Engine == nil {
			return FormConfigListResponse{}, &contract.Error{Code: contract.CodeUnavailable, Message: "auth engine not configured"}
		}
		list, err := deps.Engine.ListFormConfigs(ctx, defaultAppID(deps.Engine))
		if err != nil {
			return FormConfigListResponse{}, mapEngineError(err)
		}
		out := FormConfigListResponse{FormConfigs: make([]FormConfigSummary, 0, len(list))}
		for _, fc := range list {
			out.FormConfigs = append(out.FormConfigs, projectFormConfigSummary(fc))
		}
		return out, nil
	}
}

func formConfigsSignupDetailHandler(deps Deps) func(ctx context.Context, _ struct{}, _ contract.Principal) (FormConfigDetail, error) {
	return func(ctx context.Context, _ struct{}, _ contract.Principal) (FormConfigDetail, error) {
		if deps.Engine == nil {
			return FormConfigDetail{}, &contract.Error{Code: contract.CodeUnavailable, Message: "auth engine not configured"}
		}
		fc, err := deps.Engine.GetSignupFormConfig(ctx, defaultAppID(deps.Engine))
		if err != nil {
			return FormConfigDetail{}, mapEngineError(err)
		}
		return projectFormConfigDetail(fc), nil
	}
}

func formConfigsSignupSaveHandler(deps Deps) func(ctx context.Context, in SaveSignupFormInput, _ contract.Principal) (AckResponse, error) {
	return func(ctx context.Context, in SaveSignupFormInput, _ contract.Principal) (AckResponse, error) {
		if deps.Engine == nil {
			return AckResponse{}, &contract.Error{Code: contract.CodeUnavailable, Message: "auth engine not configured"}
		}
		fc := &formconfig.FormConfig{
			AppID:    defaultAppID(deps.Engine),
			FormType: "signup",
			Fields:   in.Fields,
			Active:   in.Active,
		}
		if err := deps.Engine.SaveSignupFormConfig(ctx, fc); err != nil {
			return AckResponse{}, mapEngineError(err)
		}
		return AckResponse{OK: true, ID: fc.ID.String()}, nil
	}
}

func formConfigsSignupDeleteHandler(deps Deps) func(ctx context.Context, _ DeleteSignupFormInput, _ contract.Principal) (AckResponse, error) {
	return func(ctx context.Context, _ DeleteSignupFormInput, _ contract.Principal) (AckResponse, error) {
		if deps.Engine == nil {
			return AckResponse{}, &contract.Error{Code: contract.CodeUnavailable, Message: "auth engine not configured"}
		}
		if err := deps.Engine.DeleteSignupFormConfig(ctx, defaultAppID(deps.Engine)); err != nil {
			return AckResponse{}, mapEngineError(err)
		}
		return AckResponse{OK: true}, nil
	}
}

func projectFormConfigSummary(fc *formconfig.FormConfig) FormConfigSummary {
	if fc == nil {
		return FormConfigSummary{}
	}
	return FormConfigSummary{
		ID: fc.ID.String(), FormType: fc.FormType,
		Version: fc.Version, Active: fc.Active,
		CreatedAt: fc.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func projectFormConfigDetail(fc *formconfig.FormConfig) FormConfigDetail {
	if fc == nil {
		return FormConfigDetail{}
	}
	return FormConfigDetail{
		FormConfigSummary: projectFormConfigSummary(fc),
		AppID:             fc.AppID.String(),
		Fields:            fc.Fields,
		UpdatedAt:         fc.UpdatedAt.UTC().Format(time.RFC3339),
	}
}
