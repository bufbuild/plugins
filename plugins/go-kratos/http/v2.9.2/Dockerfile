# syntax=docker/dockerfile:1.17
FROM --platform=$BUILDPLATFORM golang:1.25.0-bookworm AS build

ARG TARGETOS TARGETARCH
ENV CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH

RUN --mount=type=cache,target=/go/pkg/mod \
    go install -ldflags="-s -w" -trimpath go install github.com/go-kratos/kratos/cmd/protoc-gen-go-http/v2@b9fab9a \
 && mv /go/bin/${GOOS}_${GOARCH}/protoc-gen-go-http /go/bin/protoc-gen-go-http || true

FROM scratch
COPY --from=build --link --chown=root:root /etc/passwd /etc/passwd
COPY --from=build --link --chown=root:root /go/bin/protoc-gen-go-http .
USER nobody
ENTRYPOINT [ "/protoc-gen-go-http" ]
