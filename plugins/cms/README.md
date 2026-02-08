# CMS Plugin for AuthSome

A comprehensive, headless Content Management System (CMS) plugin for the AuthSome authentication framework. This plugin provides a flexible, schema-based content management solution with multi-tenancy support, full-text search, revisions, and a complete dashboard UI.

## Features

### Core Features
- **Dynamic Content Types**: Define content structures with customizable fields
- **JSONB Storage**: Flexible schema-less data storage using PostgreSQL JSONB
- **Field Types**: 20+ field types including text, rich text, numbers, dates, media, relations, and more
- **Entry Management**: Full CRUD operations with draft/published/archived/scheduled workflows
- **Multi-tenancy**: App and environment scoped content

### Advanced Features
- **Relations**: Support for one-to-one, one-to-many, many-to-one, and many-to-many relations
- **Revisions**: Automatic version history with rollback capability
- **Full-text Search**: PostgreSQL-powered search with highlighting
- **Aggregations**: Count, sum, avg, min, max with grouping support
- **Query Language**: URL and JSON-based query API with filtering, sorting, and pagination

### Dashboard Integration
- **Content Types UI**: Create and manage content type schemas
- **Entries UI**: Dynamic form generation based on content type fields
- **Navigation**: Integrated into AuthSome dashboard sidebar
- **Widgets**: Dashboard statistics widget

## Installation

The CMS plugin is included in AuthSome. Enable it in your configuration:

```yaml
auth:
  plugins:
    cms:
      enabled: true
      features:
        enableRevisions: true
        enableDrafts: true
        enableSearch: true
        enableRelations: true
      limits:
        maxContentTypes: 50
        maxFieldsPerType: 100
        maxEntriesPerType: 10000
      revisions:
        maxRevisionsPerEntry: 50
        autoCleanup: true
```

## API Reference

The CMS plugin provides two types of endpoints:

### 1. API Endpoints (JSON) - For Programmatic Access

**Base URL:** `/api/identity/cms/*`

These endpoints return JSON data and are designed for external/programmatic access.

**Authentication:**
- Bearer token authentication required
- API key authentication supported

**Required Headers:**
```
X-App-ID: {your-app-id}
X-Environment-ID: {your-environment-id}
Authorization: Bearer {your-token}
```

### 2. UI Endpoints (HTML) - For Dashboard Only

**Base URL:** `/api/identity/ui/app/{appID}/cms/*`

These endpoints return HTML pages for the AuthSome dashboard interface. They are **not** intended for programmatic access.

---

## Content Type Endpoints

### List Content Types
```http
GET /api/identity/cms/types
```

**Query Parameters:**
- `search` - Search by name or slug
- `page` - Page number (default: 1)
- `pageSize` - Items per page (default: 20)

**Example Request:**
```bash
curl -X GET "http://localhost:4000/api/identity/cms/types?page=1&pageSize=20" \
  -H "X-App-ID: d5a0ktoh1fg1tihi9c4g" \
  -H "X-Environment-ID: production" \
  -H "Authorization: Bearer your-token-here"
```

**Example Response:**
```json
{
  "contentTypes": [
    {
      "id": "ck1234567890",
      "name": "blog-posts",
      "title": "Blog Posts",
      "description": "Blog post content",
      "icon": "📝",
      "fieldCount": 5,
      "entryCount": 42
    }
  ],
  "totalItems": 1,
  "page": 1,
  "pageSize": 20
}
```

### Create Content Type
```http
POST /api/identity/cms/types
```

**Request Body:**
```json
{
  "name": "blog-posts",
  "title": "Blog Posts",
  "description": "Blog post content",
  "icon": "📝"
}
```

### Get Content Type
```http
GET /api/identity/cms/types/{slug}
```

### Update Content Type
```http
PUT /api/identity/cms/types/{slug}
```

### Delete Content Type
```http
DELETE /api/identity/cms/types/{slug}
```

---

## Content Field Endpoints

### Add Field to Content Type
```http
POST /api/identity/cms/types/{slug}/fields
```

Request Body:
```json
{
  "name": "Title",
  "slug": "title",
  "type": "text",
  "required": true,
  "unique": false,
  "indexed": true,
  "options": {
    "minLength": 1,
    "maxLength": 200
  }
}
```

### Update Field
```http
PUT /api/identity/cms/types/{slug}/fields/{fieldSlug}
```

### Delete Field
```http
DELETE /api/identity/cms/types/{slug}/fields/{fieldSlug}
```

### Reorder Fields
```http
POST /api/identity/cms/types/{slug}/fields/reorder
```

**Request Body:**
```json
{
  "fieldOrder": ["field1", "field2", "field3"]
}
```

---

## Content Entry Endpoints

All content entry endpoints follow the pattern: `/api/identity/cms/{type}/*`

### List Entries
```http
GET /api/identity/cms/{typeSlug}
```

