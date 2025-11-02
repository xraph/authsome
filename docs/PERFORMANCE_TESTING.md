# AuthSome Performance Testing Guide

This guide covers all aspects of performance testing for AuthSome, including unit benchmarks, load testing, profiling, and optimization strategies.

## Table of Contents

1. [Overview](#overview)
2. [Unit Benchmarks](#unit-benchmarks)
3. [Load Testing](#load-testing)
4. [Integration Tests](#integration-tests)
5. [Profiling](#profiling)
6. [Performance Metrics](#performance-metrics)
7. [Optimization Strategies](#optimization-strategies)
8. [CI/CD Integration](#cicd-integration)

---

## Overview

AuthSome has a comprehensive performance testing suite covering:

- **Unit Benchmarks**: Micro-benchmarks for core service methods
- **Load Tests**: Realistic user simulation with k6
- **Integration Tests**: End-to-end flow validation
- **Profiling**: CPU, memory, and goroutine analysis
- **Stress Tests**: Finding system breaking points

### Quick Start

```bash
# Run all unit tests
make test

# Run benchmarks
make bench

# Run load tests (requires k6)
make load-test

# Generate performance report
make perf-report
```

---

## Unit Benchmarks

### Running Benchmarks

```bash
# All benchmarks
make bench

# Core service benchmarks only
make bench-core

# User service benchmarks
make bench-user

# With CPU profiling
make bench-profile

# With memory profiling
make bench-mem
```

### Benchmark Comparison

Track performance changes over time:

```bash
# Baseline
make bench-compare BENCH_NAME=baseline

# After optimization
make bench-compare BENCH_NAME=optimized

# Compare results
benchstat bench-baseline.txt bench-optimized.txt
```

### Example Benchmark Output

```
BenchmarkService_Create-8         50000    35420 ns/op   12456 B/op    145 allocs/op
BenchmarkService_FindByID-8      100000    10234 ns/op    3456 B/op     45 allocs/op
BenchmarkService_FindByEmail-8   100000    11456 ns/op    3789 B/op     48 allocs/op
BenchmarkService_Update-8         80000    15678 ns/op    4567 B/op     52 allocs/op
```

### Performance Targets

| Operation | Target P95 | Target P99 | Allocations |
|-----------|-----------|-----------|-------------|
| User.Create | < 50ms | < 100ms | < 200 allocs |
| User.FindByID | < 10ms | < 20ms | < 50 allocs |
| User.FindByEmail | < 15ms | < 30ms | < 60 allocs |
| Session.Create | < 30ms | < 50ms | < 100 allocs |
| Auth.SignIn | < 100ms | < 200ms | < 250 allocs |

---

## Load Testing

### Prerequisites

Install k6:

```bash
# macOS
brew install k6

# Ubuntu/Debian
sudo gpg -k
sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg \
  --keyserver hkp://keyserver.ubuntu.com:80 \
  --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" \
  | sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update
sudo apt-get install k6

# Docker
docker pull grafana/k6
```

### Load Test Scenarios

#### 1. Basic Auth Flow

Tests complete authentication lifecycle:

```bash
make load-test
```

**Covers:**
- User registration
- User login
- Session validation
- User logout

**Configuration:**
- Duration: 2 minutes
- VUs: 10 concurrent users
- Target: < 500ms p95, < 1% errors

#### 2. Realistic Load

Simulates real-world usage patterns:

```bash
# Custom load
make load-test-custom VUS=50 DURATION=5m

# Heavy load
make load-test-heavy  # 200 VUs for 10 minutes
```

**User Behavior:**
- 30% new users (registration)
- 70% returning users (login)
- 50% profile reads
- 10% profile updates
- 5% logouts

#### 3. Stress Test

Finds breaking points:

```bash
make load-test-stress
```

**Stages:**
1. Ramp: 0 → 100 VUs (2 min)
2. Sustain: 100 VUs (5 min)
3. Spike: 100 → 500 VUs (1 min)
4. Sustain: 500 VUs (5 min)
5. Breaking: 500 → 1000 VUs (2 min)
6. Recovery: 1000 → 0 VUs (2 min)

### Interpreting Results

#### Good Performance

```
✓ checks........................: 99.95% ✓ 19990  ✗ 10
✓ http_req_duration.............: avg=156ms p(95)=298ms p(99)=445ms
✓ http_req_failed...............: 0.05%
✓ http_reqs.....................: 20000 (1000/s)
✓ iteration_duration............: avg=1.2s
```

#### Poor Performance

```
✗ checks........................: 85.23% ✓ 17046  ✗ 2954
✗ http_req_duration.............: avg=890ms p(95)=2.1s p(99)=4.8s
✗ http_req_failed...............: 14.77%
✗ http_reqs.....................: 20000 (333/s)
✗ iteration_duration............: avg=5.6s
```

### Load Test Configuration

Environment variables:

```bash
export BASE_URL="http://localhost:8080"
export API_PATH="/api/auth"
export TEST_DURATION="5m"
export TEST_VUS="50"

make load-test
```

---

## Integration Tests

### Running Integration Tests

```bash
# All integration tests
make test-integration

# Specific test
go test -v -tags=integration ./tests/integration/ -run TestAuthFlow_Complete
```

### Test Coverage

Integration tests cover:

1. **Complete Auth Flow**
   - Registration → Login → Profile Access → Logout
   - Invalid credentials handling
   - Session expiration
   - Rate limiting

2. **Concurrency** *(planned)*
   - Concurrent registrations
   - Concurrent logins
   - Race condition testing

3. **Security** *(planned)*
   - SQL injection attempts
   - XSS prevention
   - CSRF protection

### Writing Integration Tests

```go
// +build integration

func TestMyFeature(t *testing.T) {
    // Setup test database
    db := setupTestDatabase(t)
    defer db.Close()
    
    // Setup AuthSome
    auth := authsome.New(...)
    auth.Initialize(context.Background())
    
    // Create test server
    server := httptest.NewServer(app)
    defer server.Close()
    
    // Run tests
    // ...
}
```

---

## Profiling

### CPU Profiling

#### Option 1: Benchmark Profiling

```bash
# Generate CPU profile
make bench-profile

# Analyze interactively
go tool pprof profiles/cpu.prof

# Web UI
go tool pprof -http=:6060 profiles/cpu.prof
```

#### Option 2: Live Application Profiling

```bash
# Start application with pprof endpoint
# (Usually exposed at /debug/pprof)

# Capture profile
make perf-profile

# Analyze
make perf-analyze
```

### Memory Profiling

```bash
# Benchmark memory profile
make bench-mem

# Live heap profile
make perf-heap

# Analyze
go tool pprof profiles/mem.prof
```

### Goroutine Analysis

```bash
# Capture goroutine profile
make perf-goroutine

# Analyze for leaks
go tool pprof profiles/goroutine-*.prof
```

### Common pprof Commands

```bash
# Top functions by CPU/memory
(pprof) top10

# Show call graph
(pprof) list functionName

# Web visualization
(pprof) web

# Generate flamegraph
(pprof) png > flamegraph.png
```

---

## Performance Metrics

### Key Metrics to Monitor

#### Response Times

| Metric | Good | Warning | Critical |
|--------|------|---------|----------|
| P50 (median) | < 100ms | < 200ms | > 200ms |
| P95 | < 500ms | < 1000ms | > 1000ms |
| P99 | < 1000ms | < 2000ms | > 2000ms |
| P99.9 | < 2000ms | < 5000ms | > 5000ms |

#### Throughput

- **Target**: 1000+ requests/second
- **Minimum**: 500 requests/second
- **Critical**: < 100 requests/second

#### Error Rates

- **Good**: < 0.1% errors
- **Acceptable**: < 1% errors
- **Critical**: > 5% errors

#### Resource Usage

- **CPU**: < 70% average, < 90% peak
- **Memory**: Stable (no leaks), < 2GB resident
- **Goroutines**: Stable count, < 10,000
- **Database Connections**: < 80% pool utilization

### Monitoring Checklist

- [ ] Response time trends
- [ ] Throughput capacity
- [ ] Error rates and types
- [ ] CPU utilization
- [ ] Memory usage and GC pressure
- [ ] Goroutine count
- [ ] Database query performance
- [ ] Cache hit rates
- [ ] Network I/O

---

## Optimization Strategies

### 1. Database Optimization

#### Indexing

```sql
-- Essential indexes
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_sessions_token ON sessions(token);
CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);
```

#### Query Optimization

```go
// BAD: N+1 query
for _, user := range users {
    sessions, _ := repo.FindSessionsByUserID(user.ID)
}

// GOOD: Batch loading
userIDs := extractIDs(users)
sessions, _ := repo.FindSessionsByUserIDs(userIDs)
```

#### Connection Pooling

```go
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(5)
db.SetConnMaxLifetime(5 * time.Minute)
```

### 2. Caching Strategies

#### Session Caching

```go
// Redis cache for distributed systems
cache := redis.NewClient(&redis.Options{
    Addr:         "localhost:6379",
    PoolSize:     100,
    MinIdleConns: 10,
})

// Cache with TTL
cache.Set(ctx, sessionToken, userID, 24*time.Hour)
```

#### Cache Patterns

- **Cache-Aside**: Read from cache, fallback to DB
- **Write-Through**: Write to cache and DB simultaneously
- **Write-Behind**: Write to cache, async to DB

### 3. Concurrency Optimization

#### Worker Pools

```go
// Limit concurrent operations
semaphore := make(chan struct{}, 100)

for _, item := range items {
    semaphore <- struct{}{}
    go func(item Item) {
        defer func() { <-semaphore }()
        process(item)
    }(item)
}
```

#### Batch Processing

```go
// Process in batches
batchSize := 100
for i := 0; i < len(items); i += batchSize {
    end := i + batchSize
    if end > len(items) {
        end = len(items)
    }
    batch := items[i:end]
    processBatch(batch)
}
```

### 4. Memory Optimization

#### Reduce Allocations

```go
// BAD: Creates new slice on each call
func getData() []byte {
    return []byte("data")
}

// GOOD: Reuse buffer
var bufferPool = sync.Pool{
    New: func() interface{} {
        return make([]byte, 1024)
    },
}

func getData() []byte {
    buf := bufferPool.Get().([]byte)
    defer bufferPool.Put(buf)
    // use buf
    return buf[:n]
}
```

#### String Building

```go
// BAD: Multiple allocations
s := "a" + "b" + "c" + "d"

// GOOD: Single allocation
var sb strings.Builder
sb.WriteString("a")
sb.WriteString("b")
sb.WriteString("c")
sb.WriteString("d")
s := sb.String()
```

---

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Performance Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Setup Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - name: Run Unit Tests
        run: make test-coverage
      
      - name: Run Benchmarks
        run: make bench-compare BENCH_NAME=${{ github.sha }}
      
      - name: Compare Performance
        if: github.event_name == 'pull_request'
        run: |
          git fetch origin main
          git checkout origin/main
          make bench-compare BENCH_NAME=main
          benchstat bench-main.txt bench-${{ github.sha }}.txt
      
      - name: Setup k6
        run: |
          curl https://github.com/grafana/k6/releases/download/v0.47.0/k6-v0.47.0-linux-amd64.tar.gz -L | tar xvz --strip-components 1
      
      - name: Run Load Tests
        run: make load-test
      
      - name: Upload Results
        uses: actions/upload-artifact@v3
        with:
          name: performance-results
          path: |
            bench-*.txt
            tests/load/results/
```

### Performance Regression Detection

```bash
# In CI: Compare with main branch
benchstat bench-main.txt bench-pr.txt > bench-diff.txt

# Fail if performance degraded > 10%
if grep -q "slower" bench-diff.txt; then
    echo "Performance regression detected!"
    exit 1
fi
```

---

## Best Practices

### 1. Testing

- **Test Early**: Performance test during development
- **Test Often**: Run benchmarks on every PR
- **Test Realistically**: Use production-like data volumes
- **Test Under Load**: Identify bottlenecks before production

### 2. Monitoring

- **Set Baselines**: Establish performance baselines
- **Track Trends**: Monitor performance over time
- **Alert on Regression**: Automated alerts for degradation
- **Profile Regularly**: Regular profiling to catch issues

### 3. Optimization

- **Profile First**: Don't guess, measure
- **Optimize Hot Paths**: Focus on most-used code
- **Measure Impact**: Verify optimizations work
- **Document Changes**: Record optimization decisions

### 4. Iteration

- **Continuous Improvement**: Regular performance reviews
- **Benchmarking Culture**: Make performance a priority
- **Knowledge Sharing**: Document findings and solutions

---

## Troubleshooting

### High Response Times

1. **Profile the application**: `make perf-profile`
2. **Check database queries**: Enable slow query log
3. **Review cache hit rates**: Ensure caching is effective
4. **Check network latency**: Verify infrastructure
5. **Analyze hot paths**: Focus optimization efforts

### High Memory Usage

1. **Profile memory**: `make bench-mem`
2. **Check for leaks**: `make perf-heap`
3. **Review goroutines**: `make perf-goroutine`
4. **Analyze allocations**: Use pprof alloc profile
5. **Optimize hot paths**: Reduce allocations

### High Error Rates

1. **Check logs**: Review error messages
2. **Verify database**: Connection pool, timeouts
3. **Check rate limiting**: Not too aggressive
4. **Review dependencies**: External service failures
5. **Test error handling**: Proper error propagation

---

## Resources

### Tools

- **k6**: https://k6.io/
- **pprof**: https://golang.org/pkg/net/http/pprof/
- **benchstat**: `go install golang.org/x/perf/cmd/benchstat@latest`
- **Grafana**: https://grafana.com/

### Documentation

- Go Performance: https://go.dev/doc/diagnostics
- k6 Documentation: https://k6.io/docs/
- Database Optimization: https://use-the-index-luke.com/

### Community

- Go Performance Slack: gophers.slack.com #performance
- AuthSome Discussions: github.com/xraph/authsome/discussions

---

## Summary

AuthSome provides comprehensive performance testing tools:

✅ **Unit Benchmarks** - Micro-level performance tracking  
✅ **Load Tests** - Realistic user simulation  
✅ **Integration Tests** - End-to-end validation  
✅ **Profiling** - Deep performance analysis  
✅ **CI/CD Integration** - Automated performance gates  
✅ **Optimization Guides** - Proven strategies  

**Start testing:** `make bench && make load-test && make perf-report`

For questions or issues, visit: https://github.com/xraph/authsome/discussions

