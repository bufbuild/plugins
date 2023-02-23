# syntax=docker/dockerfile:1.4
FROM golang:1.19.5-bullseye AS build
RUN --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 \
    go install -ldflags="-s -w" -trimpath github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@v2.15.1

FROM scratch
COPY --from=build --link /etc/passwd /etc/passwd
COPY --from=build --link --chown=root:root /go/bin/protoc-gen-grpc-gateway .
USER nobody
ENTRYPOINT [ "/protoc-gen-grpc-gateway" ]
