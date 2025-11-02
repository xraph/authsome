# AuthSome Cloud API Reference

**Complete API documentation for control plane and proxy services**

## Base URLs

```
Control Plane API: https://api.authsome.cloud/control/v1
Customer API Proxy: https://api.authsome.cloud/v1/:appId
Management Dashboard: https://dashboard.authsome.cloud
```

## Authentication

### Dashboard API (Control Plane)

Uses JWT tokens issued after email/password or OAuth login.

```bash
# Login
POST /control/v1/auth/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "secure-password"
}

# Response
{
  "token": "eyJhbGc...",
  "user": {
    "id": "usr_123",
    "email": "user@example.com",
    "name": "John Doe"
  }
}

# Use token in subsequent requests
curl -H "Authorization: Bearer eyJhbGc..." \
  https://api.authsome.cloud/control/v1/workspaces
```

### Customer API (Proxy)

Uses application-specific API keys.

```bash
# Public key (frontend - read-only operations)
curl -H "Authorization: Bearer pk_live_abc123..." \
  https://api.authsome.cloud/v1/app_abc123/session

# Secret key (backend - full access)
curl -H "Authorization: Bearer sk_live_abc123..." \
  https://api.authsome.cloud/v1/app_abc123/users
```

## Control Plane API

### Workspaces

#### Create Workspace

```http
POST /control/v1/workspaces
Authorization: Bearer {jwt_token}
Content-Type: application/json

{
  "name": "Acme Corporation",
  "slug": "acme-corp",
  "plan": "pro"
}
```

**Response: 201 Created**
```json
{
  "id": "ws_abc123",
  "name": "Acme Corporation",
  "slug": "acme-corp",
  "plan": "pro",
  "status": "active",
  "ownerId": "usr_123",
  "createdAt": "2025-11-01T10:00:00Z",
  "updatedAt": "2025-11-01T10:00:00Z"
}
```

#### List Workspaces

```http
GET /control/v1/workspaces
Authorization: Bearer {jwt_token}
```

**Response: 200 OK**
```json
{
  "data": [
    {
      "id": "ws_abc123",
      "name": "Acme Corporation",
      "slug": "acme-corp",
      "plan": "pro",
      "status": "active",
      "role": "owner",
      "createdAt": "2025-11-01T10:00:00Z"
    }
  ],
  "total": 1,
  "page": 1,
  "pageSize": 20
}
```

#### Get Workspace

```http
GET /control/v1/workspaces/:workspaceId
Authorization: Bearer {jwt_token}
```

**Response: 200 OK**
```json
{
  "id": "ws_abc123",
  "name": "Acme Corporation",
  "slug": "acme-corp",
  "plan": "pro",
  "status": "active",
  "ownerId": "usr_123",
  "metadata": {
    "industry": "technology",
    "company_size": "50-100"
  },
  "createdAt": "2025-11-01T10:00:00Z",
  "updatedAt": "2025-11-01T10:00:00Z"
}
```

#### Update Workspace

```http
PATCH /control/v1/workspaces/:workspaceId
Authorization: Bearer {jwt_token}
Content-Type: application/json

{
  "name": "Acme Corp Inc.",
  "metadata": {
    "company_size": "100-250"
  }
}
```

**Response: 200 OK**
```json
{
  "id": "ws_abc123",
  "name": "Acme Corp Inc.",
  "slug": "acme-corp",
  "plan": "pro",
  "status": "active",
  "metadata": {
    "industry": "technology",
    "company_size": "100-250"
  },
  "updatedAt": "2025-11-01T11:00:00Z"
}
```

#### Delete Workspace

```http
DELETE /control/v1/workspaces/:workspaceId
Authorization: Bearer {jwt_token}
```

**Response: 204 No Content**

Note: Soft delete with 30-day grace period. All applications will be suspended.

### Applications

#### Create Application

```http
POST /control/v1/workspaces/:workspaceId/applications
Authorization: Bearer {jwt_token}
Content-Type: application/json

{
  "name": "Production",
  "slug": "production",
  "environment": "production",
  "region": "us-east-1",
  "config": {
    "mode": "saas",
    "session": {
      "timeout": "7d"
    }
  }
}
```

