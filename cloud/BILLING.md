# AuthSome Cloud Billing & Pricing

**Usage tracking, metering, and billing implementation**

## Pricing Model

### Tiered Plans

```yaml
Free Plan:
  mau: 10,000 (included)
  applications: 3
  storage: 5 GB
  support: Community
  data_retention: 7 days
  price: $0/month

Pro Plan:
  mau: 10,000 (included, then $0.02/MAU)
  applications: Unlimited
  storage: 20 GB (included, then $0.10/GB)
  support: Email
  data_retention: 30 days
  price: $25/month + usage

Enterprise Plan:
  mau: Custom pricing
  applications: Unlimited
  storage: Custom
  support: Priority + Slack
  data_retention: Custom
  sla: 99.99%
  dedicated_instances: Available
  price: Custom contract
```

### Usage-Based Pricing

```
Monthly Active Users (MAU):
- First 10,000 MAU: Included in Pro plan
- 10,001 - 100,000: $0.02 per MAU
- 100,001 - 1,000,000: $0.015 per MAU
- 1,000,000+: $0.01 per MAU (volume discount)

Storage:
- First 20 GB: Included in Pro plan
- Additional: $0.10 per GB/month

API Requests:
- Unlimited (fair use policy)
- Extreme usage (>10M req/day): Custom pricing

Bandwidth:
- First 100 GB: Included
- Additional: $0.05 per GB
```

## Usage Tracking

### MAU Calculation

```go
// internal/billing/mau.go
package billing

import (
    "context"
    "time"
)

// MAUCalculator tracks monthly active users
type MAUCalculator struct {
    repo repository.UsageRepository
}

// Calculate MAU for an application in a given month
func (c *MAUCalculator) Calculate(ctx context.Context, appID string, period string) (int, error) {
    // Period format: "2025-11"
    start, end := parsePeriod(period)
    
    // Count unique users with activity in period
    // Activity = any authentication event (login, signup, token refresh)
    mau, err := c.repo.CountUniqueActiveUsers(ctx, appID, start, end)
    if err != nil {
        return 0, fmt.Errorf("failed to calculate MAU: %w", err)
    }
    
    return mau, nil
}

// Track user activity for MAU calculation
func (c *MAUCalculator) TrackActivity(ctx context.Context, appID, userID string) error {
    // Use Redis HyperLogLog for efficient unique counting
    key := fmt.Sprintf("mau:%s:%s", appID, time.Now().Format("2006-01"))
    
    err := c.redis.PFAdd(ctx, key, userID).Err()
    if err != nil {
        return fmt.Errorf("failed to track activity: %w", err)
    }
    
    // Expire key after 60 days
    c.redis.Expire(ctx, key, 60*24*time.Hour)
    
    return nil
}

// Get current month MAU (real-time)
func (c *MAUCalculator) GetCurrentMAU(ctx context.Context, appID string) (int, error) {
    key := fmt.Sprintf("mau:%s:%s", appID, time.Now().Format("2006-01"))
    
    count, err := c.redis.PFCount(ctx, key).Result()
    if err != nil {
        return 0, fmt.Errorf("failed to get MAU: %w", err)
    }
    
    return int(count), nil
}
```

### Request Metering

