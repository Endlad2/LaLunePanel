# ---------- build stage ----------
FROM golang:1.24-bookworm AS builder

WORKDIR /src

# cache deps first
COPY backend/go.mod backend/go.sum* ./
RUN go mod download || true

COPY backend/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/lalune-panel ./cmd/server

# ---------- runtime stage ----------
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
        ca-certificates \
        curl \
        wget \
        tar \
        iproute2 \
        procps \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /out/lalune-panel /app/lalune-panel
COPY frontend/ /app/frontend/
COPY protocols/ /app/protocols/

# where downloaded protocol binaries + client configs live
RUN mkdir -p /data/bin /data/clients

ENV LALUNE_DATA_DIR=/data \
    LALUNE_FRONTEND_DIR=/app/frontend \
    LALUNE_PROTOCOLS_DIR=/app/protocols \
    LALUNE_LISTEN=:6333

EXPOSE 6333

CMD ["/app/lalune-panel"]
