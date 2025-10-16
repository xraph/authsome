.PHONY: help
.DEFAULT_GOAL := help

##@ General

help: ## Display this help message
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Building

build: ## Build all binaries
	@echo "Building all binaries..."
	@go build -o authsome-cli ./cmd/authsome-cli
	@go build -o authsome ./cmd/dev
	@echo "✓ Built: authsome-cli, authsome"

build-cli: ## Build authsome-cli tool
	@echo "Building authsome-cli..."
	@go build -o authsome-cli ./cmd/authsome-cli
	@echo "✓ Built: authsome-cli"

build-examples: ## Build all example binaries
	@echo "Building examples..."
	@cd examples/comprehensive && go build -o comprehensive-server .
	@echo "✓ Built example binaries"

install: build-cli ## Install authsome-cli to GOPATH/bin
	@echo "Installing authsome-cli..."
	@go install ./cmd/authsome-cli
	@echo "✓ Installed authsome-cli"

clean: ## Remove build artifacts and generated files
	@echo "Cleaning build artifacts..."
	@rm -f authsome-cli authsome
	@rm -rf clients/generated/*
	@rm -f examples/comprehensive/comprehensive-server
	@rm -f *.db *.db-journal *.db-wal *.db-shm
	@echo "✓ Cleaned"

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

test-cli: ## Test CLI tool
	@echo "Testing CLI tool..."
	@bash test_cli_comprehensive.sh

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
	@go run ./cmd/authsome-cli generate client --lang all --manifest-dir ./clients/manifest/data
	@echo "✓ Generated clients in clients/generated/"

generate-go: ## Generate Go client only
	@echo "Generating Go client..."
	@go run ./cmd/authsome-cli generate client --lang go --manifest-dir ./clients/manifest/data
	@echo "✓ Generated: clients/generated/go/"

generate-typescript: ## Generate TypeScript client only
	@echo "Generating TypeScript client..."
	@go run ./cmd/authsome-cli generate client --lang typescript --manifest-dir ./clients/manifest/data
	@echo "✓ Generated: clients/generated/typescript/"

generate-rust: ## Generate Rust client only
	@echo "Generating Rust client..."
	@go run ./cmd/authsome-cli generate client --lang rust --manifest-dir ./clients/manifest/data
	@echo "✓ Generated: clients/generated/rust/"

validate-manifests: ## Validate all manifest files
	@echo "Validating manifests..."
	@go run ./cmd/authsome-cli generate client --validate --manifest-dir ./clients/manifest/data
	@echo "✓ All manifests valid"

list-plugins: ## List available plugins from manifests
	@echo "Available plugins:"
	@go run ./cmd/authsome-cli generate client --list --manifest-dir ./clients/manifest/data

##@ Code Introspection

introspect: introspect-all ## Auto-generate manifests from code

introspect-all: ## Introspect all plugins
	@echo "Introspecting all plugins..."
	@go run ./cmd/authsome-cli generate introspect --plugin all
	@echo "✓ Generated manifests in clients/manifest/data/"

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

lint: ## Run linters
	@echo "Running linters..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found, install it: https://golangci-lint.run/usage/install/"; \
		exit 1; \
	fi

fmt: ## Format Go code
	@echo "Formatting code..."
	@go fmt ./...
	@echo "✓ Code formatted"

vet: ## Run go vet
	@echo "Running go vet..."
	@go vet ./...
	@echo "✓ No issues found"

tidy: ## Tidy Go modules
	@echo "Tidying modules..."
	@go mod tidy
	@echo "✓ Modules tidied"

check: fmt vet lint ## Run all code quality checks

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

full-workflow: clean build validate-manifests generate-clients test ## Complete workflow: clean, build, validate, generate, test
	@echo "✓ Full workflow completed successfully!"

pre-commit: fmt vet test-short ## Pre-commit checks (fast)
	@echo "✓ Pre-commit checks passed"

pre-push: check test ## Pre-push checks (comprehensive)
	@echo "✓ Pre-push checks passed"

release-prep: clean build test generate-clients validate-manifests ## Prepare for release
	@echo "✓ Release preparation complete"
	@echo ""
	@echo "Next steps:"
	@echo "  1. Review generated clients in clients/generated/"
	@echo "  2. Update version numbers"
	@echo "  3. Update CHANGELOG.md"
	@echo "  4. git tag v<version>"
	@echo "  5. git push --tags"

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

verify: ## Verify dependencies
	@echo "Verifying dependencies..."
	@go mod verify
	@echo "✓ Dependencies verified"

##@ CI/CD

ci: deps check test generate-clients validate-manifests ## Run CI pipeline
	@echo "✓ CI pipeline completed successfully!"

ci-fast: deps test-short lint ## Fast CI checks
	@echo "✓ Fast CI checks completed!"

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
