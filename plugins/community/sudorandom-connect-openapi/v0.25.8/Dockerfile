# syntax=docker/dockerfile:1.26
FROM --platform=$BUILDPLATFORM golang:1.27.0-trixie@sha256:df98008ecd2b0ecc9f0a94d1b07e3564a9c92b555369b33d9b5f60d0765b2db7 AS build

ARG TARGETOS TARGETARCH
ENV CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH

RUN --mount=type=cache,target=/go/pkg/mod \
    go install -ldflags "-s -w" -tags remote_plugin -trimpath github.com/sudorandom/protoc-gen-connect-openapi@v0.25.8 \
 && mv /go/bin/${GOOS}_${GOARCH}/protoc-gen-connect-openapi /go/bin/protoc-gen-connect-openapi || true

FROM scratch
COPY --from=build --link /etc/passwd /etc/passwd
COPY --from=build --link --chown=root:root /go/bin/protoc-gen-connect-openapi .
USER nobody
ENTRYPOINT [ "/protoc-gen-connect-openapi" ]
