# syntax=docker/dockerfile:1.10
FROM --platform=$BUILDPLATFORM golang:1.23.3-bookworm AS build

ARG TARGETOS TARGETARCH
ENV CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH

RUN --mount=type=cache,target=/go/pkg/mod \
    go install -ldflags="-s -w" -trimpath github.com/GoogleCloudPlatform/protoc-gen-bq-schema/v2@v2.0.1 \
    && mv /go/bin/${GOOS}_${GOARCH}/protoc-gen-bq-schema /go/bin/protoc-gen-bq-schema || true

FROM scratch
COPY --from=build --link --chown=root:root /etc/passwd /etc/passwd
COPY --from=build --link /go/bin/protoc-gen-bq-schema /
USER nobody
ENTRYPOINT [ "/protoc-gen-bq-schema", "-logtostderr=true" ]
