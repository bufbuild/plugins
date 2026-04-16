# syntax=docker/dockerfile:1.23
FROM --platform=$BUILDPLATFORM golang:1.26.2-trixie@sha256:c0074c718b473f3827043f86532c4c0ff537e3fe7a81b8219b0d1ccfcc2c9a09 AS build

ARG TARGETOS TARGETARCH
ENV CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH

FROM golang:1.26.2-trixie@sha256:c0074c718b473f3827043f86532c4c0ff537e3fe7a81b8219b0d1ccfcc2c9a09 AS build
RUN --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 \
    go install -ldflags="-s -w" -trimpath github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@v2.29.0 \
 && mv /go/bin/${GOOS}_${GOARCH}/protoc-gen-openapiv2 /go/bin/protoc-gen-openapiv2 || true

FROM scratch
COPY --from=build --link --chown=root:root /etc/passwd /etc/passwd
COPY --from=build --link /go/bin/protoc-gen-openapiv2 /
USER nobody
ENTRYPOINT [ "/protoc-gen-openapiv2" ]
