# syntax=docker/dockerfile:1.4
FROM golang:1.19.1-bullseye AS build
RUN --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 \
    go install -ldflags "-s -w" -trimpath github.com/planetscale/vtprotobuf/cmd/protoc-gen-go-vtproto@v0.3.0

FROM scratch
COPY --from=build --link /etc/passwd /etc/passwd
COPY --from=build --link --chown=root:root /go/bin/protoc-gen-go-vtproto .
USER nobody
ENTRYPOINT [ "/protoc-gen-go-vtproto" ]
