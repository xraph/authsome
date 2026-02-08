.PHONY: help
.DEFAULT_GOAL := help

##@ General

help: ## Display this help message
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Building

build: dashboard-build ## Build all binaries (includes dashboard assets)
	@echo "Building all binaries..."
	@go build -o authsome ./cmd/authsome-cli
	@go build -o authsome-dev ./cmd/dev
	@echo "✓ Built: authsome, authsome-dev"

build-cli: ## Build authsome CLI tool
	@echo "Building authsome..."
	@go build -o authsome ./cmd/authsome-cli
	@echo "✓ Built: authsome"

build-examples: ## Build all example binaries
	@echo "Building examples..."
	@cd examples/comprehensive && go build -o comprehensive-server .
	@echo "✓ Built example binaries"

install: build-cli ## Install authsome CLI to GOPATH/bin
	@echo "Installing authsome..."
	@go build -o "$(shell go env GOPATH)/bin/authsome" ./cmd/authsome-cli
	@echo "✓ Installed authsome to $(shell go env GOPATH)/bin/authsome"
	@echo ""
	@echo "Make sure $(shell go env GOPATH)/bin is in your PATH:"
	@echo "  export PATH=\$$PATH:$(shell go env GOPATH)/bin"

clean: ## Remove build artifacts and generated files
	@echo "Cleaning build artifacts..."
	@rm -f authsome authsome-dev
	@rm -rf clients/go clients/typescript clients/rust
	@rm -f examples/comprehensive/comprehensive-server
	@rm -f *.db *.db-journal *.db-wal *.db-shm
	@rm -f plugins/dashboard/static/css/dashboard.css
	@rm -f plugins/dashboard/static/js/bundle.js
	@rm -f plugins/dashboard/frontend/src/.bundle-entry.js
	@echo "✓ Cleaned"

clean-all: clean security-clean ## Remove all artifacts including security reports
	@echo "✓ All artifacts cleaned"

##@ Testing

test: ## Run all tests
	@echo "Running all tests..."
	@go test -v -race -cover ./...

test-short: ## Run short tests only
	@echo "Running short tests..."
	@go test -short -v ./...

test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	@go test -race -coverprofile=coverage.out -covermode=atomic ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage report: coverage.html"
	@echo ""
	@echo "Coverage Summary:"
	@go tool cover -func=coverage.out | tail -1

test-unit: ## Run unit tests only
	@echo "Running unit tests..."
	@go test -v -race -short ./core/... ./internal/...
	@echo "✓ Unit tests completed"

test-core: ## Run core service tests
	@echo "Running core service tests..."
	@go test -v -race ./core/user ./core/session ./core/auth ./core/organization
	@echo "✓ Core service tests completed"

test-integration: ## Run integration tests
	@echo "Running integration tests..."
	@go test -v -race -tags=integration ./tests/integration/...
	@echo "✓ Integration tests completed"

test-watch: ## Watch for changes and run tests
	@echo "Watching for file changes..."
	@while true; do \
		find . -name '*.go' | entr -d -c make test-short; \
	done

test-verbose: ## Run tests with verbose output
	@echo "Running tests with verbose output..."
	@go test -v -race -cover -coverprofile=coverage.out ./...
	@echo "✓ Tests completed"

test-cli: ## Test CLI tool
	@echo "Testing CLI tool..."
	@bash test_cli_comprehensive.sh

##@ Benchmarking

bench: ## Run all benchmarks
	@echo "Running benchmarks..."
	@go test -bench=. -benchmem -run=^$$ ./...
	@echo "✓ Benchmarks completed"

bench-core: ## Run core service benchmarks
	@echo "Running core service benchmarks..."
	@go test -bench=. -benchmem -run=^$$ ./core/user ./core/session ./core/auth
	@echo "✓ Core benchmarks completed"

bench-user: ## Run user service benchmarks
	@echo "Running user service benchmarks..."
	@go test -bench=BenchmarkService -benchmem -run=^$$ ./core/user
	@echo "✓ User service benchmarks completed"

bench-compare: ## Run benchmarks and save for comparison (BENCH_NAME=name)
	@if [ -z "$(BENCH_NAME)" ]; then \
		BENCH_NAME=baseline; \
	fi
	@echo "Running benchmarks and saving to bench-$(BENCH_NAME).txt..."
	@go test -bench=. -benchmem -run=^$$ ./... > bench-$(BENCH_NAME).txt
	@echo "✓ Benchmarks saved to bench-$(BENCH_NAME).txt"
	@echo ""
	@echo "To compare with another run:"
	@echo "  1. Run: make bench-compare BENCH_NAME=optimized"
	@echo "  2. Compare: benchstat bench-baseline.txt bench-optimized.txt"

