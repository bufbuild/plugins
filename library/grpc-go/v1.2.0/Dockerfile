# syntax=docker/dockerfile:1.4
FROM golang:1.19.0-bullseye AS build
ARG PLUGIN_VERSION
RUN test -n "${PLUGIN_VERSION}"
RUN --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 \
    go install -ldflags "-s -w" -trimpath google.golang.org/grpc/cmd/protoc-gen-go-grpc@${PLUGIN_VERSION}

FROM scratch
COPY --from=build --link /etc/passwd /etc/passwd
COPY --from=build --link --chown=root:root /go/bin/protoc-gen-go-grpc .
USER nobody
ENTRYPOINT [ "/protoc-gen-go-grpc" ]