**Query Parameters:**
- `search` - Full-text search across content fields
- `status` - Filter by status: `draft`, `published`, `archived`, `scheduled`
- `page` - Page number (default: 1)
- `pageSize` - Items per page (default: 20, max: 100)
- `sort` - Sort field(s), prefix with `-` for descending (e.g., `-createdAt`)
- `filter[field]` - Filter by field value using operators

**Filter Syntax:**
- `filter[_meta.status]=eq.published` - Exact match
- `filter[title]=like.%hello%` - Pattern match
- `filter[age]=gt.18` - Greater than
- `filter[price]=lte.100` - Less than or equal
- `filter[category]=in.tech,science` - In array

**Available Operators:**
- `eq` - Equal
- `ne` - Not equal
- `gt` - Greater than
- `gte` - Greater than or equal
- `lt` - Less than
- `lte` - Less than or equal
- `like` - Pattern match (use `%` as wildcard)
- `ilike` - Case-insensitive pattern match
- `in` - In array (comma-separated values)

**Example: List Published Blog Posts**
```bash
curl -X GET "http://localhost:4000/api/identity/cms/blog-posts?filter[_meta.status]=eq.published&page=1&pageSize=10&sort=-createdAt" \
  -H "X-App-ID: d5a0ktoh1fg1tihi9c4g" \
  -H "X-Environment-ID: production" \
  -H "Authorization: Bearer your-token-here"
```

**Example Response:**
```json
{
  "entries": [
    {
      "id": "ck1234567890",
      "data": {
        "title": "My First Post",
        "content": "Hello, world!",
        "author": "John Doe"
      },
      "status": "published",
      "createdAt": "2024-01-15T10:30:00Z",
      "updatedAt": "2024-01-15T10:30:00Z",
      "publishedAt": "2024-01-15T12:00:00Z"
    }
  ],
  "totalItems": 42,
  "page": 1,
  "pageSize": 10
}
```

### Create Entry
```http
POST /api/identity/cms/{typeSlug}
```

**Request Body:**
```json
{
  "data": {
    "title": "My First Post",
    "content": "Hello, world!",
    "author": "John Doe"
  },
  "status": "draft"
}
```

**Example:**
```bash
curl -X POST "http://localhost:4000/api/identity/cms/blog-posts" \
  -H "X-App-ID: d5a0ktoh1fg1tihi9c4g" \
  -H "X-Environment-ID: production" \
  -H "Authorization: Bearer your-token-here" \
  -H "Content-Type: application/json" \
  -d '{
    "data": {
      "title": "My First Post",
      "content": "Hello, world!"
    },
    "status": "draft"
  }'
```

### Get Entry
```http
GET /api/identity/cms/{typeSlug}/{entryId}
```

**Example:**
```bash
curl -X GET "http://localhost:4000/api/identity/cms/blog-posts/ck1234567890" \
  -H "X-App-ID: d5a0ktoh1fg1tihi9c4g" \
  -H "X-Environment-ID: production" \
  -H "Authorization: Bearer your-token-here"
```

### Update Entry
```http
PUT /api/identity/cms/{typeSlug}/{entryId}
```

**Request Body:**
```json
{
  "data": {
    "title": "Updated Title",
    "content": "Updated content"
  }
}
```

### Delete Entry
```http
DELETE /api/identity/cms/{typeSlug}/{entryId}
```

### Publish Entry
```http
POST /api/identity/cms/{typeSlug}/{entryId}/publish
```

**Optional Request Body:**
```json
{
  "publishAt": "2024-01-20T12:00:00Z"
}
```

### Unpublish Entry
```http
POST /api/identity/cms/{typeSlug}/{entryId}/unpublish
```

### Archive Entry
```http
POST /api/identity/cms/{typeSlug}/{entryId}/archive
```

### Advanced Query
```http
POST /api/identity/cms/{typeSlug}/query
```

**Request Body:**
```json
{
  "filter": {
    "_meta.status": { "$eq": "published" },
    "title": { "$ilike": "%hello%" },
    "age": { "$gte": 18 }
  },
  "sort": ["-createdAt", "title"],
  "page": 1,
  "pageSize": 10,
  "populate": ["author", "category"]
}
```

### Bulk Operations

**Bulk Publish:**
```http
POST /api/identity/cms/{typeSlug}/bulk/publish
```

**Request Body:**
```json
{
  "ids": ["ck001", "ck002", "ck003"]
}
```

**Bulk Unpublish:**
```http
POST /api/identity/cms/{typeSlug}/bulk/unpublish
```

**Bulk Delete:**
```http
POST /api/identity/cms/{typeSlug}/bulk/delete
```

### Revisions

#### List Revisions
```
GET /api/cms/:typeSlug/entries/:entryId/revisions
```

#### Get Revision
```
GET /api/cms/:typeSlug/entries/:entryId/revisions/:version
```

#### Restore Revision
```
POST /api/cms/:typeSlug/entries/:entryId/revisions/:version/restore
```

#### Compare Revisions
```
GET /api/cms/:typeSlug/entries/:entryId/revisions/compare?from=1&to=2
```