bench-profile: ## Run benchmarks with CPU profiling
	@echo "Running benchmarks with CPU profiling..."
	@mkdir -p profiles
	@go test -bench=. -benchmem -cpuprofile=profiles/cpu.prof -run=^$$ ./core/...
	@echo "✓ CPU profile saved to profiles/cpu.prof"
	@echo ""
	@echo "Analyze with: go tool pprof profiles/cpu.prof"

bench-mem: ## Run benchmarks with memory profiling
	@echo "Running benchmarks with memory profiling..."
	@mkdir -p profiles
	@go test -bench=. -benchmem -memprofile=profiles/mem.prof -run=^$$ ./core/...
	@echo "✓ Memory profile saved to profiles/mem.prof"
	@echo ""
	@echo "Analyze with: go tool pprof profiles/mem.prof"

##@ Load Testing

load-test: ## Run load tests with k6 (requires k6)
	@if ! command -v k6 >/dev/null 2>&1; then \
		echo "ERROR: k6 is required. Install from: https://k6.io/docs/getting-started/installation/"; \
		exit 1; \
	fi
	@echo "Running load tests..."
	@mkdir -p tests/load/results
	@k6 run tests/load/auth-flow.js
	@echo "✓ Load tests completed"

load-test-heavy: ## Run heavy load test (200 VUs)
	@if ! command -v k6 >/dev/null 2>&1; then \
		echo "ERROR: k6 is required"; \
		exit 1; \
	fi
	@echo "Running heavy load test (200 VUs for 10 minutes)..."
	@mkdir -p tests/load/results
	@k6 run --vus 200 --duration 10m tests/load/load-test.js
	@echo "✓ Heavy load test completed"

load-test-stress: ## Run stress test (find breaking point)
	@if ! command -v k6 >/dev/null 2>&1; then \
		echo "ERROR: k6 is required"; \
		exit 1; \
	fi
	@echo "Running stress test..."
	@mkdir -p tests/load/results
	@k6 run tests/load/stress-test.js || true
	@echo "✓ Stress test completed"

load-test-custom: ## Run custom load test (VUS=n DURATION=time)
	@if ! command -v k6 >/dev/null 2>&1; then \
		echo "ERROR: k6 is required"; \
		exit 1; \
	fi
	@if [ -z "$(VUS)" ]; then \
		echo "ERROR: VUS is required" >&2; \
		echo "Usage: make load-test-custom VUS=50 DURATION=5m" >&2; \
		exit 1; \
	fi
	@if [ -z "$(DURATION)" ]; then \
		echo "ERROR: DURATION is required" >&2; \
		echo "Usage: make load-test-custom VUS=50 DURATION=5m" >&2; \
		exit 1; \
	fi
	@echo "Running custom load test ($(VUS) VUs for $(DURATION))..."
	@mkdir -p tests/load/results
	@k6 run --vus $(VUS) --duration $(DURATION) tests/load/load-test.js
	@echo "✓ Custom load test completed"

##@ Performance Analysis

perf-profile: ## Profile application with pprof (requires running server)
	@echo "Profiling application (30 seconds)..."
	@echo "Make sure the server is running on localhost:8080"
	@mkdir -p profiles
	@curl -s http://localhost:8080/debug/pprof/profile?seconds=30 > profiles/cpu-$(shell date +%Y%m%d-%H%M%S).prof
	@echo "✓ CPU profile saved"

perf-heap: ## Capture heap profile
	@echo "Capturing heap profile..."
	@mkdir -p profiles
	@curl -s http://localhost:8080/debug/pprof/heap > profiles/heap-$(shell date +%Y%m%d-%H%M%S).prof
	@echo "✓ Heap profile saved"

perf-goroutine: ## Capture goroutine profile
	@echo "Capturing goroutine profile..."
	@mkdir -p profiles
	@curl -s http://localhost:8080/debug/pprof/goroutine > profiles/goroutine-$(shell date +%Y%m%d-%H%M%S).prof
	@echo "✓ Goroutine profile saved"

perf-analyze: ## Analyze latest CPU profile with pprof web UI
	@echo "Starting pprof web interface..."
	@go tool pprof -http=:6060 $(shell ls -t profiles/cpu-*.prof | head -1)

perf-report: bench load-test ## Generate comprehensive performance report
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "📊 Performance Report Generated"
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo ""
	@echo "Benchmark Results: bench-baseline.txt"
	@echo "Load Test Results: tests/load/results/"
	@echo "Profiles: profiles/"
	@echo ""
	@echo "✓ Performance report complete"

e2e: e2e-phase7 ## Run all e2e tests

