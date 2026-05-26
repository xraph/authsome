// handlers_webhooks.go: Phase C.8 — Webhooks dashboard.
package contract

import (
	"context"
	"strings"
	"time"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/webhook"

	"github.com/xraph/forge/extensions/dashboard/contract"
)

type WebhookSummary struct {
	ID        string   `json:"id"`
	URL       string   `json:"url"`
	Events    []string `json:"events"`
	Active    bool     `json:"active"`
	CreatedAt string   `json:"createdAt"`
}

type WebhookDetail struct {
	WebhookSummary
	AppID     string `json:"appId,omitempty"`
	EnvID     string `json:"envId,omitempty"`
	UpdatedAt string `json:"updatedAt"`
}

type GetWebhookInput struct {
	ID string `json:"id"`
}

type WebhookListResponse struct {
	Webhooks []WebhookSummary `json:"webhooks"`
}
type CreateWebhookInput struct {
	URL    string   `json:"url"`
	Events []string `json:"events"`
}
type UpdateWebhookInput struct {
	ID     string    `json:"id"`
	URL    *string   `json:"url,omitempty"`
	Events *[]string `json:"events,omitempty"`
	Active *bool     `json:"active,omitempty"`
}
type DeleteWebhookInput struct {
	ID string `json:"id"`
}

func webhooksListHandler(deps Deps) func(ctx context.Context, _ struct{}, _ contract.Principal) (WebhookListResponse, error) {
	return func(ctx context.Context, _ struct{}, _ contract.Principal) (WebhookListResponse, error) {
		if deps.Engine == nil {
			return WebhookListResponse{}, &contract.Error{Code: contract.CodeUnavailable, Message: "auth engine not configured"}
		}
		list, err := deps.Engine.ListWebhooks(ctx, defaultAppID(deps.Engine))
		if err != nil {
			return WebhookListResponse{}, mapEngineError(err)
		}
		out := WebhookListResponse{Webhooks: make([]WebhookSummary, 0, len(list))}
		for _, w := range list {
			out.Webhooks = append(out.Webhooks, projectWebhook(w))
		}
		return out, nil
	}
}

func webhooksCreateHandler(deps Deps) func(ctx context.Context, in CreateWebhookInput, _ contract.Principal) (AckResponse, error) {
	return func(ctx context.Context, in CreateWebhookInput, _ contract.Principal) (AckResponse, error) {
		if deps.Engine == nil {
			return AckResponse{}, &contract.Error{Code: contract.CodeUnavailable, Message: "auth engine not configured"}
		}
		url := strings.TrimSpace(in.URL)
		if url == "" {
			return AckResponse{}, &contract.Error{Code: contract.CodeBadRequest, Message: "url is required"}
		}
		if len(in.Events) == 0 {
			return AckResponse{}, &contract.Error{Code: contract.CodeBadRequest, Message: "at least one event is required"}
		}
		w := &webhook.Webhook{
			AppID: defaultAppID(deps.Engine),
			URL:   url, Events: in.Events, Active: true,
		}
		if err := deps.Engine.CreateWebhook(ctx, w); err != nil {
			return AckResponse{}, mapEngineError(err)
		}
		return AckResponse{OK: true, ID: w.ID.String()}, nil
	}
}

func webhooksUpdateHandler(deps Deps) func(ctx context.Context, in UpdateWebhookInput, _ contract.Principal) (AckResponse, error) {
	return func(ctx context.Context, in UpdateWebhookInput, _ contract.Principal) (AckResponse, error) {
		if deps.Engine == nil {
			return AckResponse{}, &contract.Error{Code: contract.CodeUnavailable, Message: "auth engine not configured"}
		}
		wid, err := parseWebhookID(in.ID)
		if err != nil {
			return AckResponse{}, err
		}
		current, err := deps.Engine.GetWebhook(ctx, wid)
		if err != nil {
			return AckResponse{}, mapEngineError(err)
		}
		if in.URL != nil {
			current.URL = *in.URL
		}
		if in.Events != nil {
			current.Events = *in.Events
		}
		if in.Active != nil {
			current.Active = *in.Active
		}
		if err := deps.Engine.UpdateWebhook(ctx, current); err != nil {
			return AckResponse{}, mapEngineError(err)
		}
		return AckResponse{OK: true, ID: wid.String()}, nil
	}
}

func webhooksDeleteHandler(deps Deps) func(ctx context.Context, in DeleteWebhookInput, _ contract.Principal) (AckResponse, error) {
	return func(ctx context.Context, in DeleteWebhookInput, _ contract.Principal) (AckResponse, error) {
		if deps.Engine == nil {
			return AckResponse{}, &contract.Error{Code: contract.CodeUnavailable, Message: "auth engine not configured"}
		}
		wid, err := parseWebhookID(in.ID)
		if err != nil {
			return AckResponse{}, err
		}
		if err := deps.Engine.DeleteWebhook(ctx, wid); err != nil {
			return AckResponse{}, mapEngineError(err)
		}
		return AckResponse{OK: true, ID: wid.String()}, nil
	}
}

func webhooksDetailHandler(deps Deps) func(ctx context.Context, in GetWebhookInput, _ contract.Principal) (WebhookDetail, error) {
	return func(ctx context.Context, in GetWebhookInput, _ contract.Principal) (WebhookDetail, error) {
		if deps.Engine == nil {
			return WebhookDetail{}, &contract.Error{Code: contract.CodeUnavailable, Message: "auth engine not configured"}
		}
		wid, err := parseWebhookID(in.ID)
		if err != nil {
			return WebhookDetail{}, err
		}
		w, err := deps.Engine.GetWebhook(ctx, wid)
		if err != nil {
			return WebhookDetail{}, mapEngineError(err)
		}
		return WebhookDetail{
			WebhookSummary: projectWebhook(w),
			AppID:          w.AppID.String(),
			EnvID:          w.EnvID.String(),
			UpdatedAt:      w.UpdatedAt.UTC().Format(time.RFC3339),
		}, nil
	}
}

func projectWebhook(w *webhook.Webhook) WebhookSummary {
	if w == nil {
		return WebhookSummary{}
	}
	return WebhookSummary{
		ID: w.ID.String(), URL: w.URL, Events: w.Events, Active: w.Active,
		CreatedAt: w.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func parseWebhookID(s string) (id.WebhookID, error) {
	if strings.TrimSpace(s) == "" {
		return id.WebhookID{}, &contract.Error{Code: contract.CodeBadRequest, Message: "id is required"}
	}
	wid, err := id.ParseWebhookID(s)
	if err != nil {
		return id.WebhookID{}, &contract.Error{Code: contract.CodeBadRequest, Message: "invalid webhook id: " + err.Error()}
	}
	return wid, nil
}
