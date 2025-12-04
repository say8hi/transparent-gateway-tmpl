.PHONY: help build run test lint lint-fix clean fmt fmt-check fmt-fix vet rename install-hooks

# variables
BINARY_NAME=api-gateway
BINARY_DIR=bin
CMD_DIR=cmd/api

# default target
.DEFAULT_GOAL := help

# rename project imports based on go.mod
# Usage: make rename MODULE=github.com/yourcompany/your-gateway
rename:
	@if [ -z "$(MODULE)" ]; then \
		echo "Error: MODULE is required"; \
		echo "Usage: make rename MODULE=github.com/yourcompany/your-gateway"; \
		exit 1; \
	fi
	@./rename-project.sh $(MODULE)

help:
	@echo "Available targets:"
	@echo "  rename MODULE=<name> - rename project imports (e.g. make rename MODULE=github.com/me/proj)"
	@echo "  install-hooks  - install git pre-commit hooks"
	@echo "  build          - build the application binary"
	@echo "  run            - run the application"
	@echo "  test           - run tests with coverage"
	@echo "  lint           - run linter (golangci-lint)"
	@echo "  lint-fix       - run linter and auto-fix issues"
	@echo "  fmt            - format code (gofumpt or go fmt)"
	@echo "  fmt-check      - check if code is formatted"
	@echo "  fmt-fix        - format and organize imports (gofumpt + goimports)"
	@echo "  vet            - run go vet"
	@echo "  clean          - remove build artifacts"

# install git hooks
install-hooks:
	@echo "Installing git hooks..."
	@git config core.hooksPath .githooks
	@echo "✓ Git hooks installed successfully!"
	@echo "Hooks will run automatically before each commit"

# build the application
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BINARY_DIR)
	@go build -o $(BINARY_DIR)/$(BINARY_NAME) ./$(CMD_DIR)
	@echo "Build complete: $(BINARY_DIR)/$(BINARY_NAME)"

# run the application
run:
	@echo "Running $(BINARY_NAME)..."
	@go run ./$(CMD_DIR)

# run tests with coverage
test:
	@echo "Running tests..."
	@go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...
	@echo "Coverage report generated: coverage.txt"

# run linter (requires golangci-lint)
lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed. Install it from https://golangci-lint.run/usage/install/"; \
		exit 1; \
	fi

# run linter with auto-fix
lint-fix:
	@echo "Running linter with auto-fix..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --fix ./...; \
	else \
		echo "golangci-lint not installed. Install it from https://golangci-lint.run/usage/install/"; \
		exit 1; \
	fi

# format code (use gofumpt if available, otherwise go fmt)
fmt:
	@if command -v gofumpt >/dev/null 2>&1; then \
		echo "Formatting code with gofumpt..."; \
		gofumpt -l -w .; \
	else \
		echo "Formatting code with go fmt..."; \
		go fmt ./...; \
	fi

# check if code is formatted
fmt-check:
	@if command -v gofumpt >/dev/null 2>&1; then \
		echo "Checking formatting with gofumpt..."; \
		UNFORMATTED=$$(gofumpt -l .); \
		if [ -n "$$UNFORMATTED" ]; then \
			echo "✗ The following files are not formatted:"; \
			echo "$$UNFORMATTED"; \
			echo ""; \
			echo "Please run: make fmt"; \
			exit 1; \
		fi; \
		echo "✓ All files are properly formatted"; \
	else \
		echo "Checking formatting with gofmt..."; \
		UNFORMATTED=$$(gofmt -l .); \
		if [ -n "$$UNFORMATTED" ]; then \
			echo "✗ The following files are not formatted:"; \
			echo "$$UNFORMATTED"; \
			echo ""; \
			echo "Please run: make fmt"; \
			exit 1; \
		fi; \
		echo "✓ All files are properly formatted"; \
	fi

# format code and organize imports (full formatting)
fmt-fix:
	@echo "Running full formatting..."
	@if command -v gofumpt >/dev/null 2>&1; then \
		echo "→ Formatting with gofumpt..."; \
		gofumpt -l -w .; \
	else \
		echo "→ Formatting with go fmt..."; \
		go fmt ./...; \
	fi
	@if command -v goimports >/dev/null 2>&1; then \
		echo "→ Organizing imports with goimports..."; \
		goimports -w .; \
	fi
	@echo "✓ Formatting complete"

# run go vet
vet:
	@echo "Running go vet..."
	@go vet ./...

# clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BINARY_DIR)
	@rm -f coverage.txt coverage.html
	@echo "Clean complete"
