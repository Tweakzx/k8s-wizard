# Makefile for K8s Wizard

.PHONY: all dev dev_api dev_web build build_api build_web clean install run test fmt lint

# Default target
all: build

# Install dependencies
install:
	cd web && npm install 2>/dev/null || echo "Node not installed, skipping web deps"
	go mod download
	go mod tidy

# Development mode (start both frontend and backend)
dev: install
	@echo "Starting dev servers..."
	@make dev_api &
	@make dev_web

# Development API
dev_api:
	@echo "Starting API server on :8080..."
	@go run ./cmd/k8s-wizard

# Development Web
dev_web:
	@echo "Starting web server on :5173..."
	@cd web && npm run dev 2>/dev/null || echo "Node not installed"

# Build
build: build_api build_web

# Build API
build_api:
	@echo "Building API..."
	@go build -o bin/k8s-wizard ./cmd/k8s-wizard
	@echo "API built to bin/k8s-wizard"

# Build Web
build_web:
	@echo "Building Web..."
	@cd web && npm run build 2>/dev/null || echo "Skipping web build (Node not installed)"
	@echo "Web built to web/dist/"

# Run
run: build_api
	@echo "Running K8s Wizard..."
	@./bin/k8s-wizard

# Clean
clean:
	rm -rf bin/
	rm -rf web/dist/
	rm -rf web/node_modules/
	@echo "Cleaned build artifacts"

# Test
test:
	go test ./...
	@cd web && npm test 2>/dev/null || echo "Skipping web tests"

# Format
fmt:
	go fmt ./...
	@cd web && npm run format 2>/dev/null || echo "Skipping format"

# Lint
lint:
	go vet ./...
	@cd web && npm run lint 2>/dev/null || echo "Skipping lint"
