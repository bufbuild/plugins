# syntax=docker/dockerfile:1.27
FROM --platform=$BUILDPLATFORM golang:1.27.3-trixie@sha256:c65ecfe613b539ac74c62a4c3b2c27e8f8a6076384c8f27e8a3c0f3be17cc385 AS build

ARG TARGETOS TARGETARCH
ENV CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH

RUN --mount=type=cache,target=/go/pkg/mod \
    go install -ldflags="-s -w" -trimpath github.com/ftshp/connect-php/cmd/protoc-gen-connect-php@v0.1.0 \
 && mv /go/bin/${GOOS}_${GOARCH}/protoc-gen-connect-php /go/bin/protoc-gen-connect-php || true

FROM scratch
COPY --from=build --link --chown=root:root /etc/passwd /etc/passwd
COPY --from=build --link /go/bin/protoc-gen-connect-php /
USER nobody
ENTRYPOINT [ "/protoc-gen-connect-php" ]
