# syntax=docker/dockerfile:1.4
FROM golang:1.19.3-bullseye AS build
WORKDIR /tmp
RUN git clone --depth 1 --branch v2.14.0 https://github.com/grpc-ecosystem/grpc-gateway.git
WORKDIR /tmp/grpc-gateway
COPY --link remote-packages.patch /tmp
RUN git apply < /tmp/remote-packages.patch
WORKDIR /tmp/grpc-gateway/protoc-gen-grpc-gateway
RUN --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 \
    go build -ldflags "-s -w" -trimpath

FROM scratch
COPY --from=build --link /etc/passwd /etc/passwd
COPY --from=build --link --chown=root:root /tmp/grpc-gateway/protoc-gen-grpc-gateway .
USER nobody
ENTRYPOINT [ "/protoc-gen-grpc-gateway" ]
