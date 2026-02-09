.PHONY: all build test lint dev clean docker-build docker-push helm-package helm-install helm-template

VERSION ?= 0.1.0
REGISTRY ?= registry.f5.com/ngf-console
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

build: build-frontend build-api build-controller build-migration-cli

build-frontend:
	cd frontend && pnpm install && pnpm build

build-api:
	cd api && go build -o bin/api-server ./cmd/server

build-controller:
	cd controller && go build -o bin/controller .

build-migration-cli:
	cd migration-cli && go build -o bin/ngf-migrate .

# ──────────────────────────────────────────────
# Docker
# ──────────────────────────────────────────────

docker-build: docker-build-frontend docker-build-api docker-build-controller docker-build-migration-cli

docker-build-frontend:
	docker build -t $(REGISTRY)/frontend:$(VERSION) frontend/

docker-build-api:
	docker build -t $(REGISTRY)/api:$(VERSION) api/

docker-build-controller:
	docker build -t $(REGISTRY)/controller:$(VERSION) controller/

docker-build-migration-cli:
	docker build -t $(REGISTRY)/migration:$(VERSION) migration-cli/

docker-push:
	docker push $(REGISTRY)/frontend:$(VERSION)
	docker push $(REGISTRY)/api:$(VERSION)
	docker push $(REGISTRY)/controller:$(VERSION)
	docker push $(REGISTRY)/migration:$(VERSION)

# ──────────────────────────────────────────────
# Helm
# ──────────────────────────────────────────────

helm-package:
	helm package deploy/helm/ngf-console

helm-install:
	helm install ngf-console deploy/helm/ngf-console --namespace ngf-system --create-namespace

helm-template:
	helm template ngf-console deploy/helm/ngf-console

# ──────────────────────────────────────────────
# Test
# ──────────────────────────────────────────────

test: test-frontend test-api test-controller test-migration-cli

test-frontend:
	cd frontend && pnpm test

test-api:
	cd api && go test ./...

test-controller:
	cd controller && go test ./...

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
	cd controller && golangci-lint run ./...
	cd migration-cli && golangci-lint run ./...

# ──────────────────────────────────────────────
# Generate
# ──────────────────────────────────────────────

generate-manifests:
	helm template ngf-console deploy/helm/ngf-console > deploy/manifests/install.yaml

# ──────────────────────────────────────────────
# Clean
# ──────────────────────────────────────────────

clean:
	rm -rf frontend/dist frontend/node_modules
	rm -f api/bin/api-server controller/bin/controller migration-cli/bin/ngf-migrate
	$(DOCKER_COMPOSE) down -v
