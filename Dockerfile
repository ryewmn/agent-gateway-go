# syntax=docker/dockerfile:1
FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/gateway ./cmd/gateway

FROM alpine:3.21
RUN addgroup -S app && adduser -S -G app -u 10001 app
COPY --from=build /out/gateway /usr/local/bin/gateway
COPY configs/local.json /etc/agent-gateway/config.json
USER 10001:10001
EXPOSE 8080
ENTRYPOINT ["gateway"]
CMD ["-config", "/etc/agent-gateway/config.json"]
