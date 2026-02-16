# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
COPY apps/worker-service/go.mod apps/worker-service/go.sum ./apps/worker-service/

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
WORKDIR /build/apps/worker-service
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /build/bin/worker-service .

# Runtime stage
FROM fedora:latest

# Install buildah and runtime dependencies
RUN dnf install -y \
    buildah \
    fuse-overlayfs \
    ca-certificates \
    git \
    && dnf clean all

# Configure buildah for rootless operation
RUN mkdir -p /etc/containers && \
    echo '[storage]' > /etc/containers/storage.conf && \
    echo 'driver = "overlay"' >> /etc/containers/storage.conf && \
    echo '[storage.options.overlay]' >> /etc/containers/storage.conf && \
    echo 'mount_program = "/usr/bin/fuse-overlayfs"' >> /etc/containers/storage.conf

# Create non-root user
RUN useradd -m -u 1000 -s /bin/bash appuser && \
    mkdir -p /home/appuser/.local/share/containers && \
    chown -R appuser:appuser /home/appuser

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/bin/worker-service /app/worker-service

# Copy configuration file
COPY apps/worker-service/config.yaml /app/config.yaml

# Create cache directories
RUN mkdir -p /var/cache/oci-build/repos /var/cache/oci-build/deps /var/log/oci-build && \
    chown -R appuser:appuser /var/cache/oci-build /var/log/oci-build

# Change ownership
RUN chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Set environment variables for buildah
ENV BUILDAH_ISOLATION=chroot \
    STORAGE_DRIVER=overlay

# Set entrypoint
ENTRYPOINT ["/app/worker-service"]

# Note: When running this container, mount the Docker socket if needed:
# docker run -v /var/run/docker.sock:/var/run/docker.sock worker-service
