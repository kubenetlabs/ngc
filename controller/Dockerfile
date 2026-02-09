FROM golang:1.23-alpine AS builder

WORKDIR /workspace

# Copy go module files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the controller binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o controller main.go

# Runtime stage
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /

COPY --from=builder /workspace/controller /controller

USER 65532:65532

ENTRYPOINT ["/controller"]
