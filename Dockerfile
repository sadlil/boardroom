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

WORKDIR /root/
COPY --from=builder /boardroom .

# Default configuration env bindings
ENV PORT=8080
ENV STORAGE_ROOT=/root/data

# Ensure empty data volume mountpoint exists
RUN mkdir -p /root/data

EXPOSE 8080
CMD ["./boardroom"]
