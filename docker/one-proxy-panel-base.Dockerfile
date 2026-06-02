FROM golang:1.23-bookworm AS go-source

FROM node:22-bookworm-slim
WORKDIR /base

COPY --from=go-source /usr/local/go /usr/local/go
ENV PATH=/usr/local/go/bin:$PATH

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates git tzdata && rm -rf /var/lib/apt/lists/*

WORKDIR /base/apps/one-proxy-panel
COPY apps/one-proxy-panel/package.json ./
RUN npm install

WORKDIR /base/apps/one-panel-api
COPY apps/one-panel-api/go.mod apps/one-panel-api/go.sum ./
RUN go mod download