**Response: 202 Accepted**
```json
{
  "id": "app_abc123",
  "workspaceId": "ws_abc123",
  "name": "Production",
  "slug": "production",
  "environment": "production",
  "status": "provisioning",
  "region": "us-east-1",
  "publicKey": "pk_live_abc123def456",
  "secretKey": "sk_live_abc123def456ghi789",
  "apiUrl": "https://api.authsome.cloud/v1/app_abc123",
  "createdAt": "2025-11-01T10:00:00Z"
}
```

Note: Provisioning is asynchronous. Poll `/applications/:id` to check status.

#### List Applications

```http
GET /control/v1/workspaces/:workspaceId/applications
Authorization: Bearer {jwt_token}
```

**Query Parameters:**
- `environment` (optional): Filter by environment (production, staging, development)
- `status` (optional): Filter by status (provisioning, active, suspended, deleted)
- `page` (optional): Page number (default: 1)
- `pageSize` (optional): Items per page (default: 20, max: 100)

**Response: 200 OK**
```json
{
  "data": [
    {
      "id": "app_abc123",
      "workspaceId": "ws_abc123",
      "name": "Production",
      "slug": "production",
      "environment": "production",
      "status": "active",
      "region": "us-east-1",
      "publicKey": "pk_live_abc123def456",
      "apiUrl": "https://api.authsome.cloud/v1/app_abc123",
      "createdAt": "2025-11-01T10:00:00Z",
      "updatedAt": "2025-11-01T10:05:00Z"
    }
  ],
  "total": 3,
  "page": 1,
  "pageSize": 20
}
```

#### Get Application

```http
GET /control/v1/applications/:applicationId
Authorization: Bearer {jwt_token}
```

**Response: 200 OK**
```json
{
  "id": "app_abc123",
  "workspaceId": "ws_abc123",
  "name": "Production",
  "slug": "production",
  "environment": "production",
  "status": "active",
  "region": "us-east-1",
  "publicKey": "pk_live_abc123def456",
  "apiUrl": "https://api.authsome.cloud/v1/app_abc123",
  "dashboardUrl": "https://dashboard.authsome.cloud/workspace/ws_abc123/app/app_abc123",
  "config": {
    "mode": "saas",
    "session": {
      "timeout": "7d"
    }
  },
  "resources": {
    "cpuLimit": "1000m",
    "memoryLimit": "2Gi",
    "storageLimit": "10Gi",
    "replicas": 2
  },
  "metrics": {
    "mau": 1250,
    "apiRequests": 145000,
    "avgLatency": 45,
    "uptime": 99.98
  },
  "createdAt": "2025-11-01T10:00:00Z",
  "updatedAt": "2025-11-01T10:05:00Z"
}
```

#### Update Application

```http
PATCH /control/v1/applications/:applicationId
Authorization: Bearer {jwt_token}
Content-Type: application/json

{
  "name": "Production (Updated)",
  "config": {
    "session": {
      "timeout": "14d"
    }
  }
}
```

**Response: 200 OK**
```json
{
  "id": "app_abc123",
  "name": "Production (Updated)",
  "config": {
    "mode": "saas",
    "session": {
      "timeout": "14d"
    }
  },
  "updatedAt": "2025-11-01T11:00:00Z"
}
```

Note: Config changes trigger rolling deployment. Application remains available.

#### Restart Application

```http
POST /control/v1/applications/:applicationId/restart
Authorization: Bearer {jwt_token}
```

**Response: 202 Accepted**
```json
{
  "id": "app_abc123",
  "status": "restarting",
  "message": "Application restart initiated"
}
```

#### Delete Application

```http
DELETE /control/v1/applications/:applicationId
Authorization: Bearer {jwt_token}
```

**Response: 204 No Content**

Note: Soft delete with 7-day grace period. Can be recovered within this period.

#### Get Application Logs

```http
GET /control/v1/applications/:applicationId/logs
Authorization: Bearer {jwt_token}
```

**Query Parameters:**
- `since` (optional): RFC3339 timestamp
- `tail` (optional): Number of lines (default: 100, max: 1000)
- `level` (optional): Filter by log level (debug, info, warn, error)

**Response: 200 OK**
```json
{
  "logs": [
    {
      "timestamp": "2025-11-01T10:30:00Z",
      "level": "info",
      "message": "User created successfully",
      "fields": {
        "userId": "usr_789",
        "email": "user@example.com"
      }
    }
  ],
  "total": 100,
  "hasMore": true
}
```

#### Get Application Metrics

