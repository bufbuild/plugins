# syntax=docker/dockerfile:1.10
FROM --platform=$BUILDPLATFORM golang:1.23.3-bookworm AS build

ARG TARGETOS TARGETARCH
ENV CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH

RUN --mount=type=cache,target=/go/pkg/mod \
    go install -ldflags "-s -w" -trimpath google.golang.org/protobuf/cmd/protoc-gen-go@v1.35.2 \
 && mv /go/bin/${GOOS}_${GOARCH}/protoc-gen-go /go/bin/protoc-gen-go || true

FROM scratch
COPY --from=build --link /etc/passwd /etc/passwd
COPY --from=build --link --chown=root:root /go/bin/protoc-gen-go .
USER nobody
ENTRYPOINT [ "/protoc-gen-go" ]
