# Stage 1: Build the webhook-server Go binary
FROM golang:1.26-bookworm AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/webhook-server ./cmd/webhook-server

# Stage 2: Minimal distroless runtime
FROM gcr.io/distroless/static-debian12

COPY --from=builder /out/webhook-server /webhook-server

EXPOSE 8080

ENTRYPOINT ["/webhook-server"]
