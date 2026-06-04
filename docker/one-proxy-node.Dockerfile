ARG NODE_BASE_IMAGE=oneproxy-node-base:latest

FROM node:26-bookworm AS web-builder
WORKDIR /workspace/apps/node/web
COPY apps/node/web ./
RUN npm run build

FROM ${NODE_BASE_IMAGE} AS builder
WORKDIR /workspace/apps/node/api
COPY apps/node/api ./
COPY --from=web-builder /workspace/apps/node/web/dist /out/web
RUN mkdir -p /out/runtime /out/web /out/zoneinfo && cp -a /usr/share/zoneinfo/. /out/zoneinfo/ && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/one-proxy-node ./cmd/one-proxy-node

FROM ${NODE_BASE_IMAGE}
WORKDIR /app
COPY --from=builder /out/one-proxy-node /app/one-proxy-node
COPY --from=builder /out/runtime /app/runtime
COPY --from=builder /out/web /app/web
COPY --from=builder /out/zoneinfo /usr/share/zoneinfo
ENV TZ=Asia/Shanghai
ENV ZONEINFO=/usr/share/zoneinfo
ENV NODE_LISTEN_ADDR=:2988
ENV NODE_HTTPS_LISTEN_ADDR=:2989
ENV NODE_CONSOLE_WEB_ROOT=/app/web
EXPOSE 2988 2989
CMD ["/app/one-proxy-node"]
