# Use the official Golang image as the base image
FROM --platform=$BUILDPLATFORM golang:1.23.0 AS base

# Build arguments provided by Docker Buildx
ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT

# Set Go environment variables dynamically
ENV GO111MODULE=on
ENV CGO_ENABLED=0
ENV GOOS=$TARGETOS
ENV GOARCH=$TARGETARCH

WORKDIR /src

# Copy Go modules
COPY go.* .

# Download dependencies
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Build the application
FROM base AS build

# Mount the source code and build caches
RUN --mount=target=. \
    --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -ldflags="-w -s" -o /app/service ./cmd/main.go

# Create the final image
FROM debian:stable-slim AS prd

# Install root certificates for TLS
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

# Copy the built binary
COPY --link --from=build /app/service /service

# Use non-root user
#USER 65532:65532

# Set the entrypoint
CMD ["/service"]
