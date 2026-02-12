.PHONY: all build test lint dev clean docker-build docker-push helm-package helm-install helm-template build-agent-heartbeat docker-build-agent-heartbeat docker-build-agent helm-package-agent

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
	docker build -t $(REGISTRY)/frontend:$(VERSION) frontend/

docker-build-api:
	docker build -t $(REGISTRY)/api:$(VERSION) api/

docker-build-operator:
	docker build -t $(REGISTRY)/operator:$(VERSION) operator/

docker-build-migration-cli:
	docker build -t $(REGISTRY)/migration:$(VERSION) migration-cli/

docker-build-agent-heartbeat:
	docker build -t $(REGISTRY)/agent-heartbeat:$(VERSION) -f agent/Dockerfile.heartbeat agent/

docker-build-agent: docker-build-agent-heartbeat

docker-push:
	docker push $(REGISTRY)/frontend:$(VERSION)
	docker push $(REGISTRY)/api:$(VERSION)
	docker push $(REGISTRY)/operator:$(VERSION)
	docker push $(REGISTRY)/migration:$(VERSION)

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
	cd frontend && pnpm test

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
