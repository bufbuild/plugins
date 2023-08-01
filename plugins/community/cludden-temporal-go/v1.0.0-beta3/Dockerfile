# syntax=docker/dockerfile:1.4
FROM golang:1.20.3-bullseye AS build
RUN --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 \
    go install -ldflags="-s -w" -trimpath github.com/cludden/protoc-gen-go-temporal/cmd/protoc-gen-go_temporal@v1.0.0-beta.3

FROM scratch
COPY --from=build --link --chown=root:root /etc/passwd /etc/passwd
COPY --from=build --link --chown=root:root /go/bin/protoc-gen-go_temporal .
USER nobody
ENTRYPOINT [ "/protoc-gen-go_temporal" ]
