# ---- Build Stage ----
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /src

ARG TARGETOS
ARG TARGETARCH

# Cache dependency downloads
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build API binary
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} go build \
    -ldflags="-s -w" \
    -o /out/api ./cmd/api

# Build Worker binary
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} go build \
    -ldflags="-s -w" \
    -o /out/worker ./cmd/worker

# ---- API Runtime ----
FROM alpine:3.20 AS api

RUN apk add --no-cache ca-certificates tzdata curl
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

COPY --from=builder /out/api /usr/local/bin/api

USER appuser
EXPOSE 8080

HEALTHCHECK --interval=15s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/healthz || exit 1

ENTRYPOINT ["api"]

# ---- Worker Runtime ----
FROM alpine:3.20 AS worker

RUN apk add --no-cache ca-certificates tzdata
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

COPY --from=builder /out/worker /usr/local/bin/worker

USER appuser

ENTRYPOINT ["worker"]
