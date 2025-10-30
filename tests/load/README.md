# AuthSome Load Testing

This directory contains load testing scripts for AuthSome using [k6](https://k6.io/).

## Prerequisites

Install k6:

```bash
# macOS
brew install k6

# Linux
sudo gpg -k
sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update
sudo apt-get install k6

# Docker
docker pull grafana/k6
```

## Running Tests

### Quick Tests (Development)

Test basic authentication flows:

```bash
k6 run auth-flow.js
```

### Load Tests (Staging/Production)

Simulate realistic user load:

```bash
# Light load (10 VUs for 1 minute)
k6 run --vus 10 --duration 1m load-test.js

# Medium load (50 VUs for 5 minutes)
k6 run --vus 50 --duration 5m load-test.js

# Heavy load (200 VUs for 10 minutes)
k6 run --vus 200 --duration 10m load-test.js
```

### Stress Tests

Find the breaking point:

```bash
k6 run stress-test.js
```

### Spike Tests

Test sudden traffic bursts:

```bash
k6 run spike-test.js
```

### Soak Tests

Test sustained load over time:

```bash
k6 run --duration 2h soak-test.js
```

## Test Scenarios

### auth-flow.js
- User registration
- User login
- Session validation
- User logout
- Password change

### load-test.js
- Simulates realistic user behavior
- Multiple concurrent users
- Mixed read/write operations
- Validates performance under normal load

### stress-test.js
- Gradually increases load
- Finds system limits
- Tests recovery behavior

### spike-test.js
- Sudden load increases
- Tests auto-scaling
- Validates rate limiting

### soak-test.js
- Extended duration testing
- Memory leak detection
- Resource exhaustion testing

## Performance Thresholds

Tests are configured with the following thresholds:

- **HTTP Duration (p95)**: < 500ms
- **HTTP Duration (p99)**: < 1000ms
- **HTTP Failures**: < 1%
- **Iteration Duration**: < 2s

## Metrics

K6 provides detailed metrics:

- **http_req_duration**: Request duration
- **http_req_failed**: Failed requests ratio
- **http_reqs**: Total requests per second
- **iterations**: Completed iterations
- **vus**: Virtual users

## Cloud Integration

Export results to Grafana Cloud:

```bash
k6 cloud run load-test.js
```

Export to InfluxDB:

```bash
k6 run --out influxdb=http://localhost:8086/k6 load-test.js
```

## Environment Variables

Configure tests via environment variables:

```bash
export BASE_URL="http://localhost:8080"
export TEST_DURATION="5m"
export TEST_VUS="50"

k6 run load-test.js
```

## CI/CD Integration

Run in CI:

```bash
# GitHub Actions
k6 run --quiet --no-color load-test.js

# GitLab CI
k6 run --out json=results.json load-test.js
```

## Interpreting Results

### Good Performance
```
http_req_duration..........: avg=150ms  p(95)=300ms  p(99)=450ms
http_req_failed............: 0.01%
http_reqs..................: 1000/s
```

### Poor Performance
```
http_req_duration..........: avg=800ms  p(95)=2s     p(99)=5s
http_req_failed............: 5.23%
http_reqs..................: 100/s
```

## Troubleshooting

### High Error Rate
- Check server logs
- Verify database connections
- Review rate limiting configuration

### High Latency
- Profile application with pprof
- Check database query performance
- Review network latency

### Resource Exhaustion
- Monitor CPU and memory usage
- Check connection pool sizes
- Review goroutine count

## Best Practices

1. **Start Small**: Begin with low VUs and short durations
2. **Ramp Up Gradually**: Increase load incrementally
3. **Monitor Server**: Watch metrics during tests
4. **Test Realistic Scenarios**: Simulate actual user behavior
5. **Repeat Tests**: Run multiple times for consistency
6. **Test in Staging First**: Never load test production without approval

## Support

For issues or questions:
- Documentation: https://k6.io/docs/
- Community: https://community.k6.io/
- GitHub: https://github.com/grafana/k6