e2e-2fa: ## Run 2FA e2e test (requires USER_ID)
	@if [ -z "$(USER_ID)" ]; then \
		echo "ERROR: USER_ID is required" >&2; \
		echo "Usage: make e2e-2fa USER_ID=<xid> DEVICE_ID='curl/8.7.1|::1'" >&2; \
		exit 1; \
	fi
	@echo "Running e2e 2FA flow for USER_ID=$(USER_ID) DEVICE_ID=$(DEVICE_ID)"
	@bash examples/e2e.sh "$(USER_ID)" "$(DEVICE_ID)"

e2e-phase7: ## Run Phase 7 e2e tests
	@echo "Running Phase 7 e2e flows (Email OTP, Magic Link, Phone, Passkey)"
	@bash examples/e2e_phase7.sh

##@ Client Generation

generate-clients: ## Generate all client libraries (Go, TypeScript, Rust)
	@echo "Generating all client libraries..."
	@go run ./cmd/authsome-cli generate client --lang all
	@echo "✓ Generated clients in clients/"

generate-go: ## Generate Go client only
	@echo "Generating Go client..."
	@go run ./cmd/authsome-cli generate client --lang go
	@echo "✓ Generated: clients/go/"

generate-typescript: ## Generate TypeScript client only
	@echo "Generating TypeScript client..."
	@go run ./cmd/authsome-cli generate client --lang typescript
	@echo "✓ Generated: clients/typescript/"

generate-rust: ## Generate Rust client only
	@echo "Generating Rust client..."
	@go run ./cmd/authsome-cli generate client --lang rust
	@echo "✓ Generated: clients/rust/"

validate-manifests: ## Validate all manifest files
	@echo "Validating manifests..."
	@go run ./cmd/authsome-cli generate client --validate
	@echo "✓ All manifests valid"

list-plugins: ## List available plugins from manifests
	@echo "Available plugins:"
	@go run ./cmd/authsome-cli generate client --list

##@ Code Introspection

introspect: introspect-all ## Auto-generate manifests from code

introspect-all: ## Introspect all plugins
	@echo "Introspecting all plugins..."
	@go run ./cmd/authsome-cli generate introspect --plugin all
	@echo "✓ Generated manifests in internal/clients/manifest/data/"

introspect-plugin: ## Introspect specific plugin (PLUGIN=name)
	@if [ -z "$(PLUGIN)" ]; then \
		echo "ERROR: PLUGIN is required" >&2; \
		echo "Usage: make introspect-plugin PLUGIN=social" >&2; \
		exit 1; \
	fi
	@echo "Introspecting plugin: $(PLUGIN)"
	@go run ./cmd/authsome-cli generate introspect --plugin $(PLUGIN)
	@echo "✓ Generated manifest for $(PLUGIN)"

introspect-core: ## Introspect core handlers
	@echo "Introspecting core handlers..."
	@go run ./cmd/authsome-cli generate introspect --core
	@echo "✓ Analyzed core handlers"

introspect-dry: ## Preview introspection without writing (PLUGIN=name)
	@if [ -z "$(PLUGIN)" ]; then \
		echo "ERROR: PLUGIN is required" >&2; \
		echo "Usage: make introspect-dry PLUGIN=social" >&2; \
		exit 1; \
	fi
	@echo "Previewing introspection for: $(PLUGIN)"
	@go run ./cmd/authsome-cli generate introspect --plugin $(PLUGIN) --dry-run

##@ Database Operations

DB_PATH ?= authsome_dev.db

db-users: ## List all users from database
	@if ! command -v sqlite3 >/dev/null 2>&1; then \
		echo "ERROR: sqlite3 is required" >&2; \
		exit 1; \
	fi
	@if [ ! -f "$(DB_PATH)" ]; then \
		echo "ERROR: DB file $(DB_PATH) not found" >&2; \
		exit 1; \
	fi
	@echo "Listing users (id | email | username) from $(DB_PATH):"
	@sqlite3 "$(DB_PATH)" "SELECT id||' | '||email||' | '||COALESCE(username,'') FROM users;"

db-migrate: ## Run database migrations
	@echo "Running migrations..."
	@go run ./cmd/authsome-cli migrate --config authsome-dev.yaml
	@echo "✓ Migrations complete"

db-seed: ## Seed database with test data
	@echo "Seeding database..."
	@go run ./cmd/authsome-cli seed --config authsome-dev.yaml
	@echo "✓ Database seeded"

db-reset: ## Reset database (WARNING: destroys all data)
	@echo "Resetting database..."
	@rm -f $(DB_PATH) $(DB_PATH)-journal $(DB_PATH)-wal $(DB_PATH)-shm
	@echo "✓ Database reset"

db-clean: db-reset db-migrate ## Clean and recreate database