```go
// internal/billing/requests.go
package billing

import (
    "context"
    "time"
    "github.com/nats-io/nats.go"
)

// RequestMeter tracks API requests for billing
type RequestMeter struct {
    nats *nats.Conn
}

// Track API request (called from proxy)
func (m *RequestMeter) Track(ctx context.Context, req *RequestMetadata) error {
    // Publish to NATS for async processing
    data, _ := json.Marshal(req)
    
    err := m.nats.Publish("billing.request", data)
    if err != nil {
        // Log error but don't fail request
        log.Error("failed to track request", "error", err)
        return nil
    }
    
    return nil
}

type RequestMetadata struct {
    ApplicationID string    `json:"applicationId"`
    WorkspaceID   string    `json:"workspaceId"`
    Method        string    `json:"method"`
    Path          string    `json:"path"`
    StatusCode    int       `json:"statusCode"`
    Duration      int64     `json:"duration"` // milliseconds
    BytesSent     int64     `json:"bytesSent"`
    Timestamp     time.Time `json:"timestamp"`
}

// Aggregator worker processes request events
type RequestAggregator struct {
    nats *nats.Conn
    repo repository.UsageRepository
}

func (a *RequestAggregator) Start(ctx context.Context) error {
    // Subscribe to request events
    _, err := a.nats.QueueSubscribe("billing.request", "aggregators", func(msg *nats.Msg) {
        var req RequestMetadata
        json.Unmarshal(msg.Data, &req)
        
        // Aggregate in PostgreSQL (batched)
        a.aggregate(ctx, &req)
    })
    
    return err
}

func (a *RequestAggregator) aggregate(ctx context.Context, req *RequestMetadata) error {
    // Use PostgreSQL INSERT ... ON CONFLICT for efficient updates
    query := `
        INSERT INTO usage_requests (
            application_id,
            workspace_id,
            period,
            count,
            bytes_sent,
            total_duration
        ) VALUES ($1, $2, $3, 1, $4, $5)
        ON CONFLICT (application_id, period)
        DO UPDATE SET
            count = usage_requests.count + 1,
            bytes_sent = usage_requests.bytes_sent + EXCLUDED.bytes_sent,
            total_duration = usage_requests.total_duration + EXCLUDED.total_duration
    `
    
    period := req.Timestamp.Format("2006-01")
    
    _, err := a.repo.Exec(ctx, query,
        req.ApplicationID,
        req.WorkspaceID,
        period,
        req.BytesSent,
        req.Duration,
    )
    
    return err
}
```

### Storage Tracking

```go
// internal/billing/storage.go
package billing

import (
    "context"
    "database/sql"
)

// StorageCalculator tracks database storage usage
type StorageCalculator struct {
    adminDB *sql.DB
}

// Calculate storage for an application
func (c *StorageCalculator) Calculate(ctx context.Context, appID string) (int64, error) {
    // Connect to application's database
    dbName := fmt.Sprintf("authsome_app_%s", appID)
    
    query := `
        SELECT pg_database_size($1)
    `
    
    var sizeBytes int64
    err := c.adminDB.QueryRowContext(ctx, query, dbName).Scan(&sizeBytes)
    if err != nil {
        return 0, fmt.Errorf("failed to calculate storage: %w", err)
    }
    
    return sizeBytes, nil
}

// Schedule periodic storage calculation
func (c *StorageCalculator) ScheduleCalculation(ctx context.Context) {
    ticker := time.NewTicker(1 * time.Hour)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            c.calculateAllApps(ctx)
        case <-ctx.Done():
            return
        }
    }
}

func (c *StorageCalculator) calculateAllApps(ctx context.Context) {
    apps, _ := c.repo.ListActiveApplications(ctx)
    
    for _, app := range apps {
        size, err := c.Calculate(ctx, app.ID)
        if err != nil {
            log.Error("failed to calculate storage", "appId", app.ID, "error", err)
            continue
        }
        
        // Store in usage_storage table
        c.repo.RecordStorage(ctx, app.ID, size)
    }
}
```

## Invoice Generation

### Monthly Invoice Process

