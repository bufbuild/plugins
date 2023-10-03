# syntax=docker/dockerfile:1.6
FROM golang:1.21.1-bullseye AS build
RUN --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 \
    go install -ldflags="-s -w" -trimpath github.com/google/gnostic/cmd/protoc-gen-openapi@v0.7.0

FROM scratch
COPY --from=build --link --chown=root:root /etc/passwd /etc/passwd
COPY --from=build --link --chown=root:root /go/bin/protoc-gen-openapi .
USER nobody
ENTRYPOINT [ "/protoc-gen-openapi" ]
