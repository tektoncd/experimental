# Start from a Debian image with the latest version of Go installed
# and a workspace (GOPATH) configured at /go.
FROM golang:1.12-alpine as builder

# Copy the local package files to the container's workspace.
ADD . /go/src/github.com/tektoncd/experimental/webhooks-extension
WORKDIR /go/src/github.com/tektoncd/experimental/webhooks-extension

# Install git, nodejs, npm
RUN apk add git nodejs npm bash

# Install dep and verify dependencies
ADD https://github.com/golang/dep/releases/download/v0.5.1/dep-linux-amd64 /usr/bin/dep
RUN chmod +x /usr/bin/dep
RUN dep ensure -vendor-only

RUN npm install
RUN npm rebuild node-sass
RUN ls -l
RUN cat ./build-it.sh
RUN npm run build

# # Build the extension command inside the container.
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix nocgo -o /app github.com/tektoncd/experimental/webhooks-extension/cmd/extension



FROM scratch
COPY --from=builder /app .
COPY --from=builder /go/src/github.com/tektoncd/experimental/webhooks-extension/dist /web

# Run the extension command by default when the container starts.
ENTRYPOINT ["./app"]

# Document that the service listens on port 8080.
EXPOSE 8080