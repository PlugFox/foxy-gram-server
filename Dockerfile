# syntax = docker/dockerfile:1.4

# Stage 1: Build the application with CGO enabled for multiple architectures
FROM --platform=$BUILDPLATFORM golang:1.23.0-alpine AS build

# Set target OS and architecture dynamically
ARG TARGETOS
ARG TARGETARCH
ENV GO111MODULE=on
ENV CGO_ENABLED=1
ENV GOOS=$TARGETOS
ENV GOARCH=$TARGETARCH

# Install necessary build dependencies for CGO
RUN apk add --no-cache gcc musl-dev

WORKDIR /src

# Copy go.mod and go.sum to leverage Docker cache
COPY go.* .

# Cache Go module downloads
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy the rest of the application code and build the application for the target architecture
COPY . .

# Build the Go binary
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -ldflags="-w -s" -o /app/bot ./cmd/main.go

# Stage 2: Create the final image with Alpine
FROM --platform=$TARGETOS/$TARGETARCH alpine:latest

# Install SQLite runtime libraries
RUN apk add --no-cache sqlite-libs

# Copy the built binary from the build stage
COPY --from=build /app/bot /service

# Use a non-root user for security
USER nobody:nogroup

# Run the application
CMD ["/service"]
