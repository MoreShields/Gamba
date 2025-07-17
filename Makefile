.PHONY: help dev down build test clean deploy proto

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Development commands
dev: ## Start complete development environment (discord-client + lol-tracker + postgres + nats)
	@echo "Stopping and removing any existing containers..."
	@docker-compose -f docker-compose.yml -f docker-compose.dev.yml --profile discord --profile lol down --remove-orphans 2>/dev/null || true
	@echo "Removing any conflicting containers..."
	@docker rm -f gambler-nats gambler-postgres discord-bot lol-tracker discord-migrate lol-tracker-migrate 2>/dev/null || true
	@if [ -f .env ]; then \
		set -a; source .env; set +a; \
		docker-compose -f docker-compose.yml -f docker-compose.dev.yml --profile discord --profile lol up --build; \
	else \
		echo "Warning: .env file not found. Some environment variables may not be set."; \
		docker-compose -f docker-compose.yml -f docker-compose.dev.yml --profile discord --profile lol up --build; \
	fi

down: ## Stop all development containers
	docker-compose -f docker-compose.yml -f docker-compose.dev.yml --profile discord --profile lol down


# Production deployment commands (used by GitHub Actions)
deploy: ## Deploy all services in production
	docker-compose -f docker-compose.yml --profile discord pull
	docker-compose -f docker-compose.yml --profile discord down
	docker-compose -f docker-compose.yml --profile discord up -d

verify-deployment: ## Verify production deployment status
	@echo "=== All Services Status ==="
	@docker-compose -f docker-compose.yml ps
	@echo ""
	@echo "=== Discord Bot ==="
	@docker-compose -f docker-compose.yml ps discord-bot discord-migrate || echo "Discord bot not deployed"
	@echo ""
	@echo "=== LoL Tracker ==="
	@docker-compose -f docker-compose.yml ps lol-tracker nats || echo "LoL Tracker not deployed"

prod-logs: ## View production logs (use SERVICE=discord|lol|nats to specify)
	@if [ "$(SERVICE)" = "discord" ]; then \
		docker-compose -f docker-compose.yml logs -f discord-bot; \
	elif [ "$(SERVICE)" = "lol" ]; then \
		docker-compose -f docker-compose.yml logs -f lol-tracker; \
	elif [ "$(SERVICE)" = "nats" ]; then \
		docker-compose -f docker-compose.yml logs -f nats; \
	else \
		docker-compose -f docker-compose.yml logs -f; \
	fi

# Protobuf commands
proto: ## Generate protobuf code for all services
	$(MAKE) -C api generate

# Build commands
build: proto build-discord build-lol ## Build all services
	@echo "All services built successfully"

build-discord: ## Build discord client service
	$(MAKE) -C discord-client build

build-lol: ## Build lol-tracker service
	$(MAKE) -C lol-tracker build

# Docker build commands (for CI/CD)
docker-build: docker-build-discord docker-build-lol ## Build all Docker images

docker-build-discord: ## Build Discord bot Docker image
	docker build -f discord-client/Dockerfile --target prod .

docker-build-lol: ## Build LoL tracker Docker image
	docker build -f lol-tracker/Dockerfile --target production .

# Test commands
test: test-discord test-lol ## Run tests for all services

test-discord: ## Run tests for discord service
	$(MAKE) -C discord-client test

test-lol: ## Run tests for lol service
	$(MAKE) -C lol-tracker test

test-unit: ## Run unit tests for all services
	$(MAKE) -C discord-client test-unit
	$(MAKE) -C lol-tracker test-unit

test-integration: ## Run integration tests for all services
	$(MAKE) -C discord-client test-integration
	$(MAKE) -C lol-tracker test-integration

# Clean commands
clean: clean-api clean-discord clean-lol ## Clean build artifacts for all services

clean-api: ## Clean API artifacts
	$(MAKE) -C api clean

clean-discord: ## Clean discord service artifacts
	$(MAKE) -C discord-client clean

