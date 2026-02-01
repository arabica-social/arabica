# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files first for caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o arabica cmd/server/main.go

# Runtime stage
FROM alpine:3.23

WORKDIR /app

# Install ca-certificates for HTTPS calls
RUN apk add --no-cache ca-certificates

# Create non-root user
RUN adduser -D -u 1000 arabica

# Create data directory
RUN mkdir -p /data && chown arabica:arabica /data

# Copy binary from builder
COPY --from=builder /app/arabica .

# Copy static assets
COPY --chown=arabica:arabica static/ ./static/

# Note: Templ components are compiled into the binary, no need to copy .templ files

# Switch to non-root user
USER arabica

# Environment defaults
ENV PORT=18910
ENV ARABICA_DB_PATH=/data/arabica.db
ENV LOG_FORMAT=json

EXPOSE 18910

CMD ["./arabica"]
