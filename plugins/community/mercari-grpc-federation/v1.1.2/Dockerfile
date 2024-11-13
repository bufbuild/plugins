# syntax=docker/dockerfile:1.9
FROM --platform=$BUILDPLATFORM golang:1.22.5-bookworm AS build

ARG TARGETOS TARGETARCH
ENV CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH

RUN --mount=type=cache,target=/go/pkg/mod \
    go install -ldflags="-s -w" -trimpath github.com/mercari/grpc-federation/cmd/protoc-gen-grpc-federation@v1.1.2 \
 && mv /go/bin/${GOOS}_${GOARCH}/protoc-gen-grpc-federation /go/bin/protoc-gen-grpc-federation || true

FROM scratch
COPY --from=build --link --chown=root:root /etc/passwd /etc/passwd
COPY --from=build --link --chown=root:root /go/bin/protoc-gen-grpc-federation .
ENV GOROOT /
USER nobody
ENTRYPOINT [ "/protoc-gen-grpc-federation" ]
