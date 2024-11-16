FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/birdgpt ./cmd/main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/birdgpt .
VOLUME ["/app/config.yaml", "/app/credentials.json"]
CMD ["/app/birdgpt"] 