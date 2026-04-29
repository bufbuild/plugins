# syntax=docker/dockerfile:1.23
FROM --platform=$BUILDPLATFORM golang:1.26.2-trixie@sha256:4a7137ea573f79c86ae451ff05817ed762ef5597fcf732259e97abeb3108d873 AS build

ARG TARGETOS TARGETARCH
ENV CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH

RUN --mount=type=cache,target=/go/pkg/mod \
    go install -ldflags="-s -w" -trimpath github.com/bufbuild/protoschema-plugins/cmd/protoc-gen-jsonschema@v0.6.0 \
    && mv /go/bin/${GOOS}_${GOARCH}/protoc-gen-jsonschema /go/bin/protoc-gen-jsonschema || true

FROM scratch
COPY --from=build --link --chown=root:root /etc/passwd /etc/passwd
COPY --from=build --link /go/bin/protoc-gen-jsonschema /
USER nobody
ENTRYPOINT [ "/protoc-gen-jsonschema" ]
