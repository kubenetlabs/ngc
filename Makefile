.PHONY: all build test lint dev clean docker-build docker-push helm-package helm-install helm-template build-agent-heartbeat docker-build-agent-heartbeat docker-build-agent helm-package-agent ci test-coverage helm-lint

VERSION ?= 0.1.0
REGISTRY ?= danny2guns
PLATFORM ?= linux/amd64
DOCKER_COMPOSE := docker compose -f deploy/docker-compose/docker-compose.yaml

# ──────────────────────────────────────────────
# Development
# ──────────────────────────────────────────────

dev: dev-compose dev-api dev-frontend

dev-frontend:
	cd frontend && pnpm dev

dev-api:
	cd api && go run ./cmd/server

dev-compose:
	$(DOCKER_COMPOSE) up -d clickhouse otel-collector

# ──────────────────────────────────────────────
# Build
# ──────────────────────────────────────────────

build: build-frontend build-api build-operator build-migration-cli build-agent-heartbeat

build-frontend:
	cd frontend && pnpm install && pnpm build

build-api:
	cd api && go build -o bin/api-server ./cmd/server

build-operator:
	cd operator && go build -o bin/operator ./cmd/

build-migration-cli:
	cd migration-cli && go build -o bin/ngf-migrate .

build-agent-heartbeat:
	cd agent && go build -o bin/heartbeat ./cmd/heartbeat

# ──────────────────────────────────────────────
# Docker
# ──────────────────────────────────────────────

docker-build: docker-build-frontend docker-build-api docker-build-operator docker-build-migration-cli docker-build-agent

docker-build-frontend:
	docker build --platform $(PLATFORM) -t $(REGISTRY)/ngf-console-frontend:$(VERSION) frontend/

docker-build-api:
	docker build --platform $(PLATFORM) -t $(REGISTRY)/ngf-console-api:$(VERSION) api/

docker-build-operator:
	docker build --platform $(PLATFORM) -t $(REGISTRY)/ngf-console-operator:$(VERSION) operator/

docker-build-migration-cli:
	docker build --platform $(PLATFORM) -t $(REGISTRY)/ngf-console-migration:$(VERSION) migration-cli/

docker-build-agent-heartbeat:
	docker build --platform $(PLATFORM) -t $(REGISTRY)/ngf-console-agent-heartbeat:$(VERSION) -f agent/Dockerfile.heartbeat agent/

docker-build-agent: docker-build-agent-heartbeat

docker-push:
	docker push $(REGISTRY)/ngf-console-frontend:$(VERSION)
	docker push $(REGISTRY)/ngf-console-api:$(VERSION)
	docker push $(REGISTRY)/ngf-console-operator:$(VERSION)
	docker push $(REGISTRY)/ngf-console-migration:$(VERSION)
	docker push $(REGISTRY)/ngf-console-agent-heartbeat:$(VERSION)

# ──────────────────────────────────────────────
# Helm
# ──────────────────────────────────────────────

helm-package:
	helm package deploy/helm/ngf-console

helm-package-agent:
	helm package charts/ngf-console-agent

helm-install:
	helm install ngf-console deploy/helm/ngf-console --namespace ngf-system --create-namespace

helm-template:
	helm template ngf-console deploy/helm/ngf-console

# ──────────────────────────────────────────────
# Test
# ──────────────────────────────────────────────

test: test-frontend test-api test-operator test-migration-cli

test-frontend:
	cd frontend && pnpm test:run

test-api:
	cd api && go test ./...

test-operator:
	cd operator && go test ./...

test-migration-cli:
	cd migration-cli && go test ./...

# ──────────────────────────────────────────────
# Lint
# ──────────────────────────────────────────────

lint: lint-frontend lint-go

lint-frontend:
	cd frontend && pnpm lint

lint-go:
	cd api && golangci-lint run ./...
	cd operator && golangci-lint run ./...
	cd migration-cli && golangci-lint run ./...

# ──────────────────────────────────────────────
# CI
# ──────────────────────────────────────────────

ci: lint test build helm-lint

test-coverage: test-coverage-go test-coverage-frontend

test-coverage-go:
	cd api && go test -coverprofile=coverage.out ./...
	cd api && go tool cover -func=coverage.out
	cd operator && go test -coverprofile=coverage.out ./...
	cd operator && go tool cover -func=coverage.out

test-coverage-frontend:
	cd frontend && pnpm test:run --coverage

helm-lint:
	helm lint deploy/helm/ngf-console

# ──────────────────────────────────────────────
# Generate
# ──────────────────────────────────────────────

generate-manifests:
	helm template ngf-console deploy/helm/ngf-console > deploy/manifests/install.yaml

generate-crds:
	cp operator/config/crd/bases/*.yaml deploy/helm/ngf-console/templates/crds/

generate-deepcopy:
	cd operator && go generate ./...

# ──────────────────────────────────────────────
# Clean
# ──────────────────────────────────────────────

clean:
	rm -rf frontend/dist frontend/node_modules
	rm -f api/bin/api-server operator/bin/operator migration-cli/bin/ngf-migrate
	$(DOCKER_COMPOSE) down -v
