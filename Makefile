.PHONY: help build run test clean setup migrate docker-build docker-run

# Default target
help:
	@echo "Available targets:"
	@echo "  setup     - Set up development environment"
	@echo "  build     - Build the application"
	@echo "  run       - Run the application"
	@echo "  test      - Run tests"
	@echo "  migrate   - Run database migrations"
	@echo "  clean     - Clean build artifacts"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-run   - Run with Docker Compose"

# Set up development environment
setup:
	@echo "Setting up development environment..."
	go mod tidy
	@if [ ! -f "config/config.yaml" ]; then \
		echo "Creating config file from example..."; \
		cp config/config.yaml config/config.local.yaml 2>/dev/null || true; \
	fi
	@echo "Setup complete!"

# Build the application
build:
	@echo "Building application..."
	go build -o bin/payment-server cmd/server/main.go
	@echo "Build complete! Binary: bin/payment-server"

# Run the application
run: build
	@echo "Starting payment server..."
	./bin/payment-server

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	go clean

# Run database migrations (requires psql)
migrate:
	@echo "Running database migrations..."
	@if command -v psql >/dev/null 2>&1; then \
		psql -U postgres -d payments -f migrations/schema.sql; \
		echo "Migrations complete!"; \
	else \
		echo "Error: psql not found. Please install PostgreSQL client."; \
		exit 1; \
	fi

# Create database (requires psql)
create-db:
	@echo "Creating database..."
	@if command -v psql >/dev/null 2>&1; then \
		psql -U postgres -c "CREATE DATABASE payments;" 2>/dev/null || echo "Database may already exist"; \
		echo "Database created!"; \
	else \
		echo "Error: psql not found. Please install PostgreSQL client."; \
		exit 1; \
	fi

# Development setup with database
dev-setup: setup create-db migrate
	@echo "Development environment ready!"

# Run with hot reload (requires air: go install github.com/cosmtrek/air@latest)
dev:
	@if command -v air >/dev/null 2>&1; then \
		air; \
	else \
		echo "Installing air for hot reload..."; \
		go install github.com/cosmtrek/air@latest; \
		air; \
	fi

# Build Docker image
docker-build:
	@echo "Building Docker image..."
	docker build -t payment-proxy:latest .

# Run with Docker Compose
docker-run:
	@echo "Starting services with Docker Compose..."
	docker-compose up --build

# Stop Docker Compose services
docker-stop:
	docker-compose down

# Format code
fmt:
	go fmt ./...
	go vet ./...

# Generate documentation
docs:
	@if command -v swag >/dev/null 2>&1; then \
		swag init -g cmd/server/main.go -o docs/swagger; \
	else \
		echo "Installing swag for documentation generation..."; \
		go install github.com/swaggo/swag/cmd/swag@latest; \
		swag init -g cmd/server/main.go -o docs/swagger; \
	fi

# Lint code (requires golangci-lint)
lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "Installing golangci-lint..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
		golangci-lint run; \
	fi

# Create a test transaction
test-payment:
	@echo "Creating test payment..."
	curl -X POST http://localhost:8080/api/v1/payment/session \
		-H "Authorization: Bearer demo_api_key_12345" \
		-H "Content-Type: application/json" \
		-d '{"content_path": "/premium/article", "user_identifier": "test_user_123"}'

# Check API health
health:
	@echo "Checking API health..."
	curl -s http://localhost:8080/health | jq '.' || curl -s http://localhost:8080/health