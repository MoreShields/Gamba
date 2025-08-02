.PHONY: help build test dev install lint format proto run

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Virtual environment configuration
VENV_DIR := $(shell if [ -d "../venv" ]; then echo "../venv"; else echo "./venv"; fi)
PYTHON := $(VENV_DIR)/bin/python
PIP := $(VENV_DIR)/bin/pip

# Development setup
install: ## Install dependencies in virtual environment
	@if [ ! -d "../venv" ]; then \
		echo "Creating local venv (parent venv not found)"; \
		python3 -m venv venv; \
		./venv/bin/pip install --upgrade pip; \
		./venv/bin/pip install -r requirements-dev.txt; \
	else \
		echo "Using parent project venv"; \
		$(PIP) install -r requirements-dev.txt; \
	fi

# Code quality
lint: ## Run linting checks
	$(PYTHON) -m flake8 lol_tracker/ tests/
	$(PYTHON) -m mypy lol_tracker/

format: ## Format code with black and isort
	$(PYTHON) -m black lol_tracker/ tests/
	$(PYTHON) -m isort lol_tracker/ tests/

format-check: ## Check if code is formatted correctly
	$(PYTHON) -m black --check lol_tracker/ tests/
	$(PYTHON) -m isort --check-only lol_tracker/ tests/

# Testing
test: ## Run all tests
	PYTHONPATH=. $(PYTHON) -m pytest tests/ -v

test-unit: ## Run unit tests only
	PYTHONPATH=. $(PYTHON) -m pytest tests/ -v -m "not integration"

test-integration: ## Run integration tests only
	PYTHONPATH=. $(PYTHON) -m pytest tests/ -v -m "integration"

test-cov: ## Run tests with coverage
	PYTHONPATH=. $(PYTHON) -m pytest tests/ --cov=lol_tracker --cov-report=html --cov-report=term

# Protocol Buffers
proto: ## Generate Python protobuf code locally for development
	@echo "Generating Python protobuf code for development..."
	@rm -rf lol_tracker/proto
	@mkdir -p lol_tracker/proto
	@if [ -d "../api/proto" ]; then \
		cd lol_tracker && ../$(PYTHON) -m grpc_tools.protoc \
			-I../../api/proto \
			--python_out=proto \
			--grpc_python_out=proto \
			--pyi_out=proto \
			../../api/proto/events/*.proto \
			../../api/proto/models/*.proto \
			../../api/proto/services/*.proto && \
		touch proto/__init__.py && \
		touch proto/events/__init__.py && \
		touch proto/models/__init__.py && \
		touch proto/services/__init__.py && \
		find proto -name "*.py" -type f -exec sed -i'' -e 's/^from services import/from . import/g' {} \; && \
		find proto -name "*.py" -type f -exec sed -i'' -e 's/^from events import/from . import/g' {} \; && \
		find proto -name "*.py" -type f -exec sed -i'' -e 's/^from models import/from . import/g' {} \; && \
		find proto -name "*-e" -type f -delete; \
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
		set -a; source ../.env; set +a; PYTHONPATH=. $(PYTHON) -m lol_tracker.main; \
	else \
		PYTHONPATH=. $(PYTHON) -m lol_tracker.main; \
	fi

dev: ## Start development environment (use root 'make dev' instead)
	@echo "Please use 'make dev' from the project root directory instead"
	@exit 1

run: dev-local ## Alias for dev-local

migrate-up-dev: ## Run pending database migrations (local development)
	@if [ -f ../.env ]; then \
		set -a; source ../.env; set +a; $(PYTHON) -m alembic upgrade head; \
	elif [ -f .env ]; then \
		set -a; source .env; set +a; $(PYTHON) -m alembic upgrade head; \
	else \
		echo "Error: .env file not found. Create .env in project root or lol-tracker/ directory."; exit 1; \
	fi

migrate-down-dev: ## Rollback last database migration (local development)
	@if [ -f ../.env ]; then \
		set -a; source ../.env; set +a; $(PYTHON) -m alembic downgrade -1; \
	elif [ -f .env ]; then \
		set -a; source .env; set +a; $(PYTHON) -m alembic downgrade -1; \
	else \
		echo "Error: .env file not found. Create .env in project root or lol-tracker/ directory."; exit 1; \
	fi

migrate-status-dev: ## Check migration status (local development)
	@if [ -f ../.env ]; then \
		set -a; source ../.env; set +a; $(PYTHON) -m alembic current; \
	elif [ -f .env ]; then \
		set -a; source .env; set +a; $(PYTHON) -m alembic current; \
	else \
		echo "Error: .env file not found. Create .env in project root or lol-tracker/ directory."; exit 1; \
	fi

