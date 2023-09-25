# syntax=docker/dockerfile:1.6
FROM golang:1.21.1-bullseye AS build
RUN git clone --depth=1 --branch v4.5.1 https://github.com/roadrunner-server/grpc.git
RUN --mount=type=cache,target=/go/pkg/mod \
    cd grpc/protoc_plugins/protoc-gen-php-grpc && \
    CGO_ENABLED=0 \
    go install -ldflags="-s -w" -trimpath

FROM scratch
COPY --from=build --link --chown=root:root /etc/passwd /etc/passwd
COPY --from=build --link --chown=root:root /go/bin/protoc-gen-php-grpc /
USER nobody
ENTRYPOINT [ "/protoc-gen-php-grpc" ]
