# syntax=docker/dockerfile:1.4
FROM golang:1.19.4-bullseye AS build
RUN --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 \
    go install -ldflags "-s -w" -trimpath github.com/infobloxopen/protoc-gen-gorm@v1.1.2

FROM scratch
COPY --from=build --link --chown=root:root /etc/passwd /etc/passwd
COPY --from=build --link --chown=root:root /go/bin/protoc-gen-gorm /
USER nobody
ENTRYPOINT [ "/protoc-gen-gorm" ]
