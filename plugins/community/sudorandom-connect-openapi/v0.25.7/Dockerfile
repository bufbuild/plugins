# syntax=docker/dockerfile:1.24
FROM --platform=$BUILDPLATFORM golang:1.26.4-trixie@sha256:3424c834a8b26df82d4b64cf0c92e0f8f309b4ff7f85581f8ec5026e6f7a44da AS build

ARG TARGETOS TARGETARCH
ENV CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH

RUN --mount=type=cache,target=/go/pkg/mod \
    go install -ldflags "-s -w" -tags remote_plugin -trimpath github.com/sudorandom/protoc-gen-connect-openapi@v0.25.7 \
 && mv /go/bin/${GOOS}_${GOARCH}/protoc-gen-connect-openapi /go/bin/protoc-gen-connect-openapi || true

FROM scratch
COPY --from=build --link /etc/passwd /etc/passwd
COPY --from=build --link --chown=root:root /go/bin/protoc-gen-connect-openapi .
USER nobody
ENTRYPOINT [ "/protoc-gen-connect-openapi" ]
