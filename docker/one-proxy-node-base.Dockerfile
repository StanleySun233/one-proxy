FROM golang:1.23-bookworm
WORKDIR /base/apps/one-proxy-node

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates tzdata && rm -rf /var/lib/apt/lists/*

COPY apps/one-proxy-node/go.mod apps/one-proxy-node/go.sum ./
RUN go mod download

RUN mkdir -p /base/runtime /base/zoneinfo && cp -a /usr/share/zoneinfo/. /base/zoneinfo/