```go
// internal/billing/invoice.go
package billing

import (
    "context"
    "time"
)

// InvoiceGenerator creates monthly invoices
type InvoiceGenerator struct {
    repo         repository.UsageRepository
    stripe       *stripe.Client
    emailService EmailService
}

// Generate invoices for all workspaces (run on 1st of month)
func (g *InvoiceGenerator) GenerateMonthly(ctx context.Context) error {
    // Get previous month
    now := time.Now()
    period := now.AddDate(0, -1, 0).Format("2006-01")
    
    workspaces, err := g.repo.ListBillableWorkspaces(ctx)
    if err != nil {
        return err
    }
    
    for _, ws := range workspaces {
        err := g.generateWorkspaceInvoice(ctx, ws, period)
        if err != nil {
            log.Error("failed to generate invoice", "workspace", ws.ID, "error", err)
            // Continue with other workspaces
        }
    }
    
    return nil
}

func (g *InvoiceGenerator) generateWorkspaceInvoice(ctx context.Context, ws *Workspace, period string) error {
    // Skip free plan
    if ws.Plan == "free" {
        return nil
    }
    
    // Calculate usage
    usage, err := g.calculateUsage(ctx, ws.ID, period)
    if err != nil {
        return err
    }
    
    // Calculate costs
    invoice := &Invoice{
        WorkspaceID: ws.ID,
        Period:      period,
        LineItems:   []LineItem{},
    }
    
    // Base subscription
    if ws.Plan == "pro" {
        invoice.LineItems = append(invoice.LineItems, LineItem{
            Description: "Pro Plan Subscription",
            Quantity:    1,
            UnitPrice:   2500, // $25.00 in cents
            Amount:      2500,
        })
    }
    
    // MAU overage
    if usage.MAU > 10000 {
        overage := usage.MAU - 10000
        pricePerMAU := g.calculateMAUPrice(usage.MAU)
        
        invoice.LineItems = append(invoice.LineItems, LineItem{
            Description: fmt.Sprintf("Additional MAU (%d users)", overage),
            Quantity:    overage,
            UnitPrice:   pricePerMAU,
            Amount:      overage * pricePerMAU,
        })
    }
    
    // Storage overage
    storageGB := usage.StorageBytes / (1024 * 1024 * 1024)
    if storageGB > 20 {
        overage := storageGB - 20
        
        invoice.LineItems = append(invoice.LineItems, LineItem{
            Description: fmt.Sprintf("Additional Storage (%d GB)", overage),
            Quantity:    int(overage),
            UnitPrice:   10, // $0.10 in cents
            Amount:      int(overage) * 10,
        })
    }
    
    // Calculate total
    for _, item := range invoice.LineItems {
        invoice.Total += item.Amount
    }
    
    // Create Stripe invoice
    stripeInvoice, err := g.createStripeInvoice(ctx, ws, invoice)
    if err != nil {
        return err
    }
    
    // Store in database
    invoice.StripeInvoiceID = stripeInvoice.ID
    invoice.Status = "pending"
    invoice.DueDate = time.Now().AddDate(0, 0, 7) // 7 days net
    
    err = g.repo.SaveInvoice(ctx, invoice)
    if err != nil {
        return err
    }
    
    // Send email notification
    g.emailService.SendInvoice(ctx, ws.OwnerEmail, invoice)
    
    return nil
}

func (g *InvoiceGenerator) calculateMAUPrice(totalMAU int) int {
    // Tiered pricing
    if totalMAU <= 100000 {
        return 2 // $0.02
    } else if totalMAU <= 1000000 {
        return 1.5 // $0.015 (rounded)
    } else {
        return 1 // $0.01
    }
}

type Invoice struct {
    ID              string     `json:"id"`
    WorkspaceID     string     `json:"workspaceId"`
    Period          string     `json:"period"`
    Status          string     `json:"status"` // pending, paid, failed
    LineItems       []LineItem `json:"lineItems"`
    Subtotal        int        `json:"subtotal"` // cents
    Tax             int        `json:"tax"`      // cents
    Total           int        `json:"total"`    // cents
    DueDate         time.Time  `json:"dueDate"`
    PaidAt          *time.Time `json:"paidAt"`
    StripeInvoiceID string     `json:"stripeInvoiceId"`
    CreatedAt       time.Time  `json:"createdAt"`
}

type LineItem struct {
    Description string `json:"description"`
    Quantity    int    `json:"quantity"`
    UnitPrice   int    `json:"unitPrice"` // cents
    Amount      int    `json:"amount"`    // cents
}
```

## Stripe Integration

### Setup