### Query API

#### Complex Query (POST)
```
POST /api/cms/:typeSlug/query
```

Request Body:
```json
{
  "filters": {
    "$and": [
      {"status": {"$eq": "published"}},
      {"publishedAt": {"$gte": "2024-01-01"}}
    ]
  },
  "sort": ["-publishedAt", "title"],
  "select": ["title", "excerpt", "author"],
  "populate": ["author", "categories"],
  "page": 1,
  "pageSize": 10
}
```

## Field Types

| Type | Description | Options |
|------|-------------|---------|
| `text` | Short text | minLength, maxLength, pattern |
| `textarea` | Multiline text | minLength, maxLength |
| `richtext` | HTML formatted text | - |
| `markdown` | Markdown text | - |
| `number` | Numeric value | min, max, step |
| `integer` | Whole number | min, max |
| `float` | Decimal number | min, max, step |
| `decimal` | Precise decimal | min, max, precision |
| `boolean` | True/false | defaultValue |
| `date` | Date only | minDate, maxDate |
| `datetime` | Date and time | minDate, maxDate |
| `time` | Time only | - |
| `email` | Email address | - |
| `url` | Web URL | - |
| `phone` | Phone number | - |
| `slug` | URL-friendly string | - |
| `uuid` | Unique identifier | - |
| `color` | Color picker | - |
| `password` | Hashed password | - |
| `json` | Arbitrary JSON | schema |
| `select` | Single select | choices |
| `multiSelect` | Multi select | choices |
| `enumeration` | Predefined values | choices |
| `relation` | Reference to content | relatedType, relationType |
| `media` | File/image reference | allowedTypes, maxSize |

## Query Operators

### Comparison Operators
- `$eq` - Equal
- `$ne` - Not equal
- `$gt` - Greater than
- `$gte` - Greater than or equal
- `$lt` - Less than
- `$lte` - Less than or equal

### String Operators
- `$like` - Pattern match (case-sensitive)
- `$ilike` - Pattern match (case-insensitive)
- `$startsWith` - Starts with
- `$endsWith` - Ends with
- `$contains` - Contains substring

### Array Operators
- `$in` - Value in array
- `$nin` - Value not in array

### Null Operators
- `$null` - Is null
- `$notNull` - Is not null

### Logical Operators
- `$and` - All conditions must match
- `$or` - Any condition must match
- `$not` - Negate condition

## Architecture

```
plugins/cms/
├── config.go           # Plugin configuration
├── plugin.go           # Main plugin entry point
├── dashboard_extension.go  # Dashboard UI integration
├── core/               # Core types and utilities
│   ├── types.go        # DTOs and enums
│   ├── field_types.go  # Field type definitions
│   ├── errors.go       # CMS-specific errors
│   └── validator.go    # Validation helpers
├── schema/             # Database models
│   ├── content_type.go
│   ├── content_field.go
│   ├── content_entry.go
│   ├── content_revision.go
│   └── content_relation.go
├── repository/         # Data access layer
│   ├── content_type.go
│   ├── content_field.go
│   ├── content_entry.go
│   ├── revision.go
│   └── relation.go
├── service/            # Business logic
│   ├── content_type.go
│   ├── content_field.go
│   ├── content_entry.go
│   ├── revision.go
│   ├── relation.go
│   └── validation.go
├── handlers/           # HTTP handlers
│   ├── content_type.go
│   ├── content_entry.go
│   └── revision.go
├── query/              # Query language
│   ├── types.go        # Query AST
│   ├── operators.go    # Filter operators
│   ├── url_parser.go   # URL query parser
│   ├── json_parser.go  # JSON query parser
│   ├── builder.go      # SQL builder
│   ├── executor.go     # Query executor
│   ├── populate.go     # Relation population
│   ├── search.go       # Full-text search
│   └── aggregate.go    # Aggregations
└── pages/              # Dashboard pages
    ├── common.go       # Shared components
    ├── content_types.go
    ├── content_type_detail.go
    └── entries.go
```

## RBAC Permissions

The CMS plugin registers the following permissions:

### Content Types
- `read on cms_content_types` - View content types
- `create on cms_content_types` - Create content types
- `update on cms_content_types` - Update content types
- `delete on cms_content_types` - Delete content types

### Content Entries
- `read on cms_content_entries` - View entries
- `create on cms_content_entries` - Create entries
- `update on cms_content_entries` - Update entries
- `delete on cms_content_entries` - Delete entries
- `publish on cms_content_entries` - Publish/unpublish entries

### Revisions
- `read on cms_content_revisions` - View revisions
- `rollback on cms_content_revisions` - Restore revisions

## Database Schema

The plugin creates the following tables:

- `cms_content_types` - Content type definitions
- `cms_content_fields` - Field definitions
- `cms_content_entries` - Content entries with JSONB data
- `cms_content_revisions` - Entry version history
- `cms_content_relations` - Many-to-many relations
- `cms_content_type_relations` - Type relation definitions

## License

See the main AuthSome license.

