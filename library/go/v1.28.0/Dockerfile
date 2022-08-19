# syntax=docker/dockerfile:1.4
FROM golang:1.19-bullseye AS build
ARG PLUGIN_VERSION
RUN test -n "${PLUGIN_VERSION}"
WORKDIR /app
RUN --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 \
    go install -ldflags "-s -w" -trimpath google.golang.org/protobuf/cmd/protoc-gen-go@${PLUGIN_VERSION}

FROM scratch
WORKDIR /app
COPY --from=build /etc/passwd /etc/passwd
COPY --from=build --chown=root:root /go/bin/protoc-gen-go .
USER nobody
ENTRYPOINT [ "/app/protoc-gen-go" ]
