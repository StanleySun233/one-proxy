FROM golang:1.23-bookworm
WORKDIR /base/apps/one-proxy-node/api

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates tzdata && rm -rf /var/lib/apt/lists/*

COPY apps/one-proxy-node/api/go.mod apps/one-proxy-node/api/go.sum ./
RUN go mod download

RUN mkdir -p /base/runtime /base/web /base/zoneinfo && cp -a /usr/share/zoneinfo/. /base/zoneinfo/
