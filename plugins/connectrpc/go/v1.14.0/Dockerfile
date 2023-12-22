# syntax=docker/dockerfile:1.4
FROM golang:1.21.5-bookworm AS build
RUN --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 \
    go install -ldflags="-s -w" -trimpath connectrpc.com/connect/cmd/protoc-gen-connect-go@v1.14.0

FROM scratch
COPY --from=build --link --chown=root:root /etc/passwd /etc/passwd
COPY --from=build --link /go/bin/protoc-gen-connect-go /
USER nobody
ENTRYPOINT [ "/protoc-gen-connect-go" ]
