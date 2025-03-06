# Default target runs the server
default: run

# Run the server using the rootCmd
run:
	go run cmd/http-server/main.go

# Build the binary
build:
	go build -o cmd/build/wealth-warden cmd/http-server/main.go

# Run database migrations
migrate:
	go run cmd/http-server/main.go migrate $(type)

# Seed essential tables
seed:
	go run cmd/http-server/main.go seed $(type)

# Clean up binaries
clean:
	rm -f ./cmd/build/wealth-warden
