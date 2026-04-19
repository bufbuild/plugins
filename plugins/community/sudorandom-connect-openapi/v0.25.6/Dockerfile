# syntax=docker/dockerfile:1.23
FROM --platform=$BUILDPLATFORM golang:1.26.2-trixie@sha256:c0074c718b473f3827043f86532c4c0ff537e3fe7a81b8219b0d1ccfcc2c9a09 AS build

ARG TARGETOS TARGETARCH
ENV CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH

RUN --mount=type=cache,target=/go/pkg/mod \
    go install -ldflags "-s -w" -tags remote_plugin -trimpath github.com/sudorandom/protoc-gen-connect-openapi@v0.25.6 \
 && mv /go/bin/${GOOS}_${GOARCH}/protoc-gen-connect-openapi /go/bin/protoc-gen-connect-openapi || true

FROM scratch
COPY --from=build --link /etc/passwd /etc/passwd
COPY --from=build --link --chown=root:root /go/bin/protoc-gen-connect-openapi .
USER nobody
ENTRYPOINT [ "/protoc-gen-connect-openapi" ]
