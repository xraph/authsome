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

### Content Types

#### List Content Types
```
GET /api/cms/types
```

Query Parameters:
- `search` - Search by name or slug
- `page` - Page number (default: 1)
- `pageSize` - Items per page (default: 20)

#### Create Content Type
```
POST /api/cms/types
```

Request Body:
```json
{
  "name": "Blog Posts",
  "slug": "blog-posts",
  "description": "Blog post content",
  "icon": "📝"
}
```

#### Get Content Type
```
GET /api/cms/types/:slug
```

#### Update Content Type
```
PUT /api/cms/types/:slug
```

#### Delete Content Type
```
DELETE /api/cms/types/:slug
```

### Content Fields

#### Add Field
```
POST /api/cms/types/:slug/fields
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

#### Update Field
```
PUT /api/cms/types/:slug/fields/:fieldSlug
```

#### Delete Field
```
DELETE /api/cms/types/:slug/fields/:fieldSlug
```

#### Reorder Fields
```
POST /api/cms/types/:slug/fields/reorder
```

### Content Entries

#### List Entries
```
GET /api/cms/:typeSlug/entries
```

Query Parameters:
- `search` - Full-text search
- `status` - Filter by status (draft, published, archived, scheduled)
- `page` - Page number
- `pageSize` - Items per page
- `sort` - Sort field (prefix with `-` for descending)
- `filter[field]` - Filter by field value

#### Create Entry
```
POST /api/cms/:typeSlug/entries
```

Request Body:
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

#### Get Entry
```
GET /api/cms/:typeSlug/entries/:entryId
```

#### Update Entry
```
PUT /api/cms/:typeSlug/entries/:entryId
```

#### Delete Entry
```
DELETE /api/cms/:typeSlug/entries/:entryId
```

#### Publish Entry
```
POST /api/cms/:typeSlug/entries/:entryId/publish
```

#### Unpublish Entry
```
POST /api/cms/:typeSlug/entries/:entryId/unpublish
```

#### Archive Entry
```
POST /api/cms/:typeSlug/entries/:entryId/archive
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