```http
GET /control/v1/applications/:applicationId/metrics
Authorization: Bearer {jwt_token}
```

**Query Parameters:**
- `period` (optional): Time period (1h, 24h, 7d, 30d) (default: 24h)
- `interval` (optional): Data point interval (1m, 5m, 1h) (default: 5m)

**Response: 200 OK**
```json
{
  "period": "24h",
  "interval": "5m",
  "metrics": {
    "requests": {
      "dataPoints": [
        {
          "timestamp": "2025-11-01T10:00:00Z",
          "value": 1234
        }
      ],
      "total": 145000,
      "avgPerMinute": 100.7
    },
    "latency": {
      "dataPoints": [
        {
          "timestamp": "2025-11-01T10:00:00Z",
          "p50": 25,
          "p95": 78,
          "p99": 142
        }
      ],
      "avgP95": 75
    },
    "errors": {
      "total": 23,
      "rate": 0.016
    },
    "activeUsers": {
      "current": 145,
      "mau": 1250
    }
  }
}
```

### API Keys

#### List API Keys

```http
GET /control/v1/applications/:applicationId/keys
Authorization: Bearer {jwt_token}
```

**Response: 200 OK**
```json
{
  "data": [
    {
      "id": "key_123",
      "name": "Production Server",
      "type": "secret",
      "key": "sk_live_abc123***",
      "scopes": ["users:read", "users:write"],
      "lastUsedAt": "2025-11-01T10:30:00Z",
      "createdAt": "2025-11-01T10:00:00Z"
    }
  ]
}
```

#### Create API Key

```http
POST /control/v1/applications/:applicationId/keys
Authorization: Bearer {jwt_token}
Content-Type: application/json

{
  "name": "Backend Service",
  "type": "secret",
  "scopes": ["users:read", "users:write", "sessions:manage"],
  "expiresAt": "2026-11-01T00:00:00Z"
}
```

**Response: 201 Created**
```json
{
  "id": "key_456",
  "name": "Backend Service",
  "type": "secret",
  "key": "sk_live_abc123def456ghi789jkl012",
  "scopes": ["users:read", "users:write", "sessions:manage"],
  "expiresAt": "2026-11-01T00:00:00Z",
  "createdAt": "2025-11-01T12:00:00Z"
}
```

**Warning:** The full secret key is only shown once. Store it securely.

#### Revoke API Key

```http
DELETE /control/v1/applications/:applicationId/keys/:keyId
Authorization: Bearer {jwt_token}
```

**Response: 204 No Content**

### Team Members

#### Invite Team Member

```http
POST /control/v1/workspaces/:workspaceId/members
Authorization: Bearer {jwt_token}
Content-Type: application/json

{
  "email": "developer@acme.com",
  "role": "developer"
}
```

**Roles:**
- `owner`: Full access, can delete workspace
- `admin`: Manage applications, members, billing
- `developer`: Manage applications, view billing
- `billing`: View billing only

**Response: 201 Created**
```json
{
  "id": "mem_123",
  "workspaceId": "ws_abc123",
  "email": "developer@acme.com",
  "role": "developer",
  "status": "invited",
  "invitedBy": "usr_123",
  "invitedAt": "2025-11-01T12:00:00Z"
}
```

Note: Invitation email sent automatically.

#### List Team Members

```http
GET /control/v1/workspaces/:workspaceId/members
Authorization: Bearer {jwt_token}
```

**Response: 200 OK**
```json
{
  "data": [
    {
      "id": "mem_123",
      "userId": "usr_123",
      "email": "owner@acme.com",
      "name": "John Doe",
      "role": "owner",
      "status": "active",
      "joinedAt": "2025-11-01T10:00:00Z"
    },
    {
      "id": "mem_456",
      "email": "developer@acme.com",
      "role": "developer",
      "status": "invited",
      "invitedAt": "2025-11-01T12:00:00Z"
    }
  ],
  "total": 2
}
```

#### Update Member Role

```http
PATCH /control/v1/workspaces/:workspaceId/members/:memberId
Authorization: Bearer {jwt_token}
Content-Type: application/json

{
  "role": "admin"
}
```

**Response: 200 OK**
```json
{
  "id": "mem_456",
  "role": "admin",
  "updatedAt": "2025-11-01T13:00:00Z"
}
```

#### Remove Team Member

```http
DELETE /control/v1/workspaces/:workspaceId/members/:memberId
Authorization: Bearer {jwt_token}
```

