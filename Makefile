.PHONY: default run migrate seed loadtest mock build test test-unit test-integration test-coverage check-test-names lint lint-fix docker-up docker-down docker-restart docker-migrate docker-seed docker-arm-up docker-arm-down docker-arm-restart docker-arm-migrate docker-arm-seed tidy pre-push observe observe-local

COMPOSE_OBS       := -f ./docker-compose.observability.yaml
COMPOSE_OBS_LOCAL := -f ./docker-compose.observability.local.yaml
COMPOSE_MAIN      := -f ./docker-compose.yaml
COMPOSE_BUILD     := -f ./docker-compose.build.yaml
COMPOSE_ARM       := -f ./docker-compose.arm.yaml
COMPOSE_ARM_OBS   := -f ./docker-compose.arm.observability.yaml

# Docker targets pull published images by default; pass build=1 to build from source instead
BUILD              = $(if $(build),$(COMPOSE_BUILD))
FETCH              = $(if $(build),--build,--pull always)

# Default target runs the app
default: run

run:
	docker compose $(COMPOSE_OBS) $(COMPOSE_OBS_LOCAL) up -d
	go run ./cmd app

migrate:
	go run ./cmd migrate $(type)

seed:
	go run ./cmd seed $(type) $(name) $(if $(users),--users $(users)) $(if $(lean),--lean)

# Runs a k6 script on throwaway Postgres + Redis; test=ledger|dashboard|stress, serve=1 only starts the server
loadtest:
	go run -tags loadtest ./cmd loadtest $(test) $(if $(port),--port $(port)) $(if $(users),--users $(users)) $(if $(serve),--serve)

mock:
	mockery --config=.mockery.yaml

build:
	@echo "Building binary ..."
	go build -o build/wealthwarden ./cmd
	@echo "Binary available at ./build"

bench:
	go test -bench=. -run=Benchmark -timeout=5s ./...

test:
	go test -race -count=1 -tags integration ./...

# cover=<file> writes a coverage profile
test-unit:
	go test -race -count=1 $(if $(cover),-coverpkg=./... -coverprofile=$(cover)) ./...

test-integration:
	go test -race -count=1 -tags integration -run '^TestIntegration' $(if $(cover),-coverpkg=./... -coverprofile=$(cover)) ./...

check-test-names:
	@bad=$$(git grep -l '^//go:build integration' -- '*_test.go' | xargs -r grep -HnE '^func Test' | grep -vE ':func Test(Integration|Main\()'); \
	if [ -n "$$bad" ]; then echo "$$bad"; echo "tests in integration-tagged files must start with TestIntegration"; exit 1; fi

test-coverage:
	go test -race -tags integration -coverpkg=./... -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

lint:
	gofmt -l -e . | grep . && exit 1 || true
	golangci-lint run

lint-fix:
	gofmt -w .
	golangci-lint run --fix

observe:
	docker compose $(COMPOSE_OBS) $(COMPOSE_OBS_LOCAL) up -d

docker-up:
	docker compose $(COMPOSE_OBS) $(COMPOSE_MAIN) $(BUILD) up -d $(FETCH)

docker-down:
	docker compose $(COMPOSE_OBS) $(COMPOSE_MAIN) down

docker-restart:
	docker compose $(COMPOSE_OBS) $(COMPOSE_MAIN) restart

docker-migrate:
	docker compose $(COMPOSE_MAIN) $(BUILD) run --rm $(FETCH) migrate migrate $(or $(type),up)

docker-seed:
	docker compose $(COMPOSE_MAIN) $(BUILD) run --rm $(FETCH) migrate seed $(type) $(name) $(if $(users),--users $(users)) $(if $(lean),--lean)

docker-arm-up:
	docker compose $(COMPOSE_OBS) $(COMPOSE_ARM_OBS) $(COMPOSE_MAIN) $(COMPOSE_ARM) $(BUILD) up -d $(FETCH)

docker-arm-down:
	docker compose $(COMPOSE_OBS) $(COMPOSE_ARM_OBS) $(COMPOSE_MAIN) $(COMPOSE_ARM) down

docker-arm-restart:
	docker compose $(COMPOSE_OBS) $(COMPOSE_ARM_OBS) $(COMPOSE_MAIN) $(COMPOSE_ARM) restart

docker-arm-migrate:
	docker compose $(COMPOSE_MAIN) $(COMPOSE_ARM) $(BUILD) run --rm $(FETCH) migrate migrate $(or $(type),up)

docker-arm-seed:
	docker compose $(COMPOSE_MAIN) $(COMPOSE_ARM) $(BUILD) run --rm $(FETCH) migrate seed $(type) $(name) $(if $(users),--users $(users)) $(if $(lean),--lean)

tidy:
	go mod tidy
	go mod verify

pre-push:
	@echo "--- Bootstrap ---"
	mockery --config=.mockery.yaml
	@echo "--- App ---"
	@go build ./... && echo "build successful" || (echo "build failed" && exit 1)
	@golangci-lint run && echo "lint successful" || (echo "lint failed" && exit 1)
	@go test -race -count=1 -tags integration ./... && echo "tests successful" || (echo "tests failed" && exit 1)
	@echo ""
	@echo "--- Client ---"
	@cd client && pnpm run build && echo "build successful" || (echo "build failed" && exit 1)
	@cd client && pnpm run format && echo "format successful" || (echo "format failed" && exit 1)
	@cd client && pnpm run lint && echo "lint successful" || (echo "lint failed" && exit 1)
	@cd client && pnpm run test && echo "tests successful" || (echo "tests failed" && exit 1)