# syntax=docker/dockerfile:1.12
FROM --platform=$BUILDPLATFORM golang:1.23.4-bookworm AS build

ARG TARGETOS TARGETARCH
ENV CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH

RUN --mount=type=cache,target=/go/pkg/mod \
    go install -ldflags="-s -w" -trimpath github.com/mfridman/protoc-gen-go-json@v1.5.0 \
    && mv /go/bin/${GOOS}_${GOARCH}/protoc-gen-go-json /go/bin/protoc-gen-go-json || true

FROM scratch
COPY --from=build --link --chown=root:root /etc/passwd /etc/passwd
COPY --from=build --link --chown=root:root /go/bin/protoc-gen-go-json .
USER nobody
ENTRYPOINT [ "/protoc-gen-go-json" ]
