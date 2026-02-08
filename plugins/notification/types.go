package notification

// Request DTOs for notification handlers

// Template DTOs
type GetTemplateRequest struct {
	ID string `path:"id" validate:"required"`
}

type UpdateTemplateRequest struct {
	ID string `path:"id" validate:"required"`
}

type DeleteTemplateRequest struct {
	ID string `path:"id" validate:"required"`
}

type ResetTemplateRequest struct {
	ID string `path:"id" validate:"required"`
}

type PreviewTemplateRequest struct {
	ID string `path:"id" validate:"required"`
}

type RenderTemplateRequest struct {
	ID string `path:"id" validate:"required"`
}

// Notification DTOs
type GetNotificationRequest struct {
	ID string `path:"id" validate:"required"`
}

type ResendNotificationRequest struct {
	ID string `path:"id" validate:"required"`
}

// Provider DTOs
type GetProviderRequest struct {
	ID string `path:"id" validate:"required"`
}

type UpdateProviderRequest struct {
	ID string `path:"id" validate:"required"`
}

type DeleteProviderRequest struct {
	ID string `path:"id" validate:"required"`
}

// Template Version DTOs
type GetTemplateVersionRequest struct {
	TemplateID string `path:"templateId" validate:"required"`
	VersionID  string `path:"versionId" validate:"required"`
}

type RestoreTemplateVersionRequest struct {
	TemplateID string `path:"templateId" validate:"required"`
	VersionID  string `path:"versionId" validate:"required"`
}

// AB Test DTOs
type GetABTestResultsRequest struct {
	TemplateID string `path:"templateId" validate:"required"`
}

type DeclareABTestWinnerRequest struct {
	TemplateID string `path:"templateId" validate:"required"`
}

// Analytics DTOs
type GetTemplateAnalyticsRequest struct {
	TemplateID string `path:"templateId" validate:"required"`
}
