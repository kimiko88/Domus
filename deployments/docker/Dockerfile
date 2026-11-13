# syntax=docker/dockerfile:1

# ─── Stage 1: Build Web Frontend Assets ───────────────────────────────────────
FROM --platform=$BUILDPLATFORM node:24-alpine AS web-builder

WORKDIR /web
COPY web/package.json web/package-lock.json ./
RUN npm ci --ignore-scripts
COPY web/ ./
RUN npm run build

# ─── Stage 2: Build Pure Go Binaries ──────────────────────────────────────────
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS builder

WORKDIR /src

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Cache Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Target platform arguments passed by Docker buildx
ARG TARGETOS
ARG TARGETARCH
ARG VERSION=1.0.0
ARG BUILD_TIME

# Compile statically linked binaries (pure Go, zero CGO for cross-architecture portability)
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-arm64} \
    go build -trimpath -ldflags="-s -w -X 'main.Version=${VERSION}' -X 'main.BuildTime=${BUILD_TIME}'" \
    -o /bin/domus ./cmd/domus

RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-arm64} \
    go build -trimpath -ldflags="-s -w -X 'main.Version=${VERSION}'" \
    -o /bin/domus-cli ./cmd/domus-cli

# ─── Stage 3: Minimal Distroless / Hardened Runtime ───────────────────────────
FROM alpine:3.23 AS runtime

RUN apk add --no-cache ca-certificates tzdata poppler-utils tesseract-ocr tesseract-ocr-data-ita tesseract-ocr-data-eng \
    && addgroup -g 1000 domus \
    && adduser -u 1000 -G domus -s /bin/sh -D domus \
    && mkdir -p /data /etc/domus /usr/local/share/domus/web \
    && chown -R domus:domus /data /etc/domus /usr/local/share/domus

COPY --from=builder /bin/domus /usr/local/bin/domus
COPY --from=builder /bin/domus-cli /usr/local/bin/domus-cli
COPY --from=web-builder /web/dist /usr/local/share/domus/web
COPY --from=builder /src/domus.toml.example /etc/domus/domus.toml

USER domus
WORKDIR /data
VOLUME ["/data"]

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -qO- http://localhost:8080/health || exit 1

ENTRYPOINT ["/usr/local/bin/domus"]
CMD ["-config", "/etc/domus/domus.toml"]
