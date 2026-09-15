.DEFAULT_GOAL := help
BIN := cryptarium
PKG := ./...

help: ## Show available targets
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "};{printf "  \033[36m%-14s\033[0m %s\n",$$1,$$2}'

build: ## Build the CLI
	go build -o bin/$(BIN) ./cmd/$(BIN)

test: ## Run tests with race detector
	gotestsum --format testname -- -race -coverprofile=coverage.out $(PKG)

lint: ## Lint
	golangci-lint run

fmt: ## Format
	gofumpt -l -w . && goimports -w .

vuln: ## Vulnerability scan
	govulncheck $(PKG)

selfscan: build ## Scan this repo with the tool itself
	./bin/$(BIN) scan . --format markdown

golden: ## Regenerate golden files (review the diff)
	go test $(PKG) -run TestGolden -update

check: fmt lint test ## Everything CI runs

lint-rules: ## Validate rule packs load and IDs are unique
	go test ./internal/rules/ -run 'TestLoadAllRulePacks|TestRuleFixturesExist' -count=1

clean:
	rm -rf bin coverage.out

.PHONY: help build test lint fmt vuln selfscan golden check clean lint-rules
