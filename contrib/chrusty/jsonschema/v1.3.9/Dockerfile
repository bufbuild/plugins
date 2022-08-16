# syntax=docker/dockerfile:1.4
FROM golang:1.18-bullseye AS build

ARG PLUGIN_VERSION
RUN test -n "${PLUGIN_VERSION}"

WORKDIR /app
RUN --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 \
    go install -ldflags "-s -w" -trimpath github.com/chrusty/protoc-gen-jsonschema/cmd/protoc-gen-jsonschema@${PLUGIN_VERSION#v}

FROM scratch
WORKDIR /app
COPY --from=build --chown=root:root /etc/passwd /etc/passwd
COPY --from=build --chown=root:root /go/bin/protoc-gen-jsonschema .
USER nobody
ENTRYPOINT [ "/app/protoc-gen-jsonschema" ]
