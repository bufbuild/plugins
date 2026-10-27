# syntax=docker/dockerfile:1.27
FROM --platform=$BUILDPLATFORM golang:1.27.3-trixie@sha256:fbb7ae40b62ff8afe31662df570d39b4f5052ac32b90dae8e60d7657a20a74f4 AS build

ARG TARGETOS TARGETARCH
ENV CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH

WORKDIR /tmp
RUN git clone --depth 1 --branch v2.33.0 https://github.com/grpc-ecosystem/grpc-gateway.git
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
