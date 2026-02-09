# NGF Console — Claude Code Conventions

## Project Overview

NGF Console is a web-based management platform for NGINX Gateway Fabric with native Kubernetes Gateway Inference Extensions support. Monorepo with four components: frontend, API, controller, migration CLI.

## Tech Stack

- **Frontend**: React 18 + TypeScript + Vite + Tailwind CSS + shadcn/ui
- **Backend API**: Go + Chi router + gorilla/websocket + client-go
- **Controller**: Go + controller-runtime
- **Migration CLI**: Go + cobra
- **Analytics DB**: ClickHouse
- **Config DB**: PostgreSQL (prod) / SQLite (dev)
- **Telemetry**: OpenTelemetry Collector

## Build Commands

```bash
make dev              # Start all services in dev mode
make build            # Build all components
make test             # Run all tests
make lint             # Run all linters
make docker-build     # Build all Docker images
```

### Frontend
```bash
cd frontend && pnpm install && pnpm dev    # Dev server
cd frontend && pnpm build                   # Production build
cd frontend && pnpm lint                    # ESLint
cd frontend && pnpm test                    # Vitest
```

### Go (API / Controller / Migration CLI)
```bash
cd api && go build ./...                    # Build API
cd api && go test ./...                     # Test API
cd controller && go build ./...             # Build controller
cd migration-cli && go build ./...          # Build migration CLI
golangci-lint run ./...                     # Lint Go code
```

## Coding Standards

### Go
- Use `internal/` for private packages, `pkg/` for shared public types
- Handler functions follow `func (h *Handler) MethodName(w http.ResponseWriter, r *http.Request)`
- Use structured logging (slog)
- Error wrapping with `fmt.Errorf("context: %w", err)`
- Group imports: stdlib, external, internal

### TypeScript / React
- Functional components only
- Use TypeScript strict mode
- State management via Zustand stores
- Data fetching via TanStack Query (React Query)
- Forms via React Hook Form + Zod validation
- All API types in `src/types/`
- Component files: PascalCase (e.g., `GatewayList.tsx`)

### General
- No hardcoded secrets or credentials
- F5 license files (.crt, .jwt, .key) are gitignored
- Enterprise features gated by edition detection (greyed out in OSS mode)
- Dark mode as default theme
