# syntax=docker/dockerfile:1.6
FROM --platform=$BUILDPLATFORM golang:1.22.3-bookworm AS build

ARG TARGETOS TARGETARCH
ENV CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH 

RUN --mount=type=cache,target=/go/pkg/mod \
    go install -ldflags="-s -w" -trimpath github.com/bufbuild/protoschema-plugins/cmd/protoc-gen-pubsub@v0.1.4 \
    && mv /go/bin/${GOOS}_${GOARCH}/protoc-gen-pubsub /go/bin/protoc-gen-pubsub || true

FROM scratch
COPY --from=build --link --chown=root:root /etc/passwd /etc/passwd
COPY --from=build --link /go/bin/protoc-gen-pubsub /
USER nobody
ENTRYPOINT [ "/protoc-gen-pubsub" ]
