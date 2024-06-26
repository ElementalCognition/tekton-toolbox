ARG GO_VERSION="1.22"
FROM golang:${GO_VERSION}-alpine as base
ENV GOPROXY="https://artifacts.src.ec.ai/artifactory/api/go/go-all"
WORKDIR /go/src/github.com/ElementalCognition/tekton-toolbox
COPY ../../go.* ./
RUN --mount=type=cache,id=gomod,target=/go/pkg/mod \
    go mod download -x

FROM base AS build
ENV CGO_ENABLED=0
COPY ../../cmd/gcs-log-proxy ./cmd/gcs-log-proxy
COPY ../../internal ./internal
COPY ../../pkg ./pkg
RUN --mount=type=cache,id=gomod,target=/go/pkg/mod \
    --mount=type=cache,id=gobuild,target=/root/.cache/go-build \
    go build -o /go/bin/gcs-log-proxy ./cmd/gcs-log-proxy

FROM alpine:3.14
COPY --from=build /go/bin/gcs-log-proxy /usr/local/bin/gcs-log-proxy
CMD ["gcs-log-proxy"]
