FROM golang:1.23-bookworm AS go-source

FROM node:26.2.0-bookworm-slim
WORKDIR /base

COPY --from=go-source /usr/local/go /usr/local/go
ENV PATH=/usr/local/go/bin:$PATH

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates git tzdata && rm -rf /var/lib/apt/lists/*

WORKDIR /base/apps/panel/web
COPY apps/panel/web/package.json apps/panel/web/package-lock.json ./
RUN npm ci

WORKDIR /base/apps/panel/api
COPY apps/panel/api/go.mod apps/panel/api/go.sum ./
RUN go mod download
