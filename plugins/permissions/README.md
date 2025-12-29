# AuthSome Permissions Plugin

**Enterprise-grade permissions system with ABAC, dynamic resources, and CEL policy language.**

## Features

### 🚀 Advanced Authorization
- **ABAC (Attribute-Based Access Control)** - Policy decisions based on user, resource, and request attributes
- **CEL Policy Language** - Google's Common Expression Language for safe, fast policy evaluation
- **Dynamic Resources** - Organizations define their own resource types and actions
- **Sub-millisecond latency** - <5ms p99 with 10K+ policies per org

### 🏢 Multi-Tenant SaaS
- **Organization-scoped namespaces** - Complete isolation between tenants
- **Custom resource definitions** - Each org defines what they need to protect
- **Policy templates** - Pre-built patterns for common scenarios
- **Platform inheritance** - Share common policies across organizations

### ⚡ Performance
- **Three-tier caching** - Local LRU + Redis + Database
- **Compiled policies** - Parse once, evaluate millions of times
- **Parallel evaluation** - Concurrent policy checks with early exit
- **Query optimization** - Cost-based evaluation ordering

### 🔧 Production-Ready
- **Migration tool** - Seamless migration from existing RBAC
- **Hybrid mode** - Run both systems during transition
- **Comprehensive metrics** - Prometheus + OpenTelemetry integration
- **Audit logging** - Complete trail of policy changes
- **Policy versioning** - Rollback to previous versions

## Quick Start

### 1. Enable the Plugin

Add to your `authsome.yaml`:

```yaml
auth:
  permissions:
    enabled: true
    mode: hybrid  # Use hybrid mode during migration
    
    engine:
      maxPolicyComplexity: 100
      evaluationTimeout: 10ms
      parallelEvaluation: true
    
    cache:
      enabled: true
      backend: redis
      localCacheSize: 10000
```

### 2. Register the Plugin

```go
import (
    "github.com/xraph/authsome"
    "github.com/xraph/authsome/plugins/permissions"
)

func main() {
    auth := authsome.New(authsome.Config{
        ModeSaaS: true,
        // ... other config
    })
    
    // Register permissions plugin
    auth.RegisterPlugin(permissions.NewPlugin())
    
    // Initialize
    if err := auth.Init(); err != nil {
        log.Fatal(err)
    }
}
```

### 3. Create Your First Policy

```bash
curl -X POST http://localhost:8080/permissions/policies \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Document owners can edit",
    "expression": "resource.owner == principal.id",
    "resourceType": "document",
    "actions": ["read", "write", "delete"],
    "enabled": true
  }'
```

### 4. Evaluate Permissions

```bash
curl -X POST http://localhost:8080/permissions/evaluate \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "principalId": "user_123",
    "action": "write",
    "resourceType": "document",
    "resourceId": "doc_456"
  }'
```

Response:
```json
{
  "allowed": true,
  "matchedPolicies": ["policy_789"],
  "evaluationTime": "2.3ms"
}
```

## Policy Examples

### Owner-Only Access
```cel
resource.owner == principal.id
```

### Admin or Owner
```cel
has_role("admin") || resource.owner == principal.id
```

### Business Hours Only
```cel
is_weekday() && in_time_range("09:00", "17:00")
```

### IP Allowlist
```cel
ip_in_range(["10.0.0.0/8", "192.168.0.0/16"])
```

### Team Members
```cel
principal.team_id == resource.team_id && 
principal.department in ["engineering", "operations"]
```

### Time-Limited Access
```cel
request.time < resource.expires_at && 
days_since(resource.created_at) < 90
```

### Confidentiality-Based
```cel
resource.metadata.confidentiality == "public" ||
(resource.metadata.confidentiality == "internal" && is_member_of(resource.org_id)) ||
(resource.metadata.confidentiality == "restricted" && principal.clearance_level >= 3)
```

## Organization Setup

### 1. Create Namespace

```bash
curl -X POST http://localhost:8080/permissions/namespaces \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "orgId": "org_abc123",
    "inheritPlatform": true,
    "templateId": "enterprise-base"
  }'
```

### 2. Define Custom Resources

```bash
curl -X POST http://localhost:8080/permissions/resources \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "namespaceId": "ns_xyz",
    "type": "document",
    "attributes": [
      {"name": "owner", "type": "string", "required": true},
      {"name": "team_id", "type": "string"},
      {"name": "confidentiality", "type": "string"},
      {"name": "tags", "type": "array"}
    ]
  }'
```

### 3. Define Actions

```bash
curl -X POST http://localhost:8080/permissions/actions \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "namespaceId": "ns_xyz",
    "name": "export",
    "description": "Export resource data"
  }'
```

## Migration from RBAC

### Automatic Migration

```bash
curl -X POST http://localhost:8080/permissions/migrate/rbac \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "orgId": "org_abc123",
    "dryRun": true,
    "validateEquivalence": true
  }'
```

### Hybrid Mode

Run both systems during migration:

```yaml
auth:
  permissions:
    mode: hybrid  # Try permissions first, fallback to RBAC
```

Check migration status:
```bash
curl http://localhost:8080/permissions/migrate/rbac/status?orgId=org_abc123 \
  -H "Authorization: Bearer $TOKEN"
```

## Configuration Reference

### Engine Settings

