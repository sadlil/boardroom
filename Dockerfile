# Build Stage
FROM golang:alpine AS builder
WORKDIR /app

# SQLite needs CGO
RUN apk add --no-cache build-base

# Enable caching for go mod downloads
COPY go.mod go.sum* ./
RUN go mod download

# Copy source and build
COPY . .
RUN GOOS=linux GOARCH=amd64 go build -o /boardroom ./cmd/boardroom

# Runtime Stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata sqlite

# Create non-root user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app
COPY --from=builder /boardroom .

# Ensure empty data volume mountpoint exists and set permissions
RUN mkdir -p /app/data && chown -R appuser:appgroup /app

USER appuser

# Default configuration env bindings
ENV PORT=8080
ENV STORAGE_ROOT=/app/data

EXPOSE 8080
CMD ["./boardroom"]
