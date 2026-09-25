# syntax=docker/dockerfile:1.26
FROM --platform=$BUILDPLATFORM golang:1.27.1-trixie@sha256:433790e515d27dc6003e847e644cc0af956985cf315c1c58a3b73ee2dd305183 AS build

ARG TARGETOS TARGETARCH
ENV CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH

RUN --mount=type=cache,target=/go/pkg/mod \
    go install -ldflags "-s -w" -tags remote_plugin -trimpath github.com/sudorandom/protoc-gen-connect-openapi@v0.27.3 \
 && mv /go/bin/${GOOS}_${GOARCH}/protoc-gen-connect-openapi /go/bin/protoc-gen-connect-openapi || true

FROM scratch
COPY --from=build --link /etc/passwd /etc/passwd
COPY --from=build --link --chown=root:root /go/bin/protoc-gen-connect-openapi .
USER nobody
ENTRYPOINT [ "/protoc-gen-connect-openapi" ]
