# syntax=docker/dockerfile:1.27
FROM --platform=$BUILDPLATFORM golang:1.27.4-trixie@sha256:f18c3d73a184fb0582828510722e723b63a93f7188d11dd223d9be6f6880e65d AS build

ARG TARGETOS TARGETARCH
ENV CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH

RUN --mount=type=cache,target=/go/pkg/mod \
    go install -ldflags="-s -w" -trimpath connectrpc.com/connect/cmd/protoc-gen-connect-go@v1.21.1 \
 && mv /go/bin/${GOOS}_${GOARCH}/protoc-gen-connect-go /go/bin/protoc-gen-connect-go || true

FROM scratch
COPY --from=build --link --chown=root:root /etc/passwd /etc/passwd
COPY --from=build --link /go/bin/protoc-gen-connect-go /
USER nobody
ENTRYPOINT [ "/protoc-gen-connect-go" ]
