# Build stage: compile a static binary so the runtime image needs no toolchain.
FROM golang:1.26-alpine AS build

WORKDIR /src

# Dependencies are copied first so that a source-only change reuses this layer.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

# Runtime stage: distroless-style Alpine with a non-root user.
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata wget && \
    adduser -D -u 10001 appuser

WORKDIR /app

COPY --from=build /out/server /app/server
COPY migrations /app/migrations

# Uploads are written here under the local storage driver; mount a volume over
# it (or switch STORAGE_DRIVER) so files survive a container restart.
RUN mkdir -p /app/uploads && chown -R appuser:appuser /app

USER appuser
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=20s --retries=3 \
  CMD wget --spider -q http://127.0.0.1:8080/health || exit 1

ENTRYPOINT ["/app/server"]
