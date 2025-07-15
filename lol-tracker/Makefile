.PHONY: help build test clean dev install lint format proto run docker-build docker-run

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Development setup
install: ## Install dependencies in virtual environment
	python3 -m venv venv
	./venv/bin/pip install --upgrade pip
	./venv/bin/pip install -r requirements-dev.txt

install-prod: ## Install production dependencies only
	python3 -m venv venv
	./venv/bin/pip install --upgrade pip
	./venv/bin/pip install -r requirements.txt

# Code quality
lint: ## Run linting checks
	./venv/bin/flake8 src/ tests/
	./venv/bin/mypy src/

format: ## Format code with black and isort
	./venv/bin/black src/ tests/
	./venv/bin/isort src/ tests/

format-check: ## Check if code is formatted correctly
	./venv/bin/black --check src/ tests/
	./venv/bin/isort --check-only src/ tests/

# Testing
test: ## Run all tests
	PYTHONPATH=src ./venv/bin/pytest tests/ -v

test-unit: ## Run unit tests only
	PYTHONPATH=src ./venv/bin/pytest tests/ -v -k "not integration"

test-integration: ## Run integration tests only
	PYTHONPATH=src ./venv/bin/pytest tests/ -v -k "integration"

test-cov: ## Run tests with coverage
	PYTHONPATH=src ./venv/bin/pytest tests/ --cov=src/lol_tracker --cov-report=html --cov-report=term

# Protocol Buffers
proto: ## Generate Python protobuf code locally for development
	@echo "Generating Python protobuf code for development..."
	@mkdir -p src/lol_tracker/proto/events src/lol_tracker/proto/models
	@if [ -d "../api/proto" ]; then \
		./venv/bin/python -m grpc_tools.protoc -I=../api/proto \
			--python_out=src/lol_tracker/proto \
			--grpc_python_out=src/lol_tracker/proto \
			../api/proto/events/*.proto ../api/proto/models/*.proto; \
		echo "Generated protobuf files for local development"; \
	else \
		echo "Error: ../api/proto not found. Proto source files are required."; \
		exit 1; \
	fi

# Development
dev-local: ## Run the service locally in development mode
	@if [ ! -f ../.env ]; then \
		echo "Warning: .env file not found at project root. Copy .env.example to .env and configure."; \
	fi
	@if [ -f ../.env ]; then \
		set -a; source ../.env; set +a; PYTHONPATH=src ./venv/bin/python -m lol_tracker.main; \
	else \
		PYTHONPATH=src ./venv/bin/python -m lol_tracker.main; \
	fi

dev: ## Start development environment (use root 'make dev' instead)
	@echo "Please use 'make dev' from the project root directory instead"
	@exit 1

run: dev-local ## Alias for dev-local

# Docker
docker-build: ## Build Docker image
	docker build -t lol-tracker:latest .

docker-run: ## Run Docker container
	docker run --rm -it \
		--env-file ../.env \
		-p 8080:8080 \
		lol-tracker:latest

prod: ## Build and run production container with docker-compose
	docker-compose up --build

# Utilities
clean: ## Clean build artifacts and cache
	rm -rf build/
	rm -rf dist/
	rm -rf *.egg-info/
	rm -rf .pytest_cache/
	rm -rf .coverage
	rm -rf htmlcov/
	find . -type d -name __pycache__ -exec rm -rf {} +
	find . -type f -name "*.pyc" -delete

clean-venv: ## Remove virtual environment
	rm -rf venv/

# Build target for CI/CD
build: install-prod proto lint test ## Build the service (install, generate proto, lint, test)

