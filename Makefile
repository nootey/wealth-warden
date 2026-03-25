.PHONY: default run migrate seed mock build test test-coverage lint lint-fix docker-up docker-down docker-rpi-up docker-rpi-down tidy pre-push

# Default target runs the app
default: run

run:
	go run ./cmd app

migrate:
	go run ./cmd migrate $(type)

seed:
	go run ./cmd seed $(type) $(name)

mock:
	mockery --config=.mockery.yaml

build:
	@echo "Building binary ..."
	go build -o build/wealthwarden ./cmd
	@echo "Binary available at ./build"

bench:
	go test -bench=. -run=Benchmark -timeout=5s ./...

test:
	go test -count=1 ./...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

lint:
	gofmt -l -e . | grep . && exit 1 || true
	golangci-lint run

lint-fix:
	gofmt -w .
	golangci-lint run --fix

docker-up:
	docker compose -f ./deployments/docker/docker-compose.yaml -p wealth-warden up -d --build

docker-down:
	docker compose -f ./deployments/docker/docker-compose.yaml -p wealth-warden down

docker-rpi-up:
	docker compose -f ./deployments/docker/docker-compose.rpi.yaml -p wealth-warden up -d --build

docker-rpi-down:
	docker compose -f ./deployments/docker/docker-compose.rpi.yaml -p wealth-warden down

tidy:
	go mod tidy
	go mod verify

pre-push:
	@echo "--- App ---"
	@go build ./... && echo "build successful" || (echo "build failed" && exit 1)
	@golangci-lint run && echo "lint successful" || (echo "lint failed" && exit 1)
	@go test -count=1 ./... && echo "test successful" || (echo "test failed" && exit 1)
	@echo ""
	@echo "--- Client ---"
	@cd client && npm run build && echo "build successful" || (echo "build failed" && exit 1)
	@cd client && npm run format && echo "format successful" || (echo "format failed" && exit 1)
	@cd client && npm run lint && echo "lint successful" || (echo "lint failed" && exit 1)