# Makefile for Simple KV Store

.PHONY: help build up down logs status test clean rebuild dev
.PHONY: test-unit test-integration test-performance test-e2e test-all
.PHONY: test-coverage test-race test-bench lint format security
.PHONY: docker-test docker-clean install-tools pre-commit test-ci

# Colors for output
GREEN=\033[0;32m
YELLOW=\033[1;33m
RED=\033[0;31m
NC=\033[0m # No Color

# Default target
help:
	@echo "Simple KV Store - Development & Testing Commands"
	@echo ""
	@echo "$(GREEN)Docker Commands:$(NC)"
	@echo "  build         Build Docker images"
	@echo "  up            Start the cluster (detached)"
	@echo "  down          Stop and remove containers"
	@echo "  logs          Show logs from all containers"
	@echo "  status        Check cluster status"
	@echo "  clean         Stop containers and remove volumes"
	@echo "  rebuild       Clean rebuild (down, build, up)"
	@echo "  shell         Open shell in node1"
	@echo ""
	@echo "$(GREEN)Testing Commands:$(NC)"
	@echo "  test          Run basic cluster tests"
	@echo "  test-unit     Run unit tests with coverage"
	@echo "  test-integration  Run integration tests"
	@echo "  test-performance  Run performance tests"
	@echo "  test-e2e      Run end-to-end tests"
	@echo "  test-all      Run all tests"
	@echo "  test-coverage Generate test coverage report"
	@echo "  test-race     Run tests with race detection"
	@echo "  test-bench    Run benchmarks"
	@echo ""
	@echo "$(GREEN)Code Quality Commands:$(NC)"
	@echo "  lint          Run all linters"
	@echo "  format        Format code"
	@echo "  security      Run security scans"
	@echo "  pre-commit    Run pre-commit hooks"
	@echo "  test-ci       Run local CI/CD pipeline test"
	@echo ""
	@echo "$(GREEN)Development Commands:$(NC)"
	@echo "  dev           Full development setup (rebuild + test)"
	@echo "  install-tools Install development tools"
	@echo "  docker-test   Test in Docker environment"
	@echo ""
	@echo "Examples:"
	@echo "  make rebuild && make test-all"
	@echo "  make test-unit"
	@echo "  make test-performance"

# Build images
build:
	@echo "$(GREEN)🔨 Building Docker images...$(NC)"
	docker-compose build

# Start cluster
up:
	@echo "$(GREEN)🚀 Starting cluster...$(NC)"
	docker-compose up -d

# Stop cluster
down:
	@echo "$(YELLOW)⏹️  Stopping cluster...$(NC)"
	docker-compose down

# Show logs
logs:
	@echo "$(GREEN)📋 Showing cluster logs...$(NC)"
	docker-compose logs -f

# Check status using our CLI
status:
	@echo "$(GREEN)📊 Checking cluster status...$(NC)"
	@echo "Docker containers status:"
	@docker-compose ps
	@echo ""
	@echo "Cluster connectivity status:"
	@./kv-cli.sh status

# Run basic cluster tests
test:
	@echo "$(GREEN)🧪 Running basic cluster tests...$(NC)"
	@echo "Waiting for cluster to be ready..."
	@sleep 3
	@./kv-cli.sh test

# Unit tests
test-unit:
	@echo "$(GREEN)🧪 Running unit tests...$(NC)"
	go test -v -race -coverprofile=coverage.out ./store/... ./server/...
	@echo "$(GREEN)✅ Unit tests completed$(NC)"

# Integration tests
test-integration:
	@echo "$(GREEN)🔗 Running integration tests...$(NC)"
	go test -v -tags=integration ./integration_test.go
	@echo "$(GREEN)✅ Integration tests completed$(NC)"

# Performance tests
test-performance:
	@echo "$(GREEN)⚡ Running performance tests...$(NC)"
	go test -v -run="^TestHighConcurrency|TestThroughput|TestLatency|TestStressTest" -timeout=30m ./performance_test.go
	@echo "$(GREEN)✅ Performance tests completed$(NC)"

# End-to-end tests
test-e2e:
	@echo "$(GREEN)🎭 Running E2E tests...$(NC)"
	@if [ ! -f ./test-auto-recovery.sh ]; then \
		echo "$(RED)❌ test-auto-recovery.sh not found$(NC)"; \
		exit 1; \
	fi
	@chmod +x ./test-auto-recovery.sh
	./test-auto-recovery.sh
	@echo "$(GREEN)✅ E2E tests completed$(NC)"

# Run all tests
test-all: test-unit test-integration test-performance test-e2e
	@echo "$(GREEN)🎉 All tests completed successfully!$(NC)"

# Test coverage report
test-coverage:
	@echo "$(GREEN)📊 Generating test coverage report...$(NC)"
	go test -coverprofile=coverage.out ./store/... ./server/...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"
	go tool cover -func=coverage.out

# Race condition tests
test-race:
	@echo "$(GREEN)🏁 Running race condition tests...$(NC)"
	go test -race -count=10 ./store/... ./server/...
	@echo "$(GREEN)✅ Race condition tests completed$(NC)"

# Benchmark tests
test-bench:
	@echo "$(GREEN)⚡ Running benchmarks...$(NC)"
	go test -bench=. -benchmem -count=3 ./store/... ./server/... | tee benchmark.txt
	@echo "$(GREEN)✅ Benchmarks completed$(NC)"

# Code formatting
format:
	@echo "$(GREEN)📝 Formatting code...$(NC)"
	gofmt -s -w .
	go mod tidy
	@echo "$(GREEN)✅ Code formatted$(NC)"

# Linting
lint:
	@echo "$(GREEN)🔍 Running linters...$(NC)"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "$(YELLOW)⚠️  golangci-lint not found, running basic checks$(NC)"; \
		go vet ./...; \
		gofmt -l . | (! grep .); \
	fi
	@echo "$(GREEN)✅ Linting completed$(NC)"

# Security scan
security:
	@echo "$(GREEN)🛡️  Running security scans...$(NC)"
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "$(YELLOW)⚠️  gosec not found, install with: go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest$(NC)"; \
	fi
	@echo "$(GREEN)✅ Security scan completed$(NC)"

# Pre-commit hooks
pre-commit:
	@echo "$(GREEN)🪝 Running pre-commit hooks...$(NC)"
	@if command -v pre-commit >/dev/null 2>&1; then \
		pre-commit run --all-files; \
	else \
		echo "$(YELLOW)⚠️  pre-commit not found, running manual checks$(NC)"; \
		$(MAKE) format lint test-unit; \
	fi
	@echo "$(GREEN)✅ Pre-commit checks completed$(NC)"

# Install development tools
install-tools:
	@echo "$(GREEN)🔧 Installing development tools...$(NC)"
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
	go install honnef.co/go/tools/cmd/staticcheck@latest
	@echo "$(YELLOW)💡 Consider installing pre-commit: pip install pre-commit$(NC)"
	@echo "$(GREEN)✅ Development tools installed$(NC)"

# Docker-based testing
docker-test:
	@echo "$(GREEN)🐳 Running tests in Docker...$(NC)"
	docker build -f Dockerfile.test -t simple-kv:test .
	docker run --rm simple-kv:test go test -v ./...
	@echo "$(GREEN)✅ Docker tests completed$(NC)"

# Clean everything
clean:
	@echo "$(YELLOW)🧹 Cleaning up...$(NC)"
	docker-compose down -v
	docker system prune -f
	rm -f coverage.out coverage.html benchmark.txt
	@echo "$(GREEN)✅ Cleanup completed$(NC)"

# Docker cleanup
docker-clean:
	@echo "$(YELLOW)🐳 Docker cleanup...$(NC)"
	docker-compose down -v
	docker system prune -f
	docker volume prune -f
	@echo "$(GREEN)✅ Docker cleanup completed$(NC)"

# Full rebuild
rebuild: down build up
	@echo "$(GREEN)🔄 Cluster rebuilt and started$(NC)"
	@echo "Run '$(YELLOW)make test$(NC)' to verify everything works"

# Open shell in node1
shell:
	@echo "$(GREEN)🐚 Opening shell in node1...$(NC)"
	docker-compose exec kv-node1 sh

# Quick development workflow
dev: rebuild test-unit
	@echo "$(GREEN)🚀 Development environment ready!$(NC)"
	@echo "Available commands:"
	@echo "  $(YELLOW)make test-all$(NC)     - Run all tests"
	@echo "  $(YELLOW)make test-e2e$(NC)     - Run E2E tests with cluster"
	@echo "  $(YELLOW)make logs$(NC)         - View cluster logs"

# Test CI/CD pipeline locally
test-ci:
	@echo "$(GREEN)🔄 Testing CI/CD pipeline locally...$(NC)"
	@./test-ci-locally.sh
