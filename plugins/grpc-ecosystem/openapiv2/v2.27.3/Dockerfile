# syntax=docker/dockerfile:1.18
FROM --platform=$BUILDPLATFORM golang:1.25.1-bookworm AS build

ARG TARGETOS TARGETARCH
ENV CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH

FROM golang:1.25.1-bookworm AS build
RUN --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 \
    go install -ldflags="-s -w" -trimpath github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@v2.27.3 \
 && mv /go/bin/${GOOS}_${GOARCH}/protoc-gen-openapiv2 /go/bin/protoc-gen-openapiv2 || true

FROM scratch
COPY --from=build --link --chown=root:root /etc/passwd /etc/passwd
COPY --from=build --link /go/bin/protoc-gen-openapiv2 /
USER nobody
ENTRYPOINT [ "/protoc-gen-openapiv2" ]
