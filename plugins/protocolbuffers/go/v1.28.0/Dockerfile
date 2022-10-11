# syntax=docker/dockerfile:1.4
FROM golang:1.19.1-bullseye AS build
RUN --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 \
    go install -ldflags "-s -w" -trimpath google.golang.org/protobuf/cmd/protoc-gen-go@v1.28.0

FROM scratch
COPY --from=build --link /etc/passwd /etc/passwd
COPY --from=build --link --chown=root:root /go/bin/protoc-gen-go .
USER nobody
ENTRYPOINT [ "/protoc-gen-go" ]
