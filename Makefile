.PHONY: help dev down build test clean prod discord-client lol-tracker proto

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Development commands
dev: ## Start development environment with hot reload and postgres
	@if [ -f .env ]; then \
		set -a; source .env; set +a; docker-compose -f discord-client/docker-compose.dev.yml up; \
	else \
		echo "Warning: .env file not found. Some environment variables may not be set."; \
		docker-compose -f discord-client/docker-compose.dev.yml up; \
	fi

down: ## Stop all development containers
	docker-compose -f discord-client/docker-compose.dev.yml down

# Production commands
prod: ## Build and run production containers
	docker-compose -f discord-client/docker-compose.yml up --build

# Protobuf commands
proto: ## Generate protobuf code for all services
	$(MAKE) -C api generate

# Build commands
build: proto discord-client lol-tracker ## Build all services
	@echo "All services built successfully"

discord-client: ## Build discord client service
	$(MAKE) -C discord-client build

lol-tracker: ## Build lol-tracker service (placeholder)
	@echo "LoL tracker service not implemented yet"

# Test commands
test: ## Run tests for all services
	$(MAKE) -C discord-client test

test-unit: ## Run unit tests for all services
	$(MAKE) -C discord-client test-unit

test-integration: ## Run integration tests for all services
	$(MAKE) -C discord-client test-integration

# Clean commands
clean: ## Clean build artifacts for all services
	$(MAKE) -C api clean
	$(MAKE) -C discord-client clean

# Database commands (delegate to discord-client)
db-shell: ## Connect to the development database shell
	docker-compose -f discord-client/docker-compose.dev.yml exec postgres psql -U gambler -d gambler_db

db-drop: ## Drop and recreate the development database
	docker-compose -f discord-client/docker-compose.dev.yml exec postgres dropdb -U gambler gambler_db
	docker-compose -f discord-client/docker-compose.dev.yml exec postgres createdb -U gambler gambler_db

# Migration commands (delegate to discord-client)
migrate-up: ## Run pending database migrations (production)
	$(MAKE) -C discord-client migrate-up

migrate-down: ## Rollback last database migration (production)
	$(MAKE) -C discord-client migrate-down

migrate-status: ## Check migration status (production)
	$(MAKE) -C discord-client migrate-status

migrate-up-dev: ## Run pending database migrations (local development)
	$(MAKE) -C discord-client migrate-up-dev

migrate-down-dev: ## Rollback last database migration (local development)
	$(MAKE) -C discord-client migrate-down-dev

migrate-status-dev: ## Check migration status (local development)
	$(MAKE) -C discord-client migrate-status-dev

migrate-create: ## Create a new migration file (usage: make migrate-create NAME=add_feature)
	$(MAKE) -C discord-client migrate-create NAME=$(NAME)

# Logging
logs: ## View bot logs (dev)
	docker-compose -f discord-client/docker-compose.dev.yml logs -f bot