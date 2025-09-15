# syntax=docker/dockerfile:1.6
FROM golang:1.22-alpine AS build
WORKDIR /app

# deps cache
COPY go.mod ./
RUN go mod download

# source
COPY . .
RUN rm -f go.sum && go mod tidy

ARG SERVICE=services/api-gateway
ENV CGO_ENABLED=0
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    cd ${SERVICE} && go build -o /out/app

FROM alpine
WORKDIR /app
COPY --from=build /out/app /app/app
# hanya untuk api-gateway; aman untuk service lain
COPY services/api-gateway/static /app/static
EXPOSE 8080
ENTRYPOINT ["/app/app"]