```yaml
engine:
  maxPolicyComplexity: 100      # Max operations per policy
  evaluationTimeout: 10ms       # Max evaluation time
  maxPoliciesPerOrg: 10000      # Policy limit per org
  parallelEvaluation: true      # Concurrent evaluation
  maxParallelEvaluations: 4     # Concurrency level
  enableAttributeCaching: true  # Cache attributes
  attributeCacheTTL: 5m         # Attribute cache TTL
```

### Cache Settings

```yaml
cache:
  enabled: true
  backend: hybrid               # memory, redis, hybrid
  localCacheSize: 10000         # LRU cache size
  localCacheTTL: 5m             # Local TTL
  redisTTL: 15m                 # Redis TTL
  warmupOnStart: true           # Pre-load on startup
  invalidateOnChange: true      # Immediate invalidation
```

### Performance Tuning

```yaml
performance:
  enableMetrics: true           # Prometheus metrics
  enableTracing: false          # OpenTelemetry traces
  traceSamplingRate: 0.01       # 1% sampling
  slowQueryThreshold: 5ms       # Log slow queries
  enableProfiling: false        # pprof endpoints
```

## Performance Benchmarks

| Scenario | Latency (p50) | Latency (p99) | Throughput |
|----------|---------------|---------------|------------|
| Simple policy (owner check) | <1ms | <2ms | 50K RPS |
| Complex policy (ABAC) | 2ms | 4ms | 25K RPS |
| 1K policies, local cache | 1ms | 3ms | 30K RPS |
| 1K policies, Redis cache | 2ms | 5ms | 20K RPS |
| 10K policies, optimized | 3ms | 8ms | 15K RPS |

*Benchmarked on: 4 CPU, 8GB RAM, PostgreSQL + Redis*

## API Reference

### Policy Management

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/permissions/policies` | POST | Create policy |
| `/permissions/policies` | GET | List policies |
| `/permissions/policies/:id` | GET | Get policy |
| `/permissions/policies/:id` | PUT | Update policy |
| `/permissions/policies/:id` | DELETE | Delete policy |
| `/permissions/policies/validate` | POST | Validate syntax |
| `/permissions/policies/test` | POST | Test policy |

### Resource Management

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/permissions/resources` | POST | Define resource type |
| `/permissions/resources` | GET | List resource types |
| `/permissions/resources/:id` | GET | Get resource definition |
| `/permissions/resources/:id` | DELETE | Delete resource type |

### Evaluation

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/permissions/evaluate` | POST | Check authorization |
| `/permissions/evaluate/batch` | POST | Batch evaluation |

### Templates

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/permissions/templates` | GET | List templates |
| `/permissions/templates/:id` | GET | Get template |
| `/permissions/templates/:id/instantiate` | POST | Use template |

## Monitoring

### Prometheus Metrics

```
permissions_evaluations_total{org_id, resource_type, action, result}
permissions_evaluation_duration_seconds{org_id}
permissions_cache_hits_total{tier="local|redis|db"}
permissions_policy_count{org_id}
permissions_errors_total{type}
```

### Grafana Dashboard

Import dashboard from: `/plugins/permissions/monitoring/grafana-dashboard.json`

### Health Check

```bash
curl http://localhost:8080/health/permissions
```

## Security Considerations

### Policy Validation
- All policies validated at creation time
- Type checking prevents runtime errors
- Complexity limits prevent DoS

### Cache Security
- Cache keys include org ID for isolation
- Redis ACLs recommended
- Automatic cache invalidation on updates

### Audit Trail
- All policy changes logged
- Actor, timestamp, old/new values tracked
- Queryable audit API

### Rate Limiting
- Per-org policy limits
- Evaluation timeout protection
- Request rate limiting recommended

## Troubleshooting

### Slow Evaluations

Check metrics:
```bash
curl http://localhost:8080/metrics | grep permissions_evaluation_duration
```

Enable profiling:
```yaml
performance:
  enableProfiling: true
```

Profile: `curl http://localhost:8080/debug/pprof/profile`

### Cache Misses

Check hit rate:
```bash
curl http://localhost:8080/metrics | grep permissions_cache_hits
```

Adjust cache sizes:
```yaml
cache:
  localCacheSize: 20000  # Increase
  localCacheTTL: 10m     # Increase
```

### Policy Errors

Validate before creating:
```bash
curl -X POST http://localhost:8080/permissions/policies/validate \
  -d '{"expression": "resource.owner == principal.id"}'
```

Test with sample data:
```bash
curl -X POST http://localhost:8080/permissions/policies/test \
  -d '{
    "expression": "...",
    "testCases": [...]
  }'
```

## Documentation

- **[Policy Language Reference](./POLICY_LANGUAGE.md)** - Complete CEL syntax guide
- **[Implementation Plan](./IMPLEMENTATION_PLAN.md)** - Development roadmap
- **[Migration Guide](./docs/MIGRATION.md)** - RBAC to Permissions migration
- **[API Reference](./docs/API.md)** - Complete REST API documentation
- **[Performance Guide](./docs/PERFORMANCE.md)** - Optimization strategies

## Examples

See `/examples/permissions/` for:
- Basic policy examples
- ABAC scenarios
- Multi-tenant setup
- Performance testing
- Client integration

## Support

- **Issues**: https://github.com/xraph/authsome/issues
- **Discussions**: https://github.com/xraph/authsome/discussions
- **Enterprise Support**: support@authsome.dev

## License

Same as AuthSome (see root LICENSE file)

