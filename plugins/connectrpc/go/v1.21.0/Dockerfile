# syntax=docker/dockerfile:1.26
FROM --platform=$BUILDPLATFORM golang:1.27.1-trixie@sha256:9baa6b4187bbb98d240372a8a235ac0bb6b5ddd52bba1431dc2f7c0705862728 AS build

ARG TARGETOS TARGETARCH
ENV CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH

RUN --mount=type=cache,target=/go/pkg/mod \
    go install -ldflags="-s -w" -trimpath connectrpc.com/connect/cmd/protoc-gen-connect-go@v1.21.0 \
 && mv /go/bin/${GOOS}_${GOARCH}/protoc-gen-connect-go /go/bin/protoc-gen-connect-go || true

FROM scratch
COPY --from=build --link --chown=root:root /etc/passwd /etc/passwd
COPY --from=build --link /go/bin/protoc-gen-connect-go /
USER nobody
ENTRYPOINT [ "/protoc-gen-connect-go" ]
