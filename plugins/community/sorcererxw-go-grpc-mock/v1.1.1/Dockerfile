# syntax=docker/dockerfile:1.4
FROM golang:1.20.3-bullseye AS build
RUN --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 \
    go install -ldflags="-s -w" -trimpath github.com/sorcererxw/protoc-gen-go-grpc-mock@v1.1.1

FROM scratch
COPY --from=build --link --chown=root:root /etc/passwd /etc/passwd
COPY --from=build --link --chown=root:root /go/bin/protoc-gen-go-grpc-mock .
USER nobody
ENTRYPOINT [ "/protoc-gen-go-grpc-mock" ]
