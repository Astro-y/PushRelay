FROM --platform=$BUILDPLATFORM node:22-alpine AS web
WORKDIR /src/web
COPY web/package.json web/pnpm-lock.yaml ./
RUN corepack enable && pnpm install --frozen-lockfile
COPY web/ ./
RUN pnpm run build

FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS api
WORKDIR /src
ARG TARGETOS
ARG TARGETARCH
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
COPY --from=web /src/web/dist ./internal/webui/dist
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w" -o /out/pushrelay ./cmd/server \
    && mkdir -p /out/data

FROM gcr.io/distroless/static-debian12:nonroot AS backend
WORKDIR /app
COPY --from=api /out/pushrelay /app/pushrelay
COPY openapi.yaml /app/openapi.yaml
COPY --from=api --chown=65532:65532 /out/data /data
VOLUME ["/data"]
EXPOSE 4426
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 CMD ["/app/pushrelay", "healthcheck"]
ENTRYPOINT ["/app/pushrelay"]
