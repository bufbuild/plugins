# syntax=docker/dockerfile:1.4
FROM golang:1.20.6-bullseye AS build
RUN --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 \
    go install -ldflags="-s -w" -trimpath github.com/roadrunner-server/grpc/protoc_plugins/v4/protoc-gen-php-grpc@v4.3.0

FROM scratch
COPY --from=build --link --chown=root:root /etc/passwd /etc/passwd
COPY --from=build --link --chown=root:root /go/bin/protoc-gen-php-grpc /
USER nobody
ENTRYPOINT [ "/protoc-gen-php-grpc" ]
