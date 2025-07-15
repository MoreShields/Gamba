.PHONY: help dev down build test clean prod discord-client lol-tracker proto

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Development commands
dev: ## Start complete development environment (discord-client + lol-tracker + postgres)
	@if [ -f .env ]; then \
		set -a; source .env; set +a; \
		docker-compose -f docker-compose.dev.yml up; \
	else \
		echo "Warning: .env file not found. Some environment variables may not be set."; \
		docker-compose -f docker-compose.dev.yml up; \
	fi

down: ## Stop all development containers
	docker-compose -f docker-compose.dev.yml down


# Production deployment commands (used by GitHub Actions)
deploy-discord: ## Deploy Discord bot in production (used by GitHub Actions)
	docker-compose -f discord-client/docker-compose.yml pull
	docker-compose -f discord-client/docker-compose.yml down
	docker-compose -f discord-client/docker-compose.yml up -d

deploy-lol-tracker: ## Deploy LoL tracker in production (used by GitHub Actions)
	docker-compose -f lol-tracker/docker-compose.yml pull
	docker-compose -f lol-tracker/docker-compose.yml down
	docker-compose -f lol-tracker/docker-compose.yml up -d

deploy-all: ## Deploy all services in production
	$(MAKE) deploy-discord
	$(MAKE) deploy-lol-tracker

verify-deployment: ## Verify production deployment status
	@echo "=== Discord Bot Status ==="
	@docker-compose -f discord-client/docker-compose.yml ps
	@echo ""
	@echo "=== LoL Tracker Status ==="
	@docker-compose -f lol-tracker/docker-compose.yml ps || echo "LoL Tracker not deployed"

prod-logs: ## View production logs (use SERVICE=discord|lol to specify)
	@if [ "$(SERVICE)" = "lol" ]; then \
		docker-compose -f lol-tracker/docker-compose.yml logs -f; \
	else \
		docker-compose -f discord-client/docker-compose.yml logs -f; \
	fi

# Protobuf commands
proto: ## Generate protobuf code for all services
	$(MAKE) -C api generate

# Build commands
build: proto discord-client lol-tracker ## Build all services
	@echo "All services built successfully"

discord-client: ## Build discord client service
	$(MAKE) -C discord-client build

lol-tracker: ## Build lol-tracker service
	$(MAKE) -C lol-tracker build

# Docker build commands (for CI/CD)
docker-build-discord: ## Build Discord bot Docker image
	docker build -f discord-client/Dockerfile --target prod discord-client

docker-build-lol-tracker: ## Build LoL tracker Docker image
	docker build -f lol-tracker/Dockerfile --target production .

# Test commands
test: ## Run tests for all services
	$(MAKE) -C discord-client test
	$(MAKE) -C lol-tracker test

test-unit: ## Run unit tests for all services
	$(MAKE) -C discord-client test-unit
	$(MAKE) -C lol-tracker test-unit

test-integration: ## Run integration tests for all services
	$(MAKE) -C discord-client test-integration
	$(MAKE) -C lol-tracker test-integration

# Clean commands
clean: ## Clean build artifacts for all services
	$(MAKE) -C api clean
	$(MAKE) -C discord-client clean
	$(MAKE) -C lol-tracker clean

clean-venv: ## Remove Python virtual environment
	rm -rf venv/

# Database commands
db-shell: ## Connect to the development database shell (use SERVICE=discord|lol to specify)
	@if [ "$(SERVICE)" = "lol" ]; then \
		docker-compose -f docker-compose.dev.yml exec postgres psql -U gambler -d lol_tracker_db; \
	else \
		docker-compose -f docker-compose.dev.yml exec postgres psql -U gambler -d discord_db; \
	fi

db-drop: ## Drop and recreate the development database (use SERVICE=discord|lol to specify)
	@if [ "$(SERVICE)" = "lol" ]; then \
		docker-compose -f docker-compose.dev.yml exec postgres dropdb -U gambler lol_tracker_db; \
		docker-compose -f docker-compose.dev.yml exec postgres createdb -U gambler lol_tracker_db; \
	else \
		docker-compose -f docker-compose.dev.yml exec postgres dropdb -U gambler discord_db; \
		docker-compose -f docker-compose.dev.yml exec postgres createdb -U gambler discord_db; \
	fi

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
logs: ## View service logs (use SERVICE=discord|lol|postgres to specify, or all if not specified)
	@if [ "$(SERVICE)" = "discord" ]; then \
		docker-compose -f docker-compose.dev.yml logs -f discord-bot; \
	elif [ "$(SERVICE)" = "lol" ]; then \
		docker-compose -f docker-compose.dev.yml logs -f lol-tracker; \
	elif [ "$(SERVICE)" = "postgres" ]; then \
		docker-compose -f docker-compose.dev.yml logs -f postgres; \
	else \
		docker-compose -f docker-compose.dev.yml logs -f; \
	fi