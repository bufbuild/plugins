# syntax=docker/dockerfile:1.27
FROM --platform=$BUILDPLATFORM golang:1.27.1-trixie@sha256:3b77fc618ec235a1ab412de7737f120dd507c57e8d87de4cbb7994fb94275ed5 AS build

ARG TARGETOS TARGETARCH
ENV CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH

RUN git clone --depth=1 --branch v6.0.0 https://github.com/roadrunner-server/grpc.git
RUN --mount=type=cache,target=/go/pkg/mod \
    cd grpc/protoc_plugins/protoc-gen-php-grpc \
 && go install -ldflags="-s -w" -trimpath \
 && mv /go/bin/${GOOS}_${GOARCH}/protoc-gen-php-grpc /go/bin/protoc-gen-php-grpc || true

FROM scratch
COPY --from=build --link --chown=root:root /etc/passwd /etc/passwd
COPY --from=build --link --chown=root:root /go/bin/protoc-gen-php-grpc /
USER nobody
ENTRYPOINT [ "/protoc-gen-php-grpc" ]
