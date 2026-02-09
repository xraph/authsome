package handlers

// Request DTOs for CMS handlers

// Content Type DTOs.
type ListContentTypesRequest struct {
	Search    string `query:"search,omitempty"`
	SortBy    string `query:"sortBy,omitempty"`
	SortOrder string `query:"sortOrder,omitempty"`
	Page      int    `query:"page,omitempty"`
	PageSize  int    `query:"pageSize,omitempty"`
}

type GetContentTypeRequest struct {
	Slug string `path:"slug" validate:"required"`
}

type UpdateContentTypeRequest struct {
	Slug string `path:"slug" validate:"required"`
}

type DeleteContentTypeRequest struct {
	Slug string `path:"slug" validate:"required"`
}

type PublishContentTypeRequest struct {
	Slug string `path:"slug" validate:"required"`
}

type UnpublishContentTypeRequest struct {
	Slug string `path:"slug" validate:"required"`
}

type CloneContentTypeRequest struct {
	Slug string `path:"slug" validate:"required"`
}

type ValidateContentTypeRequest struct {
	Slug string `path:"slug" validate:"required"`
}

type AddFieldRequest struct {
	Slug string `path:"slug" validate:"required"`
}

type UpdateFieldRequest struct {
	Slug    string `path:"slug"    validate:"required"`
	FieldID string `path:"fieldId" validate:"required"`
}

type DeleteFieldRequest struct {
	Slug    string `path:"slug"    validate:"required"`
	FieldID string `path:"fieldId" validate:"required"`
}

type ReorderFieldsRequest struct {
	Slug string `path:"slug" validate:"required"`
}

// Content Entry DTOs.
type ListEntriesRequest struct {
	TypeSlug  string `path:"typeSlug"             validate:"required"`
	Search    string `query:"search,omitempty"`
	Status    string `query:"status,omitempty"`
	SortBy    string `query:"sortBy,omitempty"`
	SortOrder string `query:"sortOrder,omitempty"`
	Page      int    `query:"page,omitempty"`
	PageSize  int    `query:"pageSize,omitempty"`
}

type CreateEntryRequest struct {
	TypeSlug string `path:"typeSlug" validate:"required"`
}

type GetEntryRequest struct {
	TypeSlug string `path:"typeSlug" validate:"required"`
	EntryID  string `path:"entryId"  validate:"required"`
}

type UpdateEntryRequest struct {
	TypeSlug string `path:"typeSlug" validate:"required"`
	EntryID  string `path:"entryId"  validate:"required"`
}

type DeleteEntryRequest struct {
	TypeSlug string `path:"typeSlug" validate:"required"`
	EntryID  string `path:"entryId"  validate:"required"`
}

type PublishEntryRequest struct {
	TypeSlug string `path:"typeSlug" validate:"required"`
	EntryID  string `path:"entryId"  validate:"required"`
}

type UnpublishEntryRequest struct {
	TypeSlug string `path:"typeSlug" validate:"required"`
	EntryID  string `path:"entryId"  validate:"required"`
}

type ArchiveEntryRequest struct {
	TypeSlug string `path:"typeSlug" validate:"required"`
	EntryID  string `path:"entryId"  validate:"required"`
}

type RestoreEntryRequest struct {
	TypeSlug string `path:"typeSlug" validate:"required"`
	EntryID  string `path:"entryId"  validate:"required"`
}

type BulkPublishRequest struct {
	TypeSlug string   `path:"typeSlug" validate:"required"`
	IDs      []string `json:"ids"      validate:"required"`
}

type BulkUnpublishRequest struct {
	TypeSlug string   `path:"typeSlug" validate:"required"`
	IDs      []string `json:"ids"      validate:"required"`
}

type BulkDeleteRequest struct {
	TypeSlug string   `path:"typeSlug" validate:"required"`
	IDs      []string `json:"ids"      validate:"required"`
}

type QueryEntriesRequest struct {
	TypeSlug string `path:"typeSlug" validate:"required"`
}

type GetEntryStatsRequest struct {
	TypeSlug string `path:"typeSlug" validate:"required"`
}

// Revision DTOs.
type ListRevisionsRequest struct {
	TypeSlug string `path:"typeSlug"            validate:"required"`
	EntryID  string `path:"entryId"             validate:"required"`
	Page     int    `query:"page,omitempty"`
	PageSize int    `query:"pageSize,omitempty"`
}

type GetRevisionRequest struct {
	TypeSlug   string `path:"typeSlug"   validate:"required"`
	EntryID    string `path:"entryId"    validate:"required"`
	RevisionID string `path:"revisionId" validate:"required"`
}

type CompareRevisionsRequest struct {
	TypeSlug   string `path:"typeSlug"   validate:"required"`
	EntryID    string `path:"entryId"    validate:"required"`
	RevisionID string `path:"revisionId" validate:"required"`
	CompareID  string `query:"compareId" validate:"required"`
}

type RestoreRevisionRequest struct {
	TypeSlug   string `path:"typeSlug"   validate:"required"`
	EntryID    string `path:"entryId"    validate:"required"`
	RevisionID string `path:"revisionId" validate:"required"`
}
