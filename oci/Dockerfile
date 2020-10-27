# Copyright 2020 The Tekton Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

FROM golang:1.15.3 as build
LABEL description="Build container"
ENV ARCH=amd64
ENV OS=linux
ENV RELEASE=dev

WORKDIR /go/src/github.com/tektoncd/experimental/oci/
ADD . /go/src/github.com/tektoncd/experimental/oci/

RUN go get -d -v ./...
RUN GOOS=${OS} GOARCH=${ARCH} go build -ldflags "-X github.com/tektoncd/experimental/oci/pkg/cmd/version.clientVersion=${RELEASE}" -o bin/oci ./cmd/oci

FROM gcr.io/distroless/base@sha256:249859465bcde1cb15128ff0d9eb2bb54de67f72a834a7576e6649cfe0a27698
LABEL maintainer "Tekton Authors <tekton-dev@googlegroups.com>"
COPY --from=build /go/src/github.com/tektoncd/experimental/oci/bin/oci /oci
CMD ["/oci"]
