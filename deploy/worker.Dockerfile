# Stage 1: Build the worker Go binary
FROM golang:1.26-bookworm AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/worker ./cmd/worker

# Stage 2: Runtime — Node.js image with buildah, fuse-overlayfs, git, and Nx CLI
FROM node:20-bookworm-slim

# Install buildah, fuse-overlayfs, and git
RUN apt-get update && apt-get install -y --no-install-recommends \
    buildah \
    fuse-overlayfs \
    git \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Install Nx CLI globally
RUN npm install -g nx --prefer-offline 2>/dev/null || npm install -g nx

# Copy compiled worker binary from builder stage
COPY --from=builder /out/worker /usr/local/bin/worker

# buildah requires /etc/containers/storage.conf; create minimal default
RUN mkdir -p /etc/containers && \
    echo '[storage]' > /etc/containers/storage.conf && \
    echo 'driver = "overlay"' >> /etc/containers/storage.conf

ENTRYPOINT ["/usr/local/bin/worker"]
