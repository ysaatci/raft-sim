# raft-sim developer tasks. Run `make help` for a list.

GO_DIR   := backend
FE_DIR   := frontend
WASM_OUT := $(FE_DIR)/public/raft.wasm
GOROOT   := $(shell go env GOROOT)

.PHONY: help test test-go test-fe test-docker wasm fe-install fe-dev fe-build up down

help: ## Show available targets
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-12s %s\n", $$1, $$2}'

test: test-go test-fe ## Run all tests

test-go: ## Run Go tests with the race detector
	cd $(GO_DIR) && go vet ./... && go test -race ./...

test-docker: ## Run Go tests with -race and coverage in a Linux container
	docker run --rm -v "$(CURDIR)/$(GO_DIR):/src" -v raftsim-gocache:/root/.cache -w /src golang:1.24 \
		sh -c 'go vet ./... && go test -race -cover ./...'

test-fe: ## Run frontend tests
	cd $(FE_DIR) && npm test -- --run

wasm: ## Build the Raft simulator to WebAssembly
	mkdir -p $(FE_DIR)/public
	cd $(GO_DIR) && GOOS=js GOARCH=wasm go build -trimpath -ldflags="-s -w" -o ../$(WASM_OUT) ./cmd/wasm
	cp "$(GOROOT)/lib/wasm/wasm_exec.js" $(FE_DIR)/public/wasm_exec.js

fe-install: ## Install frontend dependencies
	cd $(FE_DIR) && npm ci

fe-dev: wasm ## Start the Vite dev server
	cd $(FE_DIR) && npm run dev

fe-build: wasm ## Production build of the frontend
	cd $(FE_DIR) && npm run build

up: ## Start the 5-node cluster with Docker Compose
	docker compose up --build

down: ## Stop the Docker Compose cluster
	docker compose down
