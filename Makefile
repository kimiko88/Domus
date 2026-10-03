# ─── Domus Makefile ────────────────────────────────────────────────────────────
# Usage: make <target>
# Requires: go ≥ 1.23, golangci-lint, docker

BINARY        := bin/domus
CLI_BINARY    := bin/domus-cli
MODULE        := github.com/domus-hub/domus
VERSION       ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || echo "unknown")
LDFLAGS       := -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -s -w"
GOFLAGS       := -trimpath

# Docker
IMAGE_NAME    := domus
DOCKER_FILE   := deployments/docker/Dockerfile

.PHONY: all build build-cli run test test-unit test-integration test-e2e test-race \
        coverage lint fmt tidy clean docker-build docker-compose-up docker-compose-down \
        migrate-up migrate-down help

## ── Primary targets ──────────────────────────────────────────────────────────

all: tidy lint test build  ## Run full CI pipeline locally

build:  ## Build the main domus binary
	@mkdir -p bin
	go build $(GOFLAGS) $(LDFLAGS) -o $(BINARY) ./cmd/domus/

build-cli:  ## Build the domus-cli binary
	@mkdir -p bin
	go build $(GOFLAGS) $(LDFLAGS) -o $(CLI_BINARY) ./cmd/domus-cli/

run:  ## Run the server locally (reads domus.toml or DOMUS_CONFIG env)
	go run $(LDFLAGS) ./cmd/domus/

## ── Test targets ─────────────────────────────────────────────────────────────

test:  ## Run all tests (unit + integration + e2e)
	go test -count=1 ./...

test-unit:  ## Run only unit tests (fast, no external services)
	go test -count=1 -short ./internal/...

test-integration:  ## Run integration tests
	go test -count=1 -run Integration ./tests/integration/...

test-e2e:  ## Run end-to-end tests
	go test -count=1 -run E2E ./tests/e2e/...

test-race:  ## Run all tests with race detector
	go test -race -count=1 ./...

coverage:  ## Generate HTML coverage report (opens in browser)
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## ── Code quality ─────────────────────────────────────────────────────────────

lint:  ## Run golangci-lint
	golangci-lint run ./...

fmt:  ## Format code with gofmt and goimports
	gofmt -w -s .
	@command -v goimports >/dev/null 2>&1 && goimports -w -local $(MODULE) . || true

tidy:  ## Tidy and verify Go modules
	go mod tidy
	go mod verify

## ── Docker ───────────────────────────────────────────────────────────────────

docker-build:  ## Build Docker image (linux/arm64)
	docker build \
		--platform linux/arm64 \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		-t $(IMAGE_NAME):$(VERSION) \
		-t $(IMAGE_NAME):latest \
		-f $(DOCKER_FILE) .

docker-compose-up:  ## Start development stack (Mosquitto + Domus)
	docker compose -f deployments/docker/docker-compose.yml up -d

docker-compose-down:  ## Stop development stack
	docker compose -f deployments/docker/docker-compose.yml down

## ── Utilities ────────────────────────────────────────────────────────────────

clean:  ## Remove build artefacts
	rm -rf bin/ coverage.out coverage.html

help:  ## Display this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2}'
