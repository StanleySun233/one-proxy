FROM node:22-bookworm-slim AS web-builder
WORKDIR /workspace/apps/one-proxy-panel
COPY apps/one-proxy-panel/package.json ./
RUN npm install
COPY apps/one-proxy-panel ./
RUN npm run build

FROM golang:1.23-bookworm AS api-builder
WORKDIR /workspace/apps/one-panel-api
COPY apps/one-panel-api/go.mod apps/one-panel-api/go.sum ./
RUN go mod download
COPY apps/one-panel-api ./
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/one-proxy-panel ./cmd/one-proxy-panel

FROM node:22-bookworm-slim
WORKDIR /app
RUN apt-get update && apt-get install -y --no-install-recommends tzdata && rm -rf /var/lib/apt/lists/*
ENV NODE_ENV=production
ENV TZ=Asia/Shanghai
ENV PORT=2886
ENV HTTP_ADDR=127.0.0.1:2887
ENV CONTROL_PLANE_URL=http://127.0.0.1:2887

COPY --from=api-builder /out/one-proxy-panel /app/bin/one-proxy-panel
COPY --from=api-builder /workspace/apps/one-panel-api/schema /app/apps/one-panel-api/schema
COPY --from=web-builder /workspace/apps/one-proxy-panel/.next/standalone /app
COPY --from=web-builder /workspace/apps/one-proxy-panel/.next/static /app/.next/static
COPY --from=web-builder /workspace/apps/one-proxy-panel/public /app/public
COPY docker/one-proxy-panel-start.sh /app/one-proxy-panel-start.sh

RUN mkdir -p /app/data && chmod +x /app/one-proxy-panel-start.sh

EXPOSE 2886

CMD ["/app/one-proxy-panel-start.sh"]
