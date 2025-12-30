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
	go test -v -race -coverprofile=coverage.out -covermode=atomic ./...