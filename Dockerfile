# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git build-base

# Set working directory
WORKDIR /app

# Copy and download Go dependencies first (for better caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy the Go source code
COPY cmd/ ./cmd/
COPY internal/ ./internal/

# Build the Go application
RUN go build -o bin/main ./cmd/api

# Run stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates

# Set working directory
WORKDIR /app

# Copy the Go binary from the builder stage
COPY --from=builder /app/bin/main ./

# Set environment variables
ENV PORT=8080

# Expose port 8080 (fly.io standard)
EXPOSE 8080

# Run the Go API
CMD ["./main"]
