FROM golang:1.20.2-alpine3.17 as bufwasmtool-builder
WORKDIR /workspace
COPY go.mod go.sum /workspace/
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download
COPY cmd /workspace/cmd
COPY internal /workspace/internal
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 \
    go build -ldflags "-s -w" -trimpath -o /go/bin/bufwasmtool ./internal/infra/cmd/bufwasmtool
