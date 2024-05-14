# syntax=docker/dockerfile:1.7
FROM --platform=$BUILDPLATFORM golang:1.22.3-bookworm AS build

ARG TARGETOS TARGETARCH
ENV CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH

RUN --mount=type=cache,target=/go/pkg/mod \
    go install -ldflags="-s -w" -trimpath connectrpc.com/connect/cmd/protoc-gen-connect-go@v1.16.2 \
 && mv /go/bin/${GOOS}_${GOARCH}/protoc-gen-connect-go /go/bin/protoc-gen-connect-go || true

FROM scratch
COPY --from=build --link --chown=root:root /etc/passwd /etc/passwd
COPY --from=build --link /go/bin/protoc-gen-connect-go /
USER nobody
ENTRYPOINT [ "/protoc-gen-connect-go" ]
