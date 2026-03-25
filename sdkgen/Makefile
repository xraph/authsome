# AuthSome SDK Generation
#
# Targets:
#   make generate              - Generate all SDKs (Go + TypeScript + Dart) from dynamic spec
#   make generate-go           - Generate Go SDK only
#   make generate-ts           - Generate TypeScript SDK only
#   make generate-dart         - Generate Dart SDK only
#   make generate-ui-core      - Generate embedded client for ui-core package
#   make generate-flutter-core - Generate embedded client for Flutter authsome_core package
#   make dump-spec             - Dump OpenAPI spec from engine route metadata
#   make clean                 - Remove generated SDK output
#
# Environment variables:
#   SDK_OUT     - Output directory (default: ../sdk)
#   API_TITLE   - API title (default: AuthSome API)
#   API_VERSION - API version (default: 0.5.0)
#   SERVER_URL  - Server base URL (optional)
#   SPEC_FILE   - Intermediate spec file (default: spec.json)

SDK_OUT     ?= ../sdk
API_TITLE   ?= AuthSome API
API_VERSION ?= 0.5.0
SERVER_URL  ?=
SPEC_FILE   ?= spec.json

# Method overrides for embedded mode (align generated names with ui-core/flutter-core expectations)
UI_CORE_OVERRIDES = refreshTokens=refresh,deleteAccount=deleteMe,exportUserData=exportMyData,verifyMFARecovery=verifyRecoveryCode
FLUTTER_CORE_OVERRIDES = refreshTokens=refresh,deleteAccount=deleteMe,exportUserData=exportMyData,verifyMFARecovery=verifyRecoveryCode

SDKGEN_CMD  = go run ./cmd
SPECGEN_CMD = go run ./cmd/specgen

# Build flags for the sdkgen command
SDKGEN_FLAGS = --title="$(API_TITLE)" --version=$(API_VERSION)
ifneq ($(SERVER_URL),)
SDKGEN_FLAGS += --server=$(SERVER_URL)
endif

.PHONY: generate generate-go generate-ts generate-dart generate-ui-core generate-flutter-core dump-spec clean help

dump-spec: ## Dump OpenAPI spec from engine route metadata
	@echo "==> Dumping OpenAPI spec from registered routes..."
	$(SPECGEN_CMD) --out=$(SPEC_FILE) --title="$(API_TITLE)" --version=$(API_VERSION)
	@echo "==> Spec written to $(SPEC_FILE)"

generate: dump-spec ## Generate all SDKs (Go + TypeScript + Dart)
	@echo "==> Generating all SDKs..."
	$(SDKGEN_CMD) generate --lang=all --out=$(SDK_OUT) --from-spec=$(SPEC_FILE) $(SDKGEN_FLAGS)
	@echo "==> SDKs generated in $(SDK_OUT)/"

generate-go: dump-spec ## Generate Go SDK only
	@echo "==> Generating Go SDK..."
	$(SDKGEN_CMD) generate --lang=go --out=$(SDK_OUT)/go --from-spec=$(SPEC_FILE) $(SDKGEN_FLAGS)
	@echo "==> Go SDK generated in $(SDK_OUT)/go/"

generate-ts: dump-spec ## Generate TypeScript SDK only
	@echo "==> Generating TypeScript SDK..."
	$(SDKGEN_CMD) generate --lang=typescript --out=$(SDK_OUT)/typescript --from-spec=$(SPEC_FILE) $(SDKGEN_FLAGS)
	@echo "==> TypeScript SDK generated in $(SDK_OUT)/typescript/"

generate-dart: dump-spec ## Generate Dart SDK only
	@echo "==> Generating Dart SDK..."
	$(SDKGEN_CMD) generate --lang=dart --out=$(SDK_OUT)/dart --from-spec=$(SPEC_FILE) $(SDKGEN_FLAGS)
	@echo "==> Dart SDK generated in $(SDK_OUT)/dart/"

generate-ui-core: dump-spec ## Generate embedded client for ui-core package
	@echo "==> Generating embedded TypeScript client for ui-core..."
	$(SDKGEN_CMD) generate --lang=typescript --out=../ui/packages/core/src/generated \
		--from-spec=$(SPEC_FILE) --output-mode=embedded --method-overrides=$(UI_CORE_OVERRIDES) $(SDKGEN_FLAGS)
	@echo "==> Embedded client generated in ../ui/packages/core/src/generated/"

generate-flutter-core: dump-spec ## Generate embedded client for Flutter authsome_core package
	@echo "==> Generating embedded Dart client for Flutter authsome_core..."
	$(SDKGEN_CMD) generate --lang=dart --out=../flutter/packages/authsome_core/lib/src/generated \
		--from-spec=$(SPEC_FILE) --output-mode=embedded --method-overrides=$(FLUTTER_CORE_OVERRIDES) $(SDKGEN_FLAGS)
	@echo "==> Embedded Dart client generated in ../flutter/packages/authsome_core/lib/src/generated/"

clean: ## Remove generated SDK output and spec file
	@echo "==> Cleaning $(SDK_OUT)/ and $(SPEC_FILE)"
	rm -rf $(SDK_OUT) $(SPEC_FILE)

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'
