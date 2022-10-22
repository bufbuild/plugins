# syntax=docker/dockerfile:1.4
FROM golang:1.19.2-bullseye AS build
RUN --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 \
    go install -ldflags "-s -w" -trimpath github.com/pseudomuto/protoc-gen-doc/cmd/protoc-gen-doc@v1.5.1

FROM scratch
COPY --from=build --link --chown=root:root /etc/passwd /etc/passwd
COPY --from=build --link --chown=root:root /go/bin/protoc-gen-doc /
USER nobody
ENTRYPOINT [ "/protoc-gen-doc" ]