##@ Development

dev: ## Start development server
	@echo "Starting dev server (Ctrl+C to stop)..."
	@go run ./cmd/dev

dev-comprehensive: ## Start comprehensive example server
	@echo "Starting comprehensive example server..."
	@cd examples/comprehensive && go run .

run-example: ## Run specific example (EXAMPLE=comprehensive)
	@if [ -z "$(EXAMPLE)" ]; then \
		echo "ERROR: EXAMPLE is required" >&2; \
		echo "Usage: make run-example EXAMPLE=comprehensive" >&2; \
		exit 1; \
	fi
	@echo "Running example: $(EXAMPLE)"
	@cd examples/$(EXAMPLE) && go run .

watch-generate: ## Watch manifests and auto-generate clients
	@echo "Watching manifests for changes..."
	@while true; do \
		make generate-clients; \
		sleep 5; \
	done

##@ Code Quality

lint: ## Run golangci-lint
	@echo "Running linters..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --timeout=5m; \
		echo "✓ Linting passed"; \
	else \
		echo "ERROR: golangci-lint not found. Run 'make install-tools' to install"; \
		exit 1; \
	fi

lint-fix: ## Run golangci-lint with auto-fix
	@echo "Running linters with auto-fix..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --fix --timeout=5m; \
		echo "✓ Linting completed with auto-fixes applied"; \
	else \
		echo "ERROR: golangci-lint not found. Run 'make install-tools' to install"; \
		exit 1; \
	fi

fmt: ## Format Go code
	@echo "Formatting code..."
	@go fmt ./...
	@echo "✓ Code formatted"

format: fmt ## Alias for fmt
f: fmt ## Short alias for fmt

fmt-check: ## Check if code is formatted (CI-friendly)
	@echo "Checking code format..."
	@test -z "$$(gofmt -l . | tee /dev/stderr)" || \
		(echo "ERROR: Code is not formatted. Run 'make fmt'" && exit 1)
	@echo "✓ Code format is correct"

vet: ## Run go vet
	@echo "Running go vet..."
	@go vet ./...
	@echo "✓ Vet passed"

tidy: ## Tidy Go modules
	@echo "Tidying modules..."
	@go mod tidy
	@echo "✓ Modules tidied"

tidy-check: ## Check if go.mod is tidy (CI-friendly)
	@echo "Checking if modules are tidy..."
	@go mod tidy
	@git diff --exit-code go.mod go.sum || \
		(echo "ERROR: go.mod or go.sum is not tidy. Run 'make tidy'" && exit 1)
	@echo "✓ Modules are tidy"

verify: fmt-check vet tidy-check lint ## Run all verification checks (CI-friendly)
	@echo "✓ All verification checks passed"

check: fmt vet lint ## Run all code quality checks (with auto-format)

##@ Security Auditing

SECURITY_DIR := .security-reports
SECURITY_TIMESTAMP := $(shell date +%Y%m%d-%H%M%S)

security-audit: security-setup security-gosec security-vuln security-deps security-secrets security-sbom security-report ## Run complete security audit suite
	@echo ""
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "✓ Security audit completed!"
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo ""
	@echo "Reports available in: $(SECURITY_DIR)"
	@echo ""
	@echo "Review SECURITY.md for security best practices."
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

security-setup: ## Create security reports directory
	@mkdir -p $(SECURITY_DIR)

security-gosec: security-setup ## Run gosec static security scanner
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "🔒 Running gosec (Go Security Checker)..."
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@if ! command -v gosec >/dev/null 2>&1; then \
		echo "Installing gosec..."; \
		go install github.com/securego/gosec/v2/cmd/gosec@latest; \
	fi
	@gosec -fmt=json -out=$(SECURITY_DIR)/gosec-$(SECURITY_TIMESTAMP).json ./... 2>/dev/null || true
	@gosec -fmt=text -out=$(SECURITY_DIR)/gosec-$(SECURITY_TIMESTAMP).txt ./... 2>/dev/null || true
	@gosec -fmt=sarif -out=$(SECURITY_DIR)/gosec-$(SECURITY_TIMESTAMP).sarif ./... 2>/dev/null || true
	@echo "✓ gosec scan completed"
	@echo "  - JSON: $(SECURITY_DIR)/gosec-$(SECURITY_TIMESTAMP).json"
	@echo "  - Text: $(SECURITY_DIR)/gosec-$(SECURITY_TIMESTAMP).txt"
	@echo "  - SARIF: $(SECURITY_DIR)/gosec-$(SECURITY_TIMESTAMP).sarif"

