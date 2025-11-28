# Single Dockerfile for api and worker
# Builds both binaries in one image

############################
# Stage 1: Builder
############################
FROM golang:1.25.0-alpine AS builder
WORKDIR /app

# Install git for go modules
RUN apk add --no-cache git

# Copy modules and download deps
COPY go.mod go.sum ./
RUN go mod download

# Copy all source code
COPY . .

# Build binaries
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/worker ./cmd/worker

############################
# Stage 2: Final image
############################
FROM alpine:3.19
WORKDIR /app

# Runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create storage dirs
RUN mkdir -p /app/storage/original /app/storage/processed

# Copy binaries from builder
COPY --from=builder /app/api /app/api
COPY --from=builder /app/worker /app/worker

# Copy configs, static files, migrations
COPY --from=builder /app/config.yaml /app/config.yaml
COPY --from=builder /app/migrations /app/migrations
COPY --from=builder /app/static /app/static

# Add unprivileged user
RUN adduser -D -s /bin/sh appuser
RUN chown -R appuser:appuser /app
USER appuser

EXPOSE 8080 9090 

ARG APP=api
ENV APP=${APP}

# Default command (api). We support switching to worker by building with --build-arg APP=worker
# or by overriding the command in docker-compose (recommended for quick tests).
CMD ["/bin/sh", "-c", "/app/${APP}"]
