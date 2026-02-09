# Build stage
FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
ARG COMMIT=unknown
ARG DATE=unknown

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-s -w \
      -X github.com/kubenetlabs/ngc/api/pkg/version.Version=${VERSION} \
      -X github.com/kubenetlabs/ngc/api/pkg/version.Commit=${COMMIT} \
      -X github.com/kubenetlabs/ngc/api/pkg/version.Date=${DATE}" \
    -o /api-server ./cmd/server

# Runtime stage
FROM gcr.io/distroless/static-debian12

COPY --from=builder /api-server /api-server

EXPOSE 8080

ENTRYPOINT ["/api-server"]