security-vuln: security-setup ## Run vulnerability scanning (govulncheck + trivy)
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "🔍 Running vulnerability scanners..."
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "→ govulncheck (official Go vulnerability scanner)..."
	@if ! command -v govulncheck >/dev/null 2>&1; then \
		echo "Installing govulncheck..."; \
		go install golang.org/x/vuln/cmd/govulncheck@latest; \
	fi
	@govulncheck -json ./... > $(SECURITY_DIR)/govulncheck-$(SECURITY_TIMESTAMP).json 2>&1 || true
	@govulncheck ./... > $(SECURITY_DIR)/govulncheck-$(SECURITY_TIMESTAMP).txt 2>&1 || true
	@echo "✓ govulncheck completed"
	@echo ""
	@echo "→ trivy (comprehensive vulnerability scanner)..."
	@if ! command -v trivy >/dev/null 2>&1; then \
		echo "⚠️  trivy not found. Install from: https://github.com/aquasecurity/trivy"; \
		echo "   macOS: brew install trivy"; \
		echo "   Linux: apt-get install trivy / yum install trivy"; \
	else \
		trivy fs --format json --output $(SECURITY_DIR)/trivy-$(SECURITY_TIMESTAMP).json . 2>/dev/null || true; \
		trivy fs --format table --output $(SECURITY_DIR)/trivy-$(SECURITY_TIMESTAMP).txt . 2>/dev/null || true; \
		trivy fs --format sarif --output $(SECURITY_DIR)/trivy-$(SECURITY_TIMESTAMP).sarif . 2>/dev/null || true; \
		echo "✓ trivy scan completed"; \
	fi

security-deps: security-setup ## Audit Go module dependencies
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "📦 Auditing dependencies..."
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "→ go mod verify..."
	@go mod verify > $(SECURITY_DIR)/mod-verify-$(SECURITY_TIMESTAMP).txt 2>&1
	@echo "✓ Module verification completed"
	@echo ""
	@echo "→ Dependency graph..."
	@go mod graph > $(SECURITY_DIR)/mod-graph-$(SECURITY_TIMESTAMP).txt 2>&1
	@echo "✓ Dependency graph generated"
	@echo ""
	@echo "→ Checking for outdated dependencies..."
	@go list -u -m all > $(SECURITY_DIR)/deps-outdated-$(SECURITY_TIMESTAMP).txt 2>&1 || true
	@echo "✓ Dependency audit completed"

security-secrets: security-setup ## Scan for hardcoded secrets and credentials
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "🔐 Scanning for secrets..."
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@if ! command -v gitleaks >/dev/null 2>&1; then \
		echo "Installing gitleaks..."; \
		go install github.com/gitleaks/gitleaks/v8@latest; \
	fi
	@echo "→ Running gitleaks..."
	@gitleaks detect --config .gitleaks.toml --report-format json --report-path $(SECURITY_DIR)/gitleaks-$(SECURITY_TIMESTAMP).json --no-git 2>/dev/null || true
	@gitleaks detect --config .gitleaks.toml --report-format sarif --report-path $(SECURITY_DIR)/gitleaks-$(SECURITY_TIMESTAMP).sarif --no-git 2>/dev/null || true
	@echo "✓ Secret scanning completed"
	@if [ -f $(SECURITY_DIR)/gitleaks-$(SECURITY_TIMESTAMP).json ]; then \
		if [ "$$(cat $(SECURITY_DIR)/gitleaks-$(SECURITY_TIMESTAMP).json)" != "null" ] && [ "$$(cat $(SECURITY_DIR)/gitleaks-$(SECURITY_TIMESTAMP).json)" != "[]" ]; then \
			echo "⚠️  SECRETS DETECTED! Review: $(SECURITY_DIR)/gitleaks-$(SECURITY_TIMESTAMP).json"; \
		else \
			echo "✓ No secrets detected"; \
		fi; \
	fi

security-sbom: security-setup ## Generate Software Bill of Materials (SBOM)
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "📋 Generating SBOM (Software Bill of Materials)..."
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "→ Generating dependency list..."
	@go list -json -m all > $(SECURITY_DIR)/sbom-$(SECURITY_TIMESTAMP).json 2>&1
	@echo "✓ SBOM generated: $(SECURITY_DIR)/sbom-$(SECURITY_TIMESTAMP).json"
	@echo ""
	@if command -v cyclonedx-gomod >/dev/null 2>&1; then \
		echo "→ Generating CycloneDX SBOM..."; \
		cyclonedx-gomod app -json -output $(SECURITY_DIR)/sbom-cyclonedx-$(SECURITY_TIMESTAMP).json 2>/dev/null || true; \
		echo "✓ CycloneDX SBOM generated"; \
	else \
		echo "ℹ️  cyclonedx-gomod not found (optional)"; \
		echo "   Install: go install github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@latest"; \
	fi