**Response: 204 No Content**

### Billing

#### Get Usage

```http
GET /control/v1/workspaces/:workspaceId/usage
Authorization: Bearer {jwt_token}
```

**Query Parameters:**
- `period` (optional): YYYY-MM format (default: current month)

**Response: 200 OK**
```json
{
  "period": "2025-11",
  "workspaceId": "ws_abc123",
  "plan": "pro",
  "usage": {
    "mau": {
      "current": 1250,
      "included": 10000,
      "overage": 0,
      "cost": 0
    },
    "applications": {
      "count": 3,
      "limit": null
    },
    "storage": {
      "bytes": 5368709120,
      "gb": 5.0,
      "included": 10,
      "overage": 0,
      "cost": 0
    },
    "apiRequests": {
      "count": 1450000,
      "included": null
    }
  },
  "estimatedCost": 25.00,
  "billingDate": "2025-12-01T00:00:00Z"
}
```

#### List Invoices

```http
GET /control/v1/workspaces/:workspaceId/invoices
Authorization: Bearer {jwt_token}
```

**Response: 200 OK**
```json
{
  "data": [
    {
      "id": "inv_123",
      "period": "2025-10",
      "amount": 47.50,
      "status": "paid",
      "paidAt": "2025-11-01T00:00:00Z",
      "downloadUrl": "https://api.authsome.cloud/control/v1/invoices/inv_123/download"
    }
  ],
  "total": 12
}
```

#### Update Payment Method

```http
POST /control/v1/workspaces/:workspaceId/payment-method
Authorization: Bearer {jwt_token}
Content-Type: application/json

{
  "stripePaymentMethodId": "pm_xxx"
}
```

**Response: 200 OK**
```json
{
  "paymentMethod": {
    "type": "card",
    "card": {
      "brand": "visa",
      "last4": "4242",
      "expMonth": 12,
      "expYear": 2026
    }
  }
}
```

## Customer API (Proxy)

All AuthSome core API endpoints are available through the proxy with the application ID in the path.

### Request Format

```bash
# Base URL structure
https://api.authsome.cloud/v1/{appId}/{authsome-endpoint}

# Examples
POST https://api.authsome.cloud/v1/app_abc123/auth/signup
GET  https://api.authsome.cloud/v1/app_abc123/users/usr_789
POST https://api.authsome.cloud/v1/app_abc123/organizations
```

### Authentication Headers

```bash
# Public key (limited operations)
Authorization: Bearer pk_live_abc123def456

# Secret key (full access)
Authorization: Bearer sk_live_abc123def456ghi789
```

### Example: User Signup

```http
POST /v1/app_abc123/auth/signup
Authorization: Bearer pk_live_abc123def456
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "secure-password-123",
  "name": "Jane Doe"
}
```

**Response: 201 Created**
```json
{
  "user": {
    "id": "usr_789",
    "email": "user@example.com",
    "name": "Jane Doe",
    "emailVerified": false,
    "createdAt": "2025-11-01T12:00:00Z"
  },
  "session": {
    "id": "ses_456",
    "token": "eyJhbGciOiJIUzI1NiIs...",
    "expiresAt": "2025-11-08T12:00:00Z"
  }
}
```

### Response Headers

All proxied responses include these headers:

```
X-AuthSome-App-ID: app_abc123
X-AuthSome-Request-ID: req_xyz789
X-AuthSome-Region: us-east-1
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 997
X-RateLimit-Reset: 1698854400
```

## Rate Limits

### Control Plane API

```
Free Plan:
- 100 requests per minute
- 10,000 requests per day

Pro Plan:
- 1,000 requests per minute
- 100,000 requests per day

Enterprise:
- Custom limits
```

### Customer API (Proxy)

```
Free Plan:
- 10 requests per second per API key
- 100,000 requests per month

Pro Plan:
- 100 requests per second per API key
- Unlimited requests

Enterprise:
- Custom limits
```

**Rate Limit Headers:**
```
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 997
X-RateLimit-Reset: 1698854400
Retry-After: 60
```

## Error Responses

### Standard Error Format

```json
{
  "error": {
    "code": "invalid_request",
    "message": "Email is required",
    "details": {
      "field": "email",
      "reason": "missing_field"
    },
    "requestId": "req_xyz789"
  }
}
```

### Error Codes

