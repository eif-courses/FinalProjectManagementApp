# ========================================
# Railway-optimized Go Dockerfile (No CSS build)
# ========================================

# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache \
    git \
    curl \
    make \
    gcc \
    musl-dev

WORKDIR /app

# Install templ CLI
RUN go install github.com/a-h/templ/cmd/templ@latest

# Copy go mod files first (better caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Generate templ files
RUN templ generate

# Build the Go application
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-w -s" -o main .

# ========================================
# Runtime stage
# ========================================
FROM alpine:3.20

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    curl

# Create app user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

WORKDIR /app

# Copy built application
COPY --from=builder /app/main .

# Copy static assets (CSS already built)
COPY --from=builder /app/assets ./assets
COPY --from=builder /app/static ./static
COPY --from=builder /app/locales ./locales
COPY --from=builder /app/migrations ./migrations
COPY --from=builder /app/.templui.json .

# Create necessary directories
RUN mkdir -p /app/uploads /app/tmp

# Change ownership
RUN chown -R appuser:appgroup /app

# Switch to app user
USER appuser

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD curl -f http://0.0.0.0:${PORT:-8080}/health || exit 1

# Run the application
CMD ["./main"]