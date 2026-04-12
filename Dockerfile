# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/scuffinger .

# Runtime stage
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/scuffinger .
COPY config/config.yaml /app/config/config.yaml

EXPOSE 8080

ENTRYPOINT ["/app/scuffinger"]
CMD ["serve", "--config", "/app/config/config.yaml"]

