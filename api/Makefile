.PHONY: generate clean install-tools help

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Install required tools
install-tools: ## Install protobuf compilation tools
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@if [ ! -d "../venv" ]; then \
		echo "Creating Python virtual environment at project root..."; \
		cd .. && python3 -m venv venv; \
	fi
	@echo "Installing Python protobuf tools..."
	../venv/bin/pip install grpcio-tools

# Generate code for all languages
generate: generate-go ## Generate protobuf code for all languages
	@if [ -d "../venv" ] && [ -f "../venv/bin/python" ]; then \
		$(MAKE) generate-python; \
	else \
		echo "Python virtual environment not found. Run 'make install-tools' first."; \
	fi

generate-go: ## Generate Go protobuf code
	@echo "Generating Go protobuf code..."
	@mkdir -p gen/go
	protoc -I=proto \
		--go_out=gen/go --go_opt=paths=source_relative \
		--go-grpc_out=gen/go --go-grpc_opt=paths=source_relative \
		proto/events/*.proto proto/models/*.proto proto/services/*.proto

generate-python: ## Generate Python protobuf code
	@echo "Generating Python protobuf code..."
	@mkdir -p gen/python
	../venv/bin/python -m grpc_tools.protoc -I=proto \
		--python_out=gen/python \
		--grpc_python_out=gen/python \
		proto/events/*.proto proto/models/*.proto proto/services/*.proto

clean: ## Clean generated files
	rm -rf gen/

# Add to .gitignore
gitignore: ## Add gen/ to .gitignore
	@if [ ! -f .gitignore ] || ! grep -q "^gen/" .gitignore; then \
		echo "gen/" >> .gitignore; \
		echo "Added gen/ to .gitignore"; \
	else \
		echo "gen/ already in .gitignore"; \
	fi