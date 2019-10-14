FROM golang:1.13 as builder

RUN curl -fsSL -o /usr/local/bin/dep https://github.com/golang/dep/releases/download/v0.5.0/dep-linux-amd64 && chmod +x /usr/local/bin/dep
WORKDIR /go/src/github.com/tektoncd/experimental/webhooks-extension/config/triggers_prototype/interceptor
COPY . .
RUN dep ensure -v
RUN CGO_ENABLED=0 GOOS=linux go build -o app cmd/server/server.go

FROM alpine:latest
# mailcap adds mime detection and ca-certificates help with TLS (basic stuff)
RUN apk --no-cache add ca-certificates mailcap && addgroup -S app && adduser -S app -G app
USER app
WORKDIR /app
COPY --from=builder /go/src/github.com/tektoncd/experimental/webhooks-extension/config/triggers_prototype/interceptor/app .
ENTRYPOINT ["./app"]