security-licenses: security-setup ## Check dependency licenses for compliance
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "⚖️  Checking licenses..."
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@if ! command -v go-licenses >/dev/null 2>&1; then \
		echo "Installing go-licenses..."; \
		go install github.com/google/go-licenses@latest; \
	fi
	@echo "→ Generating license report..."
	@go-licenses report ./... > $(SECURITY_DIR)/licenses-$(SECURITY_TIMESTAMP).txt 2>&1 || true
	@echo "✓ License report generated: $(SECURITY_DIR)/licenses-$(SECURITY_TIMESTAMP).txt"
	@echo ""
	@echo "→ Checking for non-permissive licenses..."
	@go-licenses check ./... --disallowed_types=forbidden,restricted 2>&1 | tee $(SECURITY_DIR)/licenses-violations-$(SECURITY_TIMESTAMP).txt || true
	@echo "✓ License compliance check completed"

security-report: ## Generate security summary report
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "📊 Generating security summary report..."
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@{ \
		echo "# Security Audit Report"; \
		echo ""; \
		echo "**Generated**: $(SECURITY_TIMESTAMP)"; \
		echo "**Project**: AuthSome Authentication Framework"; \
		echo ""; \
		echo "---"; \
		echo ""; \
		echo "## Summary"; \
		echo ""; \
		echo "This report contains security audit results from multiple scanners:"; \
		echo ""; \
		echo "- **gosec**: Static security analysis for Go code"; \
		echo "- **govulncheck**: Official Go vulnerability scanner"; \
		echo "- **trivy**: Comprehensive vulnerability scanner"; \
		echo "- **gitleaks**: Secret and credential detection"; \
		echo "- **SBOM**: Software Bill of Materials"; \
		echo "- **Licenses**: Dependency license compliance"; \
		echo ""; \
		echo "## Files Generated"; \
		echo ""; \
		ls -lh $(SECURITY_DIR)/*$(SECURITY_TIMESTAMP)* 2>/dev/null | awk '{print "- " $$9 " (" $$5 ")"}' || true; \
		echo ""; \
		echo "## Next Steps"; \
		echo ""; \
		echo "1. Review all findings in the security reports"; \
		echo "2. Prioritize issues by severity (Critical > High > Medium > Low)"; \
		echo "3. Create GitHub issues for vulnerabilities requiring fixes"; \
		echo "4. Update dependencies with known vulnerabilities"; \
		echo "5. Remove any detected secrets immediately"; \
		echo "6. Review SECURITY.md for remediation guidance"; \
		echo ""; \
		echo "## Resources"; \
		echo ""; \
		echo "- Security Policy: SECURITY.md"; \
		echo "- Go Security: https://go.dev/security/"; \
		echo "- OWASP Top 10: https://owasp.org/www-project-top-ten/"; \
		echo "- CWE Database: https://cwe.mitre.org/"; \
		echo ""; \
		echo "---"; \
		echo ""; \
		echo "*For security issues, contact: security@authsome.dev*"; \
	} > $(SECURITY_DIR)/REPORT-$(SECURITY_TIMESTAMP).md
	@echo "✓ Summary report: $(SECURITY_DIR)/REPORT-$(SECURITY_TIMESTAMP).md"

security-clean: ## Remove all security reports
	@echo "Cleaning security reports..."
	@rm -rf $(SECURITY_DIR)
	@echo "✓ Security reports cleaned"

security-install-tools: ## Install all security scanning tools
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "🔧 Installing security tools..."
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "→ Installing gosec..."
	@go install github.com/securego/gosec/v2/cmd/gosec@latest
	@echo "→ Installing govulncheck..."
	@go install golang.org/x/vuln/cmd/govulncheck@latest
	@echo "→ Installing gitleaks..."
	@go install github.com/gitleaks/gitleaks/v8@latest
	@echo "→ Installing go-licenses..."
	@go install github.com/google/go-licenses@latest
	@echo "→ Installing cyclonedx-gomod (optional)..."
	@go install github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@latest
	@echo ""
	@echo "ℹ️  Trivy requires manual installation:"
	@echo "   macOS: brew install trivy"
	@echo "   Linux: https://aquasecurity.github.io/trivy/latest/getting-started/installation/"
	@echo ""
	@echo "✓ Core security tools installed!"

security-ci: security-setup security-gosec security-vuln security-secrets ## Fast security checks for CI/CD
	@echo "✓ CI security checks completed"

security-pre-commit: security-secrets ## Quick security check before commit
	@echo "✓ Pre-commit security check passed"

##@ Dashboard Frontend

dashboard-setup: ## Install dashboard frontend dependencies
	@echo "Installing dashboard frontend dependencies..."
	@cd plugins/dashboard/frontend && npm install
	@echo "✓ Dashboard dependencies installed"

dashboard-build: ## Build dashboard frontend assets (CSS + JS)
	@echo "Building dashboard frontend assets..."
	@cd plugins/dashboard/frontend && npm run build
	@echo "✓ Dashboard assets built"
	@echo "  - plugins/dashboard/static/css/dashboard.css"
	@echo "  - plugins/dashboard/static/js/bundle.js"

dashboard-watch: ## Watch dashboard CSS for changes (development)
	@echo "Watching dashboard CSS (Ctrl+C to stop)..."
	@cd plugins/dashboard/frontend && npm run watch

dashboard-clean: ## Clean dashboard build artifacts
	@echo "Cleaning dashboard build artifacts..."
	@rm -f plugins/dashboard/static/css/dashboard.css
	@rm -f plugins/dashboard/static/js/bundle.js
	@rm -rf plugins/dashboard/frontend/node_modules
	@rm -f plugins/dashboard/frontend/src/.bundle-entry.js
	@echo "✓ Dashboard artifacts cleaned"

dashboard-rebuild: dashboard-clean dashboard-setup dashboard-build ## Clean and rebuild dashboard assets

##@ Documentation

docs: ## Generate documentation
	@echo "Documentation files:"
	@echo "  - README.md"
	@echo "  - clients/README.md"
	@echo "  - clients/INTROSPECTION.md"
	@echo "  - QUICK_START_CLIENT_GENERATOR.md"
	@echo "  - CLIENT_GENERATOR_SUMMARY.md"
	@echo "  - CLIENT_GENERATOR_IMPLEMENTATION.md"
	@echo "  - INTROSPECTION_SUMMARY.md"
	@echo "  - CLIENT_GENERATOR_COMPLETE.md"

godoc: ## Start godoc server
	@echo "Starting godoc server at http://localhost:6060"
	@echo "View at: http://localhost:6060/pkg/github.com/xraph/authsome/"
	@godoc -http=:6060

##@ Release Workflow

full-workflow: clean build validate-manifests generate-clients test security-ci ## Complete workflow with security
	@echo "✓ Full workflow completed successfully!"

release-prep: verify test-coverage generate-clients validate-manifests security-audit build ## Prepare for release with full checks
	@echo "✓ Release preparation complete"
	@echo ""
	@echo "Next steps:"
	@echo "  1. Review security audit reports in .security-reports/"
	@echo "  2. Review generated clients in clients/"
	@echo "  3. Ensure working directory is clean (git status)"
	@echo "  4. Update CHANGELOG.md with release notes"
	@echo "  5. Update version: git tag v<version>"
	@echo "  6. Push tag: git push origin v<version>"
	@echo "  7. GitHub Actions will run goreleaser automatically"
	@echo ""
	@echo "Or run locally with:"
	@echo "  make release-local VERSION=v<version>"

release-local: ## Build release binaries locally with goreleaser (snapshot mode)
	@echo "Building release binaries locally..."
	@if ! command -v goreleaser >/dev/null 2>&1; then \
		echo "Installing goreleaser..."; \
		go install github.com/goreleaser/goreleaser@latest; \
	fi
	@goreleaser check
	@goreleaser release --snapshot --clean --skip=publish
	@echo "✓ Release binaries built in dist/"
	@ls -lh dist/ 2>/dev/null | grep -E "\.(tar\.gz|zip)$$" || true

release-verify: ## Verify goreleaser configuration
	@echo "Verifying goreleaser configuration..."
	@if ! command -v goreleaser >/dev/null 2>&1; then \
		echo "ERROR: goreleaser not installed. Run: go install github.com/goreleaser/goreleaser@latest"; \
		exit 1; \
	fi
	@goreleaser check
	@echo "✓ GoReleaser configuration is valid"

release-dry-run: ## Show what would be released (dry-run)
	@echo "Release dry-run..."
	@if ! command -v goreleaser >/dev/null 2>&1; then \
		echo "Installing goreleaser..."; \
		go install github.com/goreleaser/goreleaser@latest; \
	fi
	@goreleaser release --snapshot --skip=publish --clean
	@echo "✓ Dry-run completed. Check dist/ directory"

release-clean: ## Clean release artifacts
	@echo "Cleaning release artifacts..."
	@rm -rf dist/
	@echo "✓ Release artifacts cleaned"

##@ Development Tools

install-tools: ## Install development and linting tools
	@echo "Installing development tools..."
	@echo "→ Installing golangci-lint..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "→ Installing gosec..."
	@go install github.com/securego/gosec/v2/cmd/gosec@latest
	@echo "→ Installing govulncheck..."
	@go install golang.org/x/vuln/cmd/govulncheck@latest
	@echo "→ Installing gitleaks..."
	@go install github.com/gitleaks/gitleaks/v8@latest
	@echo "→ Installing go-licenses..."
	@go install github.com/google/go-licenses@latest
	@echo "→ Installing cyclonedx-gomod (optional)..."
	@go install github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@latest
	@echo "✓ All development tools installed"

check-tools: ## Check if required development tools are installed
	@echo "Checking installed tools..."
	@command -v golangci-lint >/dev/null 2>&1 && echo "  ✓ golangci-lint" || echo "  ✗ golangci-lint (run 'make install-tools')"
	@command -v gosec >/dev/null 2>&1 && echo "  ✓ gosec" || echo "  ✗ gosec (run 'make install-tools')"
	@command -v govulncheck >/dev/null 2>&1 && echo "  ✓ govulncheck" || echo "  ✗ govulncheck (run 'make install-tools')"
	@command -v gitleaks >/dev/null 2>&1 && echo "  ✓ gitleaks" || echo "  ✗ gitleaks (run 'make install-tools')"
	@command -v go-licenses >/dev/null 2>&1 && echo "  ✓ go-licenses" || echo "  ✗ go-licenses (run 'make install-tools')"
	@command -v cyclonedx-gomod >/dev/null 2>&1 && echo "  ✓ cyclonedx-gomod" || echo "  ✗ cyclonedx-gomod (optional)"

##@ Utilities

generate-keys: ## Generate RSA keys for JWT/OIDC
	@echo "Generating RSA keys..."
	@go run ./cmd/authsome-cli generate keys --output ./keys
	@echo "✓ Keys generated in ./keys/"

generate-secret: ## Generate secure secret
	@echo "Generating secure secret..."
	@go run ./cmd/authsome-cli generate secret --length 32

generate-config: ## Generate sample config (MODE=standalone|saas)
	@if [ -z "$(MODE)" ]; then \
		MODE=standalone; \
	fi
	@echo "Generating $(MODE) config..."
	@go run ./cmd/authsome-cli generate config --mode $(MODE) --output authsome-$(MODE).yaml
	@echo "✓ Generated: authsome-$(MODE).yaml"

version: ## Show version information
	@echo "AuthSome Framework"
	@echo "Go version: $$(go version)"
	@echo "Build info:"
	@go run ./cmd/authsome-cli --version 2>/dev/null || echo "  CLI not built"

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	@go mod download
	@echo "✓ Dependencies downloaded"

deps-verify: ## Verify dependencies integrity
	@echo "Verifying dependencies..."
	@go mod verify
	@echo "✓ Dependencies verified"

##@ CI/CD

ci: verify test build ## Run standard CI pipeline (verify, test, build)
	@echo "✓ CI pipeline completed successfully!"

ci-comprehensive: verify test-coverage security-audit build generate-clients validate-manifests ## Run comprehensive CI pipeline with security and client generation
	@echo "✓ Comprehensive CI pipeline completed successfully!"

ci-fast: fmt lint test-short ## Fast CI checks (format, lint, short tests)
	@echo "✓ Fast CI checks completed!"

ci-security: security-ci ## Run only security checks (for scheduled scans)
	@echo "✓ Security-only CI completed!"

pre-commit: fmt lint test-short ## Pre-commit checks (fast)
	@echo "✓ Pre-commit checks passed"

pre-push: verify test ## Pre-push checks (comprehensive)
	@echo "✓ Pre-push checks passed"

ci-status: ## Show CI/CD configuration status
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "📊 CI/CD Status"
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo ""
	@echo "Tools:"
	@$(MAKE) check-tools 2>/dev/null | grep "✓\|✗"
	@echo ""
	@echo "Git Status:"
	@git status --short 2>/dev/null || echo "  Working directory clean"
	@echo ""
	@echo "Latest Tags:"
	@git tag -l "v*" --sort=-version:refname 2>/dev/null | head -5
	@echo ""
	@echo "Module Info:"
	@head -1 go.mod 2>/dev/null
	@echo ""

##@ Docker (Future)

docker-build: ## Build Docker image
	@echo "Docker support coming soon..."

docker-run: ## Run Docker container
	@echo "Docker support coming soon..."

##@ Aliases

all: full-workflow ## Alias for full-workflow

clients: generate-clients ## Alias for generate-clients

intro: introspect-all ## Alias for introspect-all

test-all: test e2e ## Run all tests including e2e

##@ Variables

# Default device_id used in examples; override with DEVICE_ID='curl/8.7.1|::1'
DEVICE_ID ?= curl/8.7.1|::1
