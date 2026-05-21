# syntax=docker/dockerfile:1.23
FROM --platform=$BUILDPLATFORM golang:1.26.3-trixie@sha256:a085df697019cb63b40a70f6a92b948f7dc9df96dfcb2c20ba6eed25ce28f5b3 AS build

ARG TARGETOS TARGETARCH
ENV CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH

RUN --mount=type=cache,target=/go/pkg/mod \
    go install -ldflags="-s -w" -trimpath connectrpc.com/connect/cmd/protoc-gen-connect-go@v1.20.0 \
 && mv /go/bin/${GOOS}_${GOARCH}/protoc-gen-connect-go /go/bin/protoc-gen-connect-go || true

FROM scratch
COPY --from=build --link --chown=root:root /etc/passwd /etc/passwd
COPY --from=build --link /go/bin/protoc-gen-connect-go /
USER nobody
ENTRYPOINT [ "/protoc-gen-connect-go" ]
