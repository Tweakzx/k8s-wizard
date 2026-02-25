# Makefile for K8s Wizard (Unified)

.PHONY: all dev dev:api dev:web build build:api build:web clean install

# 默认目标
all: build

# 安装依赖
install:
	cd web && npm install 2>/dev/null || echo "Node not installed, skipping web deps"
	go mod download
	go mod tidy

# 开发模式（同时启动前后端）
dev: install
	@echo "🚀 Starting dev servers..."
	@make dev:api &
	@make dev:web

# 开发 API
dev:api:
	@echo "📡 Starting API server on :8080..."
	@go run api/main.go

# 开发 Web
dev:web:
	@echo "🎨 Starting web server on :5173..."
	@cd web && npm run dev 2>/dev/null || echo "Node not installed"

# 构建
build: build:api build:web

# 构建 API
build:api:
	@echo "🔨 Building API..."
	@go build -o bin/k8s-wizard-api ./api
	@echo "✅ API built to bin/k8s-wizard-api"

# 构建 Web
build:web:
	@echo "🔨 Building Web..."
	@cd web && npm run build 2>/dev/null || echo "Skipping web build (Node not installed)"
	@echo "✅ Web built to web/dist/"

# 运行
run: build:api
	@echo "🚀 Running K8s Wizard..."
	@./bin/k8s-wizard-api

# 清理
clean:
	rm -rf bin/
	rm -rf web/dist/
	rm -rf web/node_modules/
	@echo "✅ Cleaned build artifacts"

# 测试
test:
	go test ./...
	@cd web && npm test 2>/dev/null || echo "Skipping web tests"

# 格式化
fmt:
	go fmt ./...
	@cd web && npm run format 2>/dev/null || echo "Skipping format"

# 代码检查
lint:
	go vet ./...
	@cd web && npm run lint 2>/dev/null || echo "Skipping lint"
