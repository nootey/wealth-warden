.PHONY: default run migrate-up migrate-down migrate-status migrate-fresh seed seed-full seed-bulk seed-one loadtest mock build bench test test-unit test-integration test-coverage test-mutation check-test-names lint lint-fix observe docker-up docker-down docker-restart docker-migrate-up docker-migrate-down docker-migrate-status docker-migrate-fresh docker-seed docker-seed-full docker-seed-bulk docker-seed-one tidy pre-push

COMPOSE_OBS       := -f ./docker-compose.observability.yaml
COMPOSE_OBS_LOCAL := -f ./docker-compose.observability.local.yaml
COMPOSE_MAIN      := -f ./docker-compose.yaml
COMPOSE_BUILD     := -f ./docker-compose.build.yaml
COMPOSE_ARM       := -f ./docker-compose.arm.yaml
COMPOSE_ARM_OBS   := -f ./docker-compose.arm.observability.yaml
COMPOSE_PROXY     := -f ./docker-compose.proxy.yaml
COMPOSE_PROXY_OBS := -f ./docker-compose.proxy.observability.yaml

# Docker targets pull published images by default; pass build=1 to build from source instead
BUILD              = $(if $(build),$(COMPOSE_BUILD))
FETCH              = $(if $(build),--build,--pull always)

# Docker targets run the server stack by default; pass arm=1 for an ARM board
ARM                = $(if $(arm),$(COMPOSE_ARM))
ARM_OBS            = $(if $(arm),$(COMPOSE_ARM_OBS))

# Pass proxy=1 to serve through an external reverse proxy on PROXY_NETWORK instead of host ports
PROXY              = $(if $(proxy),$(COMPOSE_PROXY))
PROXY_OBS          = $(if $(proxy),$(COMPOSE_PROXY_OBS))
STACK              = $(COMPOSE_OBS) $(ARM_OBS) $(PROXY_OBS) $(COMPOSE_MAIN) $(ARM) $(PROXY)
DOCKER_RUN         = docker compose $(COMPOSE_MAIN) $(ARM) $(BUILD) run --rm $(FETCH) migrate

# Default target runs the app
default: run

run:
	docker compose $(COMPOSE_OBS) $(COMPOSE_OBS_LOCAL) up -d
	go run ./cmd app

migrate-up:
	go run ./cmd migrate up

migrate-down:
	go run ./cmd migrate down

migrate-status:
	go run ./cmd migrate status

# Drops the database, migrates and seeds; seed=full for dev data
migrate-fresh:
	go run ./cmd migrate fresh-seed-$(or $(seed),basic)

seed:
	go run ./cmd seed basic

seed-full:
	go run ./cmd seed full

seed-bulk:
	go run ./cmd seed bulk $(if $(users),--users $(users)) $(if $(lean),--lean)

seed-one:
	go run ./cmd seed individual $(name)

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

# Mutation tests per package, logs in logs/mutation; target=go|client, jobs=N, integration=1, only="utils ..."
test-mutation:
	./scripts/mutation-test.sh $(if $(target),--target $(target)) $(if $(jobs),--jobs $(jobs)) $(if $(integration),--integration) $(only)

lint:
	gofmt -l -e . | grep . && exit 1 || true
	golangci-lint run

lint-fix:
	gofmt -w .
	golangci-lint run --fix

observe:
	docker compose $(COMPOSE_OBS) $(COMPOSE_OBS_LOCAL) up -d

docker-up:
	docker compose $(STACK) $(BUILD) up -d $(FETCH)

docker-down:
	docker compose $(STACK) down

docker-restart:
	docker compose $(STACK) restart

docker-migrate-up:
	$(DOCKER_RUN) migrate up

docker-migrate-down:
	$(DOCKER_RUN) migrate down

docker-migrate-status:
	$(DOCKER_RUN) migrate status

docker-migrate-fresh:
	$(DOCKER_RUN) migrate fresh-seed-$(or $(seed),basic)

docker-seed:
	$(DOCKER_RUN) seed basic

docker-seed-full:
	$(DOCKER_RUN) seed full

docker-seed-bulk:
	$(DOCKER_RUN) seed bulk $(if $(users),--users $(users)) $(if $(lean),--lean)

docker-seed-one:
	$(DOCKER_RUN) seed individual $(name)

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