clean-lol: ## Clean lol service artifacts
	$(MAKE) -C lol-tracker clean

setup-venv: ## Create and setup Python virtual environment with all dev dependencies
	@if [ ! -d "venv" ]; then \
		echo "Creating Python virtual environment..."; \
		python3 -m venv venv; \
		./venv/bin/pip install --upgrade pip; \
	fi
	@echo "Installing lol-tracker dependencies..."
	@./venv/bin/pip install -r lol-tracker/requirements-dev.txt
	@echo "Virtual environment setup complete!"

clean-venv: ## Remove Python virtual environment
	rm -rf venv/

# Database commands
db-shell-discord: ## Connect to the discord database shell
	docker-compose -f docker-compose.yml -f docker-compose.dev.yml exec postgres psql -U gambler -d gambler_db

db-shell-lol: ## Connect to the lol database shell
	docker-compose -f docker-compose.yml -f docker-compose.dev.yml exec postgres psql -U gambler -d lol_tracker_db

db-drop-discord: ## Drop and recreate the discord development database
	docker-compose -f docker-compose.yml -f docker-compose.dev.yml exec postgres dropdb -U gambler gambler_db
	docker-compose -f docker-compose.yml -f docker-compose.dev.yml exec postgres createdb -U gambler gambler_db

db-drop-lol: ## Drop and recreate the lol development database
	docker-compose -f docker-compose.yml -f docker-compose.dev.yml exec postgres dropdb -U gambler lol_tracker_db
	docker-compose -f docker-compose.yml -f docker-compose.dev.yml exec postgres createdb -U gambler lol_tracker_db

# Migration commands
migrate-dev-up-discord: ## Run pending database migrations for discord service (local development)
	$(MAKE) -C discord-client migrate-up-dev

migrate-dev-up-lol: ## Run pending database migrations for lol service (local development)
	$(MAKE) setup-venv
	$(MAKE) -C lol-tracker migrate-up-dev

migrate-up-discord: ## Run pending database migrations for discord service (production)
	$(MAKE) -C discord-client migrate-up

migrate-up-lol: ## Run pending database migrations for lol service (production)
	$(MAKE) setup-venv
	$(MAKE) -C lol-tracker migrate-up

migrate-down-discord: ## Rollback last database migration for discord service
	$(MAKE) -C discord-client migrate-down

migrate-down-lol: ## Rollback last database migration for lol service
	$(MAKE) setup-venv
	$(MAKE) -C lol-tracker migrate-down

migrate-status-discord: ## Check migration status for discord service
	$(MAKE) -C discord-client migrate-status

migrate-status-lol: ## Check migration status for lol service
	$(MAKE) setup-venv
	$(MAKE) -C lol-tracker migrate-status

migrate-create-discord: ## Create a new migration file for discord service (usage: make migrate-create-discord NAME=add_feature)
	$(MAKE) -C discord-client migrate-create NAME=$(NAME)

migrate-create-lol: ## Create a new migration file for lol service (usage: make migrate-create-lol NAME=add_feature)
	$(MAKE) setup-venv
	$(MAKE) -C lol-tracker migrate-create NAME=$(NAME)

# Logging
logs: ## View service logs (use SERVICE=discord|lol|postgres|nats to specify, or all if not specified)
	@if [ "$(SERVICE)" = "discord" ]; then \
		docker-compose -f docker-compose.yml -f docker-compose.dev.yml logs -f discord-bot; \
	elif [ "$(SERVICE)" = "lol" ]; then \
		docker-compose -f docker-compose.yml -f docker-compose.dev.yml logs -f lol-tracker; \
	elif [ "$(SERVICE)" = "postgres" ]; then \
		docker-compose -f docker-compose.yml -f docker-compose.dev.yml logs -f postgres; \
	elif [ "$(SERVICE)" = "nats" ]; then \
		docker-compose -f docker-compose.yml -f docker-compose.dev.yml logs -f nats; \
	else \
		docker-compose -f docker-compose.yml -f docker-compose.dev.yml logs -f; \
	fi