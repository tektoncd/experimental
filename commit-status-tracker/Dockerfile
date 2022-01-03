# Build the manager binary
#FROM golang:1.16 as builder
#
#WORKDIR /workspace
## Copy the Go Modules manifests
#COPY go.mod go.mod
#COPY go.sum go.sum
## cache deps before building and copying source so that we don't need to re-download as much
## and so that source changes don't invalidate our downloaded layer
#RUN go mod download
#
## Copy the go source
#COPY main.go main.go
#COPY controllers/ controllers/
#
## Build
#RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o manager main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM registry.access.redhat.com/ubi8-minimal@sha256:e0814339ffc6c933652bed0c5f8b6416b9a3d40be2f49f95e6e4128387d2a24a

ENV OPERATOR=/usr/local/bin/commit-status-tracker \
    USER_UID=1001 \
    USER_NAME=commit-status-tracker
COPY bin/manager ${OPERATOR}

COPY docker-scripts/bin /usr/local/bin
RUN  /usr/local/bin/user_setup


ENTRYPOINT ["/manager"]
USER ${USER_UID}