```go
// internal/billing/stripe.go
package billing

import (
    "github.com/stripe/stripe-go/v76"
    "github.com/stripe/stripe-go/v76/customer"
    "github.com/stripe/stripe-go/v76/invoice"
    "github.com/stripe/stripe-go/v76/paymentmethod"
)

type StripeService struct {
    apiKey string
}

func NewStripeService(apiKey string) *StripeService {
    stripe.Key = apiKey
    return &StripeService{apiKey: apiKey}
}

// Create Stripe customer when workspace created
func (s *StripeService) CreateCustomer(ctx context.Context, ws *Workspace) (*stripe.Customer, error) {
    params := &stripe.CustomerParams{
        Email: stripe.String(ws.OwnerEmail),
        Name:  stripe.String(ws.Name),
        Metadata: map[string]string{
            "workspace_id": ws.ID,
        },
    }
    
    return customer.New(params)
}

// Create invoice in Stripe
func (s *StripeService) CreateInvoice(ctx context.Context, ws *Workspace, inv *Invoice) (*stripe.Invoice, error) {
    // Get or create Stripe customer
    stripeCustomerID, err := s.getStripeCustomerID(ctx, ws)
    if err != nil {
        return nil, err
    }
    
    // Create invoice items
    for _, item := range inv.LineItems {
        _, err := stripe.InvoiceItem.New(&stripe.InvoiceItemParams{
            Customer:    stripe.String(stripeCustomerID),
            Amount:      stripe.Int64(int64(item.Amount)),
            Currency:    stripe.String("usd"),
            Description: stripe.String(item.Description),
        })
        if err != nil {
            return nil, err
        }
    }
    
    // Create invoice
    params := &stripe.InvoiceParams{
        Customer: stripe.String(stripeCustomerID),
        Metadata: map[string]string{
            "workspace_id": ws.ID,
            "period":       inv.Period,
        },
        DaysUntilDue: stripe.Int64(7),
    }
    
    return invoice.New(params)
}

// Handle webhook events
func (s *StripeService) HandleWebhook(ctx context.Context, payload []byte, signature string) error {
    event, err := webhook.ConstructEvent(payload, signature, s.webhookSecret)
    if err != nil {
        return err
    }
    
    switch event.Type {
    case "invoice.paid":
        return s.handleInvoicePaid(ctx, event)
    case "invoice.payment_failed":
        return s.handleInvoiceFailed(ctx, event)
    case "customer.subscription.deleted":
        return s.handleSubscriptionDeleted(ctx, event)
    }
    
    return nil
}

func (s *StripeService) handleInvoicePaid(ctx context.Context, event stripe.Event) error {
    var invoice stripe.Invoice
    json.Unmarshal(event.Data.Raw, &invoice)
    
    workspaceID := invoice.Metadata["workspace_id"]
    
    // Update invoice status
    return s.repo.UpdateInvoiceStatus(ctx, workspaceID, invoice.ID, "paid")
}
```

### Webhook Endpoint

```go
// control/billing/handler.go
func (h *BillingHandler) StripeWebhook(c *forge.Context) error {
    payload, err := c.Body()
    if err != nil {
        return c.JSON(400, ErrorResponse{Message: "Invalid payload"})
    }
    
    signature := c.Request().Header.Get("Stripe-Signature")
    
    err = h.stripeService.HandleWebhook(c.Context(), payload, signature)
    if err != nil {
        return c.JSON(400, ErrorResponse{Message: err.Error()})
    }
    
    return c.JSON(200, map[string]string{"status": "success"})
}
```

## Usage Alerts

### Threshold Monitoring

```go
// internal/billing/alerts.go
package billing

// Check usage thresholds and send alerts
type AlertService struct {
    repo         repository.UsageRepository
    emailService EmailService
}

func (s *AlertService) CheckThresholds(ctx context.Context) error {
    workspaces, _ := s.repo.ListActiveWorkspaces(ctx)
    
    for _, ws := range workspaces {
        usage, _ := s.getCurrentUsage(ctx, ws.ID)
        
        // Check MAU threshold
        if ws.Plan == "pro" {
            included := 10000
            current := usage.MAU
            
            // Alert at 80%, 90%, 100%
            thresholds := []int{80, 90, 100}
            for _, threshold := range thresholds {
                limit := (included * threshold) / 100
                
                if current >= limit && !s.hasAlerted(ws.ID, "mau", threshold) {
                    s.sendUsageAlert(ctx, ws, "mau", current, included, threshold)
                    s.markAlerted(ws.ID, "mau", threshold)
                }
            }
        }
        
        // Check storage threshold
        // Similar logic for storage
    }
    
    return nil
}

func (s *AlertService) sendUsageAlert(ctx context.Context, ws *Workspace, metric string, current, limit, threshold int) {
    subject := fmt.Sprintf("Usage Alert: %s at %d%%", metric, threshold)
    
    body := fmt.Sprintf(`
        Your %s usage is at %d%% of your plan limit.
        
        Current: %d
        Included: %d
        
        Overage charges will apply beyond the included amount.
        
        View details: https://dashboard.authsome.cloud/workspace/%s/billing
    `, metric, threshold, current, limit, ws.ID)
    
    s.emailService.Send(ctx, ws.OwnerEmail, subject, body)
}
```

