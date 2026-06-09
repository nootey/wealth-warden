.PHONY: default run migrate seed mock build test test-coverage lint lint-fix docker-up docker-down docker-migrate docker-rpi-up docker-rpi-down docker-rpi-migrate tidy pre-push

COMPOSE_OBS     := -f ./deployments/docker/docker-compose.observability.yaml
COMPOSE_MAIN    := -f ./deployments/docker/docker-compose.yaml
COMPOSE_RPI     := -f ./deployments/docker/docker-compose.rpi.yaml
COMPOSE_RPI_OBS := -f ./deployments/docker/docker-compose.rpi.observability.yaml

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
	docker compose $(COMPOSE_OBS) $(COMPOSE_MAIN) -p wealth-warden up -d --build

docker-down:
	docker compose $(COMPOSE_OBS) $(COMPOSE_MAIN) -p wealth-warden down

docker-restart:
	docker compose $(COMPOSE_OBS) $(COMPOSE_MAIN) -p wealth-warden restart

docker-migrate:
	docker compose $(COMPOSE_MAIN) -p wealth-warden run --rm --build migrate migrate $(or $(type),up)

docker-rpi-up:
	docker compose $(COMPOSE_OBS) $(COMPOSE_RPI_OBS) $(COMPOSE_RPI) -p wealth-warden up -d --build

docker-rpi-down:
	docker compose $(COMPOSE_OBS) $(COMPOSE_RPI_OBS) $(COMPOSE_RPI) -p wealth-warden down

docker-rpi-restart:
	docker compose $(COMPOSE_OBS) $(COMPOSE_RPI_OBS) $(COMPOSE_RPI) -p wealth-warden restart

docker-rpi-migrate:
	docker compose $(COMPOSE_RPI) -p wealth-warden run --rm --build migrate migrate $(or $(type),up)

tidy:
	go mod tidy
	go mod verify

pre-push:
	@echo "--- Bootstrap ---"
	mockery --config=.mockery.yaml
	@echo "--- App ---"
	@go build ./... && echo "build successful" || (echo "build failed" && exit 1)
	@golangci-lint run && echo "lint successful" || (echo "lint failed" && exit 1)
	@go test -count=1 ./... && echo "tests successful" || (echo "tests failed" && exit 1)
	@echo ""
	@echo "--- Client ---"
	@cd client && pnpm run build && echo "build successful" || (echo "build failed" && exit 1)
	@cd client && pnpm run format && echo "format successful" || (echo "format failed" && exit 1)
	@cd client && pnpm run lint && echo "lint successful" || (echo "lint failed" && exit 1)
	@cd client && pnpm run test && echo "tests successful" || (echo "tests failed" && exit 1)