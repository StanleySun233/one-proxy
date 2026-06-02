ARG NODE_BASE_IMAGE=oneproxy-node-base:latest

FROM ${NODE_BASE_IMAGE} AS builder
WORKDIR /workspace/apps/one-proxy-node
COPY apps/one-proxy-node ./
RUN mkdir -p /out/runtime /out/zoneinfo && cp -a /usr/share/zoneinfo/. /out/zoneinfo/ && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/one-proxy-node ./cmd/one-proxy-node

FROM ${NODE_BASE_IMAGE}
WORKDIR /app
COPY --from=builder /out/one-proxy-node /app/one-proxy-node
COPY --from=builder /out/runtime /app/runtime
COPY --from=builder /out/zoneinfo /usr/share/zoneinfo
ENV TZ=Asia/Shanghai
ENV ZONEINFO=/usr/share/zoneinfo
ENV NODE_LISTEN_ADDR=:2988
ENV NODE_HTTPS_LISTEN_ADDR=:2989
EXPOSE 2988 2989
CMD ["/app/one-proxy-node"]
