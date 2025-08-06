# Makefile for Simple KV Store

.PHONY: help build up down logs status test clean rebuild

# Default target
help:
	@echo "Simple KV Store - Docker Commands"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  build     Build Docker images"
	@echo "  up        Start the cluster (detached)"
	@echo "  down      Stop and remove containers"
	@echo "  logs      Show logs from all containers"
	@echo "  status    Check cluster status"
	@echo "  test      Run cluster tests"
	@echo "  clean     Stop containers and remove volumes"
	@echo "  rebuild   Clean rebuild (down, build, up)"
	@echo "  shell     Open shell in node1"
	@echo ""
	@echo "Examples:"
	@echo "  make up     # Start cluster"
	@echo "  make test   # Run tests"
	@echo "  make logs   # View logs"

# Build images
build:
	docker-compose build

# Start cluster
up:
	docker-compose up -d

# Stop cluster
down:
	docker-compose down

# Show logs
logs:
	docker-compose logs -f

# Check status using our CLI
status:
	@echo "Docker containers status:"
	@docker-compose ps
	@echo ""
	@echo "Cluster connectivity status:"
	@./kv-cli.sh status

# Run tests
test:
	@echo "Waiting for cluster to be ready..."
	@sleep 3
	@./kv-cli.sh test

# Clean everything
clean:
	docker-compose down -v
	docker system prune -f

# Full rebuild
rebuild: down build up
	@echo "Cluster rebuilt and started"
	@echo "Run 'make test' to verify everything works"

# Open shell in node1
shell:
	docker-compose exec kv-node1 sh

# Quick development workflow
dev: rebuild test
	@echo "Development environment ready!"

# See live logs
logs:
	docker-compose logs -f
