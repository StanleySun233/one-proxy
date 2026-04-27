FROM golang:1.23-bookworm AS builder
WORKDIR /workspace/apps/one-proxy-node
COPY apps/one-proxy-node/go.mod apps/one-proxy-node/go.sum ./
RUN go mod download
COPY apps/one-proxy-node ./
RUN mkdir -p /out/runtime /out/zoneinfo && cp -a /usr/share/zoneinfo/. /out/zoneinfo/ && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/one-proxy-node ./cmd/one-proxy-node

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --chown=nonroot:nonroot --from=builder /out/one-proxy-node /app/one-proxy-node
COPY --chown=nonroot:nonroot --from=builder /out/runtime /app/runtime
COPY --chown=nonroot:nonroot --from=builder /out/zoneinfo /usr/share/zoneinfo
ENV TZ=Asia/Shanghai
ENV ZONEINFO=/usr/share/zoneinfo
ENV NODE_LISTEN_ADDR=:2988
ENV NODE_HTTPS_LISTEN_ADDR=:2989
EXPOSE 2988 2989
CMD ["/app/one-proxy-node"]
