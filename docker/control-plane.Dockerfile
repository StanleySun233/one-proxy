FROM node:22-bookworm-slim AS web-builder
WORKDIR /workspace/apps/control-plane-web
COPY apps/control-plane-web/package.json ./
RUN npm install
COPY apps/control-plane-web ./
RUN npm run build

FROM golang:1.23-bookworm AS api-builder
WORKDIR /workspace/apps/control-plane-api
COPY apps/control-plane-api/go.mod apps/control-plane-api/go.sum ./
RUN go mod download
COPY apps/control-plane-api ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/control-plane ./cmd/control-plane

FROM node:22-bookworm-slim
WORKDIR /app
RUN apt-get update && apt-get install -y --no-install-recommends tzdata && rm -rf /var/lib/apt/lists/*
ENV NODE_ENV=production
ENV TZ=Asia/Shanghai
ENV PORT=2886
ENV HTTP_ADDR=127.0.0.1:2887
ENV CONTROL_PLANE_URL=http://127.0.0.1:2887

COPY --from=api-builder /out/control-plane /app/bin/control-plane
COPY --from=api-builder /workspace/apps/control-plane-api/schema /app/apps/control-plane-api/schema
COPY --from=web-builder /workspace/apps/control-plane-web/.next/standalone /app
COPY --from=web-builder /workspace/apps/control-plane-web/.next/static /app/.next/static
COPY --from=web-builder /workspace/apps/control-plane-web/public /app/public
COPY docker/control-plane-start.sh /app/control-plane-start.sh

RUN chmod +x /app/control-plane-start.sh

EXPOSE 2886

CMD ["/app/control-plane-start.sh"]
