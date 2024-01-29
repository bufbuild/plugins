# syntax=docker/dockerfile:1.6
FROM golang:1.21.6-bookworm AS build
RUN --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 \
    go install -ldflags="-s -w" -trimpath github.com/mfridman/protoc-gen-go-json@v1.2.0

FROM scratch
COPY --from=build --link --chown=root:root /etc/passwd /etc/passwd
COPY --from=build --link --chown=root:root /go/bin/protoc-gen-go-json .
USER nobody
ENTRYPOINT [ "/protoc-gen-go-json" ]
