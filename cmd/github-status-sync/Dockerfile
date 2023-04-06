ARG GO_VERSION="1.19"
FROM golang:${GO_VERSION}-alpine as base
ENV GOPROXY="https://artifacts.src.ec.ai/artifactory/api/go/go-all"
WORKDIR /go/src/github.com/ElementalCognition/tekton-toolbox
COPY ../../go.* ./
RUN --mount=type=cache,id=gomod,target=/go/pkg/mod \
    go mod download -x

FROM base AS build
ENV CGO_ENABLED=0
COPY ../../cmd/github-status-sync ./cmd/github-status-sync
COPY ../../internal ./internal
COPY ../../pkg ./pkg
RUN --mount=type=cache,id=gomod,target=/go/pkg/mod \
    --mount=type=cache,id=gobuild,target=/root/.cache/go-build \
    go build -o /go/bin/github-status-sync ./cmd/github-status-sync

FROM alpine:3.17
COPY --from=build /go/bin/github-status-sync /usr/local/bin/github-status-sync
CMD ["github-status-sync"]
