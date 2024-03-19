# syntax=docker/dockerfile:1.6
FROM golang:1.22.1-bookworm AS build
RUN --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 \
    go install -ldflags="-s -w" -trimpath github.com/mercari/grpc-federation/cmd/protoc-gen-grpc-federation@v0.13.4

FROM scratch
COPY --from=build --link --chown=root:root /etc/passwd /etc/passwd
COPY --from=build --link --chown=root:root /go/bin/protoc-gen-grpc-federation .
ENV GOROOT /
USER nobody
ENTRYPOINT [ "/protoc-gen-grpc-federation" ]