## Billing Dashboard

### Usage Display (Next.js)

```typescript
// cmd/dashboard/app/workspace/[id]/billing/page.tsx
import { Card } from '@/components/ui/card'
import { Progress } from '@/components/ui/progress'

export default async function BillingPage({ params }: { params: { id: string } }) {
  const usage = await fetch(
    `http://control-plane:8080/api/v1/workspaces/${params.id}/usage`
  ).then(r => r.json())
  
  const plan = usage.plan === 'pro' ? 'Pro Plan' : 'Free Plan'
  
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold">Billing & Usage</h1>
        <p className="text-muted-foreground">Current plan: {plan}</p>
      </div>
      
      {/* MAU Usage */}
      <Card className="p-6">
        <h2 className="text-xl font-semibold mb-4">Monthly Active Users</h2>
        <div className="space-y-2">
          <div className="flex justify-between">
            <span>{usage.usage.mau.current.toLocaleString()} / {usage.usage.mau.included.toLocaleString()} included</span>
            <span className="font-semibold">
              {usage.usage.mau.overage > 0 && `+$${(usage.usage.mau.cost / 100).toFixed(2)}`}
            </span>
          </div>
          <Progress value={(usage.usage.mau.current / usage.usage.mau.included) * 100} />
        </div>
      </Card>
      
      {/* Storage Usage */}
      <Card className="p-6">
        <h2 className="text-xl font-semibold mb-4">Storage</h2>
        <div className="space-y-2">
          <div className="flex justify-between">
            <span>{usage.usage.storage.gb.toFixed(2)} GB / {usage.usage.storage.included} GB included</span>
            <span className="font-semibold">
              {usage.usage.storage.overage > 0 && `+$${(usage.usage.storage.cost / 100).toFixed(2)}`}
            </span>
          </div>
          <Progress value={(usage.usage.storage.gb / usage.usage.storage.included) * 100} />
        </div>
      </Card>
      
      {/* Estimated Cost */}
      <Card className="p-6">
        <h2 className="text-xl font-semibold mb-4">Current Billing Period</h2>
        <div className="flex justify-between items-center">
          <span className="text-muted-foreground">Estimated cost for {usage.period}</span>
          <span className="text-3xl font-bold">${(usage.estimatedCost / 100).toFixed(2)}</span>
        </div>
        <p className="text-sm text-muted-foreground mt-2">
          Billing date: {new Date(usage.billingDate).toLocaleDateString()}
        </p>
      </Card>
      
      {/* Recent Invoices */}
      <Card className="p-6">
        <h2 className="text-xl font-semibold mb-4">Recent Invoices</h2>
        <InvoiceList workspaceId={params.id} />
      </Card>
    </div>
  )
}
```

## Tax Handling

### Tax Calculation

```go
// internal/billing/tax.go
package billing

import (
    "github.com/stripe/stripe-go/v76/tax/calculation"
)

// Calculate tax using Stripe Tax
func (s *StripeService) CalculateTax(ctx context.Context, ws *Workspace, amount int64) (int64, error) {
    params := &stripe.TaxCalculationParams{
        Currency: stripe.String("usd"),
        LineItems: []*stripe.TaxCalculationLineItemParams{
            {
                Amount:    stripe.Int64(amount),
                Reference: stripe.String("subscription"),
            },
        },
        CustomerDetails: &stripe.TaxCalculationCustomerDetailsParams{
            Address: &stripe.AddressParams{
                Country: stripe.String(ws.BillingCountry),
                State:   stripe.String(ws.BillingState),
            },
            AddressSource: stripe.String("billing"),
        },
    }
    
    calc, err := calculation.New(params)
    if err != nil {
        return 0, err
    }
    
    return calc.TaxAmountExclusive, nil
}
```

---

**Next:** [Security Model](./SECURITY.md)

