# syntax=docker/dockerfile:1.27
FROM --platform=$BUILDPLATFORM golang:1.27.3-trixie@sha256:fbb7ae40b62ff8afe31662df570d39b4f5052ac32b90dae8e60d7657a20a74f4 AS build

ARG TARGETOS TARGETARCH
ENV CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH

RUN --mount=type=cache,target=/go/pkg/mod \
    go install -ldflags="-s -w" -trimpath github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv3@v2.33.0 \
 && mv /go/bin/${GOOS}_${GOARCH}/protoc-gen-openapiv3 /go/bin/protoc-gen-openapiv3 || true

FROM scratch
COPY --from=build --link --chown=root:root /etc/passwd /etc/passwd
COPY --from=build --link /go/bin/protoc-gen-openapiv3 /
USER nobody
ENTRYPOINT [ "/protoc-gen-openapiv3" ]
