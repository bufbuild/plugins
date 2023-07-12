# syntax=docker/dockerfile:1.4
FROM golang:1.20.6-bullseye AS build
RUN --mount=type=cache,target=/go/pkg/mod \
    git clone --depth=1 --branch=v4.2.0 https://github.com/roadrunner-server/grpc.git \
    && cd grpc/protoc_plugins/protoc-gen-php-grpc \
    && CGO_ENABLED=0 go build -ldflags "-s -w" -trimpath

FROM scratch
COPY --from=build --link --chown=root:root /etc/passwd /etc/passwd
COPY --from=build --link --chown=root:root /go/grpc/protoc_plugins/protoc-gen-php-grpc/protoc-gen-php-grpc /
USER nobody
ENTRYPOINT [ "/protoc-gen-php-grpc" ]
