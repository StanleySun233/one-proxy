ARG PANEL_BASE_IMAGE=oneproxy-panel-base:latest

FROM ${PANEL_BASE_IMAGE} AS web-builder
WORKDIR /workspace/apps/panel/web
COPY apps/panel/web ./
RUN cp -a /base/apps/panel/web/node_modules ./node_modules
RUN npm run build

FROM ${PANEL_BASE_IMAGE} AS api-builder
WORKDIR /workspace/apps/panel/api
COPY apps/panel/api ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/one-proxy-panel ./cmd/one-proxy-panel

FROM ${PANEL_BASE_IMAGE}
WORKDIR /app
ENV NODE_ENV=production
ENV TZ=Asia/Shanghai
ENV PORT=2886
ENV HTTP_ADDR=127.0.0.1:2887
ENV CONTROL_PLANE_URL=http://127.0.0.1:2887

COPY --from=api-builder /out/one-proxy-panel /app/bin/one-proxy-panel
COPY --from=api-builder /workspace/apps/panel/api/migrations /app/apps/panel/api/migrations
COPY --from=web-builder /workspace/apps/panel/web/.next/standalone /app
COPY --from=web-builder /workspace/apps/panel/web/.next/static /app/.next/static
COPY --from=web-builder /workspace/apps/panel/web/public /app/public
COPY docker/one-proxy-panel-start.sh /app/one-proxy-panel-start.sh

RUN mkdir -p /app/data && chmod +x /app/one-proxy-panel-start.sh

EXPOSE 2886

CMD ["/app/one-proxy-panel-start.sh"]
