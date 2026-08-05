# syntax=docker/dockerfile:1.24
FROM --platform=$BUILDPLATFORM golang:1.26.5-trixie@sha256:4ee9ffa999b4583ce281939cdff828763083610292f252279a0cee77473bd9a7 AS build

ARG TARGETOS TARGETARCH
ENV CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH

FROM golang:1.26.5-trixie@sha256:4ee9ffa999b4583ce281939cdff828763083610292f252279a0cee77473bd9a7 AS build
RUN --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 \
    go install -ldflags="-s -w" -trimpath github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@v2.30.0 \
 && mv /go/bin/${GOOS}_${GOARCH}/protoc-gen-openapiv2 /go/bin/protoc-gen-openapiv2 || true

FROM scratch
COPY --from=build --link --chown=root:root /etc/passwd /etc/passwd
COPY --from=build --link /go/bin/protoc-gen-openapiv2 /
USER nobody
ENTRYPOINT [ "/protoc-gen-openapiv2" ]
