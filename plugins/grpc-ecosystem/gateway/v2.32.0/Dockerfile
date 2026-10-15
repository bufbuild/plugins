# syntax=docker/dockerfile:1.27
FROM --platform=$BUILDPLATFORM golang:1.27.2-trixie@sha256:e58d6f83b3416618d8bcac2b3dde1b7f7e3c4a77d25e88637f8bbae81536c48d AS build

ARG TARGETOS TARGETARCH
ENV CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH

WORKDIR /tmp
RUN git clone --depth 1 --branch v2.32.0 https://github.com/grpc-ecosystem/grpc-gateway.git
COPY --link separate_pkg_additional_imports.patch /tmp/separate_pkg_additional_imports.patch
WORKDIR /tmp/grpc-gateway
RUN git apply /tmp/separate_pkg_additional_imports.patch
RUN --mount=type=cache,target=/go/pkg/mod \
    GOOS= GOARCH= go test -run 'TestGenerator_GenerateSeparatePackage' ./protoc-gen-grpc-gateway/internal/gengateway/
WORKDIR /tmp/grpc-gateway/protoc-gen-grpc-gateway
RUN --mount=type=cache,target=/go/pkg/mod \
    go install -ldflags="-s -w" -trimpath \
    && mv /go/bin/${GOOS}_${GOARCH}/protoc-gen-grpc-gateway /go/bin/protoc-gen-grpc-gateway || true

FROM scratch
COPY --from=build --link --chown=root:root /etc/passwd /etc/passwd
COPY --from=build --link --chown=root:root /go/bin/protoc-gen-grpc-gateway /
USER nobody
ENTRYPOINT [ "/protoc-gen-grpc-gateway" ]
