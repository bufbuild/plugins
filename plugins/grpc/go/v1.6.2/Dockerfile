# syntax=docker/dockerfile:1.23
FROM --platform=$BUILDPLATFORM golang:1.26.3-trixie@sha256:d08bf3ed2bd263088ca8e23fefaf10f1b71769f6932f0a4017ba28d2a5baf001 AS build

ARG TARGETOS TARGETARCH
ENV CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH

WORKDIR /tmp
RUN git clone --depth 1 --branch cmd/protoc-gen-go-grpc/v1.6.2 https://github.com/grpc/grpc-go.git
WORKDIR /tmp/grpc-go
COPY separate-package.patch /tmp/grpc-go
RUN git apply separate-package.patch
WORKDIR /tmp/grpc-go/cmd/protoc-gen-go-grpc
RUN --mount=type=cache,target=/go/pkg/mod \
    go build -o protoc-gen-go-grpc -ldflags "-s -w" -trimpath

FROM scratch
COPY --from=build --link /etc/passwd /etc/passwd
COPY --from=build --link --chown=root:root /tmp/grpc-go/cmd/protoc-gen-go-grpc/protoc-gen-go-grpc .
USER nobody
ENTRYPOINT [ "/protoc-gen-go-grpc" ]
