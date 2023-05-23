# syntax=docker/dockerfile:1.4
FROM golang:1.20.1-bullseye AS build

COPY separate_pkg_additional_imports.patch /tmp/separate_pkg_additional_imports.patch
WORKDIR /tmp
RUN git clone --depth 1 --branch v2.15.2 https://github.com/grpc-ecosystem/grpc-gateway.git
WORKDIR /tmp/grpc-gateway
RUN git apply /tmp/separate_pkg_additional_imports.patch
WORKDIR /tmp/grpc-gateway/protoc-gen-grpc-gateway
RUN --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 \
    go build -ldflags "-s -w" -trimpath

FROM scratch
COPY --from=build --link /etc/passwd /etc/passwd
COPY --from=build --link --chown=root:root /tmp/grpc-gateway/protoc-gen-grpc-gateway .
USER nobody
ENTRYPOINT [ "/protoc-gen-grpc-gateway" ]
