.PHONY: help generate-reexports test test-coverage bench clean

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

generate-reexports: ## Generate re-export code for all adapters
	@echo "Generating re-exports for all adapters..."
	@go run tools/generate_reexports.go fiber > adapters/fiber/reexports_generated.go
	@go run tools/generate_reexports.go echo > adapters/echo/reexports_generated.go
	@go run tools/generate_reexports.go gin > adapters/gin/reexports_generated.go
	@go run tools/generate_reexports.go fibernative > adapters/fibernative/reexports_generated.go
	@echo "✅ Re-exports generated successfully!"

test: ## Run all tests
	@go test ./... -v

test-coverage: ## Run tests with coverage report
	@go test ./... -cover

test-coverage-html: ## Generate HTML coverage report
	@go test ./... -coverprofile=coverage.out
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

bench: ## Run benchmarks
	@go test ./... -bench=. -benchmem

bench-generator: ## Run generator benchmarks only
	@go test . -bench=Generator -benchmem

clean: ## Clean generated files and build artifacts
	@rm -f coverage.out coverage.html
	@find . -name "*.test" -delete
	@echo "✅ Cleaned up generated files"

fmt: ## Format code
	@go fmt ./...

vet: ## Run go vet
	@go vet ./...

lint: ## Run golangci-lint (requires golangci-lint installed)
	@golangci-lint run

check: fmt vet test ## Run format, vet, and tests

.DEFAULT_GOAL := help

