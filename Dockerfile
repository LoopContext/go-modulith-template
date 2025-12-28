# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Build argument for target binary (default to server)
ARG TARGET=server

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/service ./cmd/${TARGET}/main.go

# Run stage
FROM alpine:3.20

# Install runtime dependencies (ca-certificates for HTTPS/gRPC TLS)
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/bin/service ./service

# Copy configurations
COPY configs/ ./configs/

# Expose ports (standard defaults, can be overridden)
EXPOSE 8080 9050

# Run the service
CMD ["./service"]
