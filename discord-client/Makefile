.PHONY: help dev down build test clean prod

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

dev: ## Start development environment with hot reload and postgres
	docker-compose -f docker-compose.dev.yml up

down: ## Stop all development containers
	docker-compose -f docker-compose.dev.yml down

prod: ## Build and run production container (requires external postgres)
	docker-compose up --build

build: ## Build the application binary
	go build -o bin/gambler main.go

build-docker: ## Build production Docker image
	docker build -t gambler:latest .

test: ## Run all tests
	go test -v ./...

test-unit: ## Run unit tests only (excluding integration tests)
	go test -v -short ./...

test-integration: ## Run integration tests with testcontainers
	go test -v ./repository/...

clean: ## Clean build artifacts
	rm -rf bin/ tmp/

db-shell: ## Connect to the development database shell
	docker-compose -f docker-compose.dev.yml exec postgres psql -U gambler -d gambler_db

db-drop: ## Drop the development database
	docker-compose -f docker-compose.dev.yml exec postgres dropdb -U gambler gambler_db
	docker-compose -f docker-compose.dev.yml exec postgres createdb -U gambler gambler_db

logs: ## View bot logs (dev)
	docker-compose -f docker-compose.dev.yml logs -f bot

migrate-up: ## Run pending database migrations (production)
	docker run --rm --env-file .env gambler:latest ./gambler migrate up 

migrate-down: ## Rollback last database migration (production)
	docker run --rm --env-file .env gambler:latest ./gambler migrate down 1

migrate-status: ## Check migration status (production)
	docker run --rm --env-file .env gambler:latest ./gambler migrate status

migrate-up-dev: ## Run pending database migrations (local development)
	@if [ -f .env ]; then \
		set -a; source .env; set +a; go run main.go migrate up; \
	elif [ -f ../.env ]; then \
		set -a; source ../.env; set +a; go run main.go migrate up; \
	else \
		echo "Error: .env file not found. Create .env in project root or discord-client/ directory."; exit 1; \
	fi

migrate-down-dev: ## Rollback last database migration (local development)
	@if [ -f .env ]; then \
		set -a; source .env; set +a; go run main.go migrate down 1; \
	elif [ -f ../.env ]; then \
		set -a; source ../.env; set +a; go run main.go migrate down 1; \
	else \
		echo "Error: .env file not found. Create .env in project root or discord-client/ directory."; exit 1; \
	fi

migrate-status-dev: ## Check migration status (local development)
	@if [ -f .env ]; then \
		set -a; source .env; set +a; go run main.go migrate status; \
	elif [ -f ../.env ]; then \
		set -a; source ../.env; set +a; go run main.go migrate status; \
	else \
		echo "Error: .env file not found. Create .env in project root or discord-client/ directory."; exit 1; \
	fi

migrate-create: ## Create a new migration file (usage: make migrate-create NAME=add_feature)
	@if [ -z "$(NAME)" ]; then echo "Usage: make migrate-create NAME=<migration_name>"; exit 1; fi
	@TIMESTAMP=$$(date +%Y%m%d%H%M%S); \
	touch database/migrations/$${TIMESTAMP}_$(NAME).up.sql; \
	touch database/migrations/$${TIMESTAMP}_$(NAME).down.sql; \
	echo "Created migration files:"; \
	echo "  database/migrations/$${TIMESTAMP}_$(NAME).up.sql"; \
	echo "  database/migrations/$${TIMESTAMP}_$(NAME).down.sql"