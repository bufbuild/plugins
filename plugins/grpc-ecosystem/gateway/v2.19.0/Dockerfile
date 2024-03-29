# syntax=docker/dockerfile:1.4
FROM golang:1.21.5-bookworm AS build

WORKDIR /tmp
RUN git clone --depth 1 --branch v2.19.0 https://github.com/grpc-ecosystem/grpc-gateway.git
COPY --link separate_pkg_additional_imports.patch /tmp/separate_pkg_additional_imports.patch
WORKDIR /tmp/grpc-gateway
RUN git apply /tmp/separate_pkg_additional_imports.patch
WORKDIR /tmp/grpc-gateway/protoc-gen-grpc-gateway
RUN --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 \
    go build -ldflags "-s -w" -trimpath

FROM scratch
COPY --from=build --link --chown=root:root /etc/passwd /etc/passwd
COPY --from=build --link /tmp/grpc-gateway/protoc-gen-grpc-gateway .
USER nobody
ENTRYPOINT [ "/protoc-gen-grpc-gateway" ]
