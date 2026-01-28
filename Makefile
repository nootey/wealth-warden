# Default target runs the server
default: run

# Run the server using the rootCmd
run:
	go run ./cmd http

# Run database migrations
migrate:
	go run ./cmd migrate $(type)

# Seed essential tables
seed:
	go run ./cmd seed $(type)

# Perform first time setup.
bootstrap:
	@echo "Tidying Go modules and installing tools..."
	go mod tidy
	@echo "Bootstrap complete."

test:
	go test -v ./...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

lint:
	golangci-lint run

lint-fix:
	golangci-lint run --fix