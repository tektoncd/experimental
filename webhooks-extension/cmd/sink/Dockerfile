# Start from a Debian image with the latest version of Go installed
# and a workspace (GOPATH) configured at /go.
FROM golang:1.12-alpine as builder

# Copy the local package files to the container's workspace.
ADD . /go/src/github.com/tektoncd/experimental/webhooks-extension
WORKDIR /go/src/github.com/tektoncd/experimental/webhooks-extension

# Install git
RUN apk add git

# Install dep and verify dependencies
ADD https://github.com/golang/dep/releases/download/v0.5.1/dep-linux-amd64 /usr/bin/dep
RUN chmod +x /usr/bin/dep
RUN dep ensure -vendor-only

# Build the sink command inside the container.
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix nocgo -o /app github.com/tektoncd/experimental/webhooks-extension/cmd/sink



FROM scratch
COPY --from=builder /app .

# Run the sink command by default when the container starts.
ENTRYPOINT ["./app"]

# Document that the service listens on port 8080.
EXPOSE 8080