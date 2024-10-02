# syntax=docker/dockerfile:1.10
FROM --platform=$BUILDPLATFORM golang:1.23.2-bookworm AS build

ARG TARGETOS TARGETARCH
ENV CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH

RUN --mount=type=cache,target=/go/pkg/mod \
    go install -ldflags="-s -w" -trimpath pluginrpc.com/pluginrpc/cmd/protoc-gen-pluginrpc-go@v0.5.0 \
    && mv /go/bin/${GOOS}_${GOARCH}/protoc-gen-pluginrpc-go /go/bin/protoc-gen-pluginrpc-go || true

FROM scratch
COPY --from=build --link --chown=root:root /etc/passwd /etc/passwd
COPY --from=build --link --chown=root:root /go/bin/protoc-gen-pluginrpc-go .
USER nobody
ENTRYPOINT [ "/protoc-gen-pluginrpc-go" ]
