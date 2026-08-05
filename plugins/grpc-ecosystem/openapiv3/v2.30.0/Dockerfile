# syntax=docker/dockerfile:1.24
FROM --platform=$BUILDPLATFORM golang:1.26.6-trixie@sha256:b75d466dd608587fd66cca705a307ba65b889827d06ad61d6a75f0482b51b7c7 AS build

ARG TARGETOS TARGETARCH
ENV CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH

RUN --mount=type=cache,target=/go/pkg/mod \
    go install -ldflags="-s -w" -trimpath github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv3@v2.30.0 \
 && mv /go/bin/${GOOS}_${GOARCH}/protoc-gen-openapiv3 /go/bin/protoc-gen-openapiv3 || true

FROM scratch
COPY --from=build --link --chown=root:root /etc/passwd /etc/passwd
COPY --from=build --link /go/bin/protoc-gen-openapiv3 /
USER nobody
ENTRYPOINT [ "/protoc-gen-openapiv3" ]