**400 Bad Request**
```json
{
  "error": {
    "code": "invalid_request",
    "message": "Invalid JSON in request body"
  }
}
```

**401 Unauthorized**
```json
{
  "error": {
    "code": "unauthorized",
    "message": "Invalid or expired API key"
  }
}
```

**403 Forbidden**
```json
{
  "error": {
    "code": "forbidden",
    "message": "Insufficient permissions for this operation"
  }
}
```

**404 Not Found**
```json
{
  "error": {
    "code": "not_found",
    "message": "Application not found"
  }
}
```

**429 Too Many Requests**
```json
{
  "error": {
    "code": "rate_limit_exceeded",
    "message": "Rate limit exceeded. Retry after 60 seconds",
    "retryAfter": 60
  }
}
```

**500 Internal Server Error**
```json
{
  "error": {
    "code": "internal_error",
    "message": "An unexpected error occurred",
    "requestId": "req_xyz789"
  }
}
```

**503 Service Unavailable**
```json
{
  "error": {
    "code": "service_unavailable",
    "message": "Application is currently unavailable",
    "reason": "maintenance"
  }
}
```

## Webhooks

### Configuring Webhooks

```http
POST /control/v1/applications/:applicationId/webhooks
Authorization: Bearer {jwt_token}
Content-Type: application/json

{
  "url": "https://your-app.com/webhooks/authsome",
  "events": [
    "application.provisioned",
    "application.failed",
    "application.deleted",
    "usage.threshold"
  ],
  "secret": "whsec_your-secret"
}
```

### Webhook Events

#### application.provisioned
```json
{
  "event": "application.provisioned",
  "timestamp": "2025-11-01T10:05:00Z",
  "data": {
    "applicationId": "app_abc123",
    "workspaceId": "ws_abc123",
    "status": "active",
    "publicKey": "pk_live_abc123def456",
    "apiUrl": "https://api.authsome.cloud/v1/app_abc123"
  }
}
```

#### application.failed
```json
{
  "event": "application.failed",
  "timestamp": "2025-11-01T10:05:00Z",
  "data": {
    "applicationId": "app_abc123",
    "error": "Database provisioning failed",
    "retryCount": 3
  }
}
```

#### usage.threshold
```json
{
  "event": "usage.threshold",
  "timestamp": "2025-11-15T14:30:00Z",
  "data": {
    "workspaceId": "ws_abc123",
    "threshold": "mau_80_percent",
    "current": 8500,
    "limit": 10000
  }
}
```

## SDK Examples

### JavaScript/TypeScript

```typescript
import { AuthSomeCloud } from '@authsome/cloud'

const client = new AuthSomeCloud({
  apiKey: process.env.AUTHSOME_CONTROL_TOKEN
})

// Create workspace
const workspace = await client.workspaces.create({
  name: 'My Company',
  plan: 'pro'
})

// Create application
const app = await client.applications.create(workspace.id, {
  name: 'Production',
  environment: 'production',
  region: 'us-east-1'
})

// Use application API
const authsome = new AuthSome({
  appId: app.id,
  apiKey: app.secretKey,
  apiUrl: 'https://api.authsome.cloud'
})

const user = await authsome.users.create({
  email: 'user@example.com',
  password: 'secure-password'
})
```

### Go

```go
package main

import (
    "github.com/xraph/authsome-cloud-go"
)

func main() {
    client := authsomecloud.New(&authsomecloud.Config{
        APIKey: os.Getenv("AUTHSOME_CONTROL_TOKEN"),
    })
    
    // Create workspace
    workspace, err := client.Workspaces.Create(ctx, &authsomecloud.CreateWorkspaceRequest{
        Name: "My Company",
        Plan: "pro",
    })
    
    // Create application
    app, err := client.Applications.Create(ctx, workspace.ID, &authsomecloud.CreateApplicationRequest{
        Name:        "Production",
        Environment: "production",
        Region:      "us-east-1",
    })
    
    // Use application API
    auth := authsome.New(&authsome.Config{
        AppID:  app.ID,
        APIKey: app.SecretKey,
        APIURL: "https://api.authsome.cloud",
    })
    
    user, err := auth.Users.Create(ctx, &authsome.CreateUserRequest{
        Email:    "user@example.com",
        Password: "secure-password",
    })
}
```

---

**For complete AuthSome Core API documentation, see: https://docs.authsome.dev/api**

