# syntax=docker/dockerfile:1.6
# filename:Dockerfile
# syntax=docker/dockerfile:1.6
FROM golang:1.22-alpine AS build
WORKDIR /app

# 1) deps dulu (cache-friendly)
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

# 2) copy seluruh source
COPY . .

# 3) refresh deps SETELAH source dicopy
RUN --mount=type=cache,target=/go/pkg/mod go mod tidy

# 4) build biner
ARG SERVICE=services/api-gateway
ENV CGO_ENABLED=0
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    sh -lc "cd ${SERVICE} && go build -trimpath -ldflags='-s -w' -o /out/app"

# 5) stage final
FROM alpine:3.20
WORKDIR /app
# runtime deps minimal
RUN apk add --no-cache ca-certificates tzdata
COPY --from=build /out/app /app/app

# opsi: copy static hanya bila diminta
ARG WITH_STATIC=0
# jika butuh: docker build --build-arg SERVICE=services/api-gateway --build-arg WITH_STATIC=1 ...
RUN if [ "$WITH_STATIC" = "1" ] && [ -d "/tmp/static" ]; then true; fi
# cara yang portable untuk api-gateway:
# (gunakan COPY terpisah dari compose via build args / target khusus bila perlu)

ENTRYPOINT ["/app/app"]
# NOTE: Tidak EXPOSE port di sini (tiap service beda). Biarkan compose yang map portnya.
