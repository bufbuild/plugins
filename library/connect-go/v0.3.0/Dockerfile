# syntax=docker/dockerfile:1.4
FROM golang:1.19-bullseye AS build
ARG PLUGIN_VERSION
RUN test -n "${PLUGIN_VERSION}"
WORKDIR /app
ENV GOBIN=/app
RUN --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 \
    go install -ldflags="-s -w" github.com/bufbuild/connect-go/cmd/protoc-gen-connect-go@${PLUGIN_VERSION}

FROM scratch
WORKDIR /app
COPY --from=build --chown=root:root /etc/passwd /etc/passwd
COPY --from=build /app/protoc-gen-connect-go .
USER nobody
ENTRYPOINT [ "/app/protoc-gen-connect-go" ]
