ARG GO_VERSION="1.17"
FROM golang:${GO_VERSION}-alpine as base
ENV GOPROXY="https://artifacts.src.ec.ai/artifactory/api/go/go-all"
WORKDIR /go/src/github.com/ElementalCognition/tekton-toolbox
COPY ../../go.* ./
RUN --mount=type=cache,id=gomod,target=/go/pkg/mod \
    go mod download -x

FROM base AS build
ENV CGO_ENABLED=0
COPY ../../cmd/github-pipeline-config ./cmd/github-pipeline-config
COPY ../../internal ./internal
COPY ../../pkg ./pkg
RUN --mount=type=cache,id=gomod,target=/go/pkg/mod \
    --mount=type=cache,id=gobuild,target=/root/.cache/go-build \
    go build -o /go/bin/github-pipeline-config ./cmd/github-pipeline-config

FROM alpine:3.14
COPY --from=build /go/bin/github-pipeline-config /usr/local/bin/github-pipeline-config
CMD ["github-pipeline-config"]
