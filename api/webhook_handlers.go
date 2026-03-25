package api

import (
	"fmt"
	"net/http"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/webhook"
)

// ──────────────────────────────────────────────────
// Webhook route registration
// ──────────────────────────────────────────────────

func (a *API) registerWebhookRoutes(router forge.Router) error {
	g := router.Group("/v1", forge.WithGroupTags("webhooks"))

	if err := g.POST("/webhooks", a.handleCreateWebhook,
		forge.WithSummary("Create webhook"),
		forge.WithDescription("Creates a new webhook endpoint registration."),
		forge.WithOperationID("createWebhook"),
		forge.WithRequestSchema(CreateWebhookRequest{}),
		forge.WithCreatedResponse(webhook.Webhook{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.GET("/webhooks", a.handleListWebhooks,
		forge.WithSummary("List webhooks"),
		forge.WithDescription("Returns all registered webhooks for an app."),
		forge.WithOperationID("listWebhooks"),
		forge.WithResponseSchema(http.StatusOK, "Webhook list", WebhookListResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.GET("/webhooks/:webhookId", a.handleGetWebhook,
		forge.WithSummary("Get webhook"),
		forge.WithDescription("Returns details of a specific webhook."),
		forge.WithOperationID("getWebhook"),
		forge.WithResponseSchema(http.StatusOK, "Webhook details", webhook.Webhook{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.PATCH("/webhooks/:webhookId", a.handleUpdateWebhook,
		forge.WithSummary("Update webhook"),
		forge.WithDescription("Updates a webhook's URL, events, or active status."),
		forge.WithOperationID("updateWebhook"),
		forge.WithRequestSchema(UpdateWebhookRequest{}),
		forge.WithResponseSchema(http.StatusOK, "Updated webhook", webhook.Webhook{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	return g.DELETE("/webhooks/:webhookId", a.handleDeleteWebhook,
		forge.WithSummary("Delete webhook"),
		forge.WithDescription("Deletes a webhook registration."),
		forge.WithOperationID("deleteWebhook"),
		forge.WithResponseSchema(http.StatusOK, "Deleted", StatusResponse{}),
		forge.WithErrorResponses(),
	)
}

// ──────────────────────────────────────────────────
// Webhook handlers
// ──────────────────────────────────────────────────

func (a *API) handleCreateWebhook(ctx forge.Context, req *CreateWebhookRequest) (*webhook.Webhook, error) {
	if req.URL == "" {
		return nil, forge.BadRequest("url is required")
	}
	if len(req.Events) == 0 {
		return nil, forge.BadRequest("at least one event type is required")
	}

	appID, err := a.resolveAppID(req.AppID)
	if err != nil {
		return nil, forge.BadRequest("invalid app_id")
	}

	w := &webhook.Webhook{
		AppID:  appID,
		URL:    req.URL,
		Events: req.Events,
	}

	if err := a.engine.CreateWebhook(ctx.Context(), w); err != nil {
		return nil, mapError(err)
	}

	return nil, ctx.JSON(http.StatusCreated, w)
}

func (a *API) handleListWebhooks(ctx forge.Context, req *ListWebhooksRequest) (*WebhookListResponse, error) {
	appID, err := a.resolveAppID(req.AppID)
	if err != nil {
		return nil, forge.BadRequest("invalid app_id")
	}

	webhooks, err := a.engine.ListWebhooks(ctx.Context(), appID)
	if err != nil {
		return nil, mapError(err)
	}

	if webhooks == nil {
		webhooks = []*webhook.Webhook{}
	}
	resp := &WebhookListResponse{Webhooks: webhooks}
	return nil, ctx.JSON(http.StatusOK, resp)
}

func (a *API) handleGetWebhook(ctx forge.Context, _ *GetWebhookRequest) (*webhook.Webhook, error) {
	webhookID, err := id.ParseWebhookID(ctx.Param("webhookId"))
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid webhook id: %v", err))
	}

	w, err := a.engine.GetWebhook(ctx.Context(), webhookID)
	if err != nil {
		return nil, mapError(err)
	}

	return w, nil
}

func (a *API) handleUpdateWebhook(ctx forge.Context, req *UpdateWebhookRequest) (*webhook.Webhook, error) {
	webhookID, err := id.ParseWebhookID(ctx.Param("webhookId"))
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid webhook id: %v", err))
	}

	w, err := a.engine.GetWebhook(ctx.Context(), webhookID)
	if err != nil {
		return nil, mapError(err)
	}

	if req.URL != nil {
		w.URL = *req.URL
	}
	if req.Events != nil {
		w.Events = req.Events
	}
	if req.Active != nil {
		w.Active = *req.Active
	}

	if err := a.engine.UpdateWebhook(ctx.Context(), w); err != nil {
		return nil, mapError(err)
	}

	return w, nil
}

func (a *API) handleDeleteWebhook(ctx forge.Context, _ *DeleteWebhookRequest) (*StatusResponse, error) {
	webhookID, err := id.ParseWebhookID(ctx.Param("webhookId"))
	if err != nil {
		return nil, forge.BadRequest(fmt.Sprintf("invalid webhook id: %v", err))
	}

	if err := a.engine.DeleteWebhook(ctx.Context(), webhookID); err != nil {
		return nil, mapError(err)
	}

	resp := &StatusResponse{Status: "deleted"}
	return nil, ctx.JSON(http.StatusOK, resp)
}
