# syntax=docker/dockerfile:1.27
FROM --platform=$BUILDPLATFORM golang:1.27.1-trixie@sha256:3b77fc618ec235a1ab412de7737f120dd507c57e8d87de4cbb7994fb94275ed5 AS build

ARG TARGETOS TARGETARCH
ENV CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH

RUN --mount=type=cache,target=/go/pkg/mod \
    go install -ldflags="-s -w" -trimpath pluginrpc.com/pluginrpc/cmd/protoc-gen-pluginrpc-go@v0.6.0 \
    && mv /go/bin/${GOOS}_${GOARCH}/protoc-gen-pluginrpc-go /go/bin/protoc-gen-pluginrpc-go || true

FROM scratch
COPY --from=build --link --chown=root:root /etc/passwd /etc/passwd
COPY --from=build --link --chown=root:root /go/bin/protoc-gen-pluginrpc-go .
USER nobody
ENTRYPOINT [ "/protoc-gen-pluginrpc-go" ]
