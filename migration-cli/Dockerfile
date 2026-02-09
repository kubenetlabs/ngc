FROM golang:1.23-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /ngf-migrate main.go

FROM gcr.io/distroless/static-debian12

COPY --from=build /ngf-migrate /ngf-migrate

ENTRYPOINT ["/ngf-migrate"]
