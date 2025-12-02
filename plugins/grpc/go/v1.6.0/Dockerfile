# syntax=docker/dockerfile:1.19
FROM --platform=$BUILDPLATFORM golang:1.25.4-bookworm AS build

ARG TARGETOS TARGETARCH
ENV CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH

WORKDIR /tmp
RUN git clone --depth 1 --branch cmd/protoc-gen-go-grpc/v1.6.0 https://github.com/grpc/grpc-go.git
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
