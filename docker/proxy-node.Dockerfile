FROM golang:1.23-bookworm AS builder
WORKDIR /workspace/apps/proxy-node
COPY apps/proxy-node/go.mod apps/proxy-node/go.sum ./
RUN go mod download
COPY apps/proxy-node ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/proxy-node ./cmd/proxy-node

FROM debian:bookworm-slim
WORKDIR /app
RUN useradd -r -u 10001 -m proxynode && mkdir -p /app/runtime && chown -R proxynode:proxynode /app
COPY --from=builder /out/proxy-node /app/proxy-node
USER proxynode
ENV NODE_LISTEN_ADDR=:2888
ENV NODE_HTTPS_LISTEN_ADDR=:2889
EXPOSE 2888 2889
CMD ["/app/proxy-node"]
