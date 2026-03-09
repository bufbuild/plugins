# syntax=docker/dockerfile:1.17
FROM --platform=$BUILDPLATFORM golang:1.26.1-bookworm AS build

ARG TARGETOS TARGETARCH
ENV CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH

RUN --mount=type=cache,target=/go/pkg/mod \
    go install -ldflags="-s -w" -trimpath github.com/cjermain/protoc-gen-pydantic@v0.10.0 \
 && mv /go/bin/${GOOS}_${GOARCH}/protoc-gen-pydantic /go/bin/protoc-gen-pydantic || true

FROM scratch
COPY --from=build --link --chown=root:root /etc/passwd /etc/passwd
COPY --from=build --link --chown=root:root /go/bin/protoc-gen-pydantic .
USER nobody
ENTRYPOINT [ "/protoc-gen-pydantic" ]
