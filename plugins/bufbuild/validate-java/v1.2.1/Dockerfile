# syntax=docker/dockerfile:1.12
FROM golang:1.23.5-bookworm AS build
RUN CGO_ENABLED=0 go install -ldflags "-s -w" -trimpath github.com/envoyproxy/protoc-gen-validate/cmd/protoc-gen-validate-java@v1.2.1

FROM scratch
COPY --from=build --link /etc/passwd /etc/passwd
COPY --from=build --link --chown=root:root /go/bin/protoc-gen-validate-java .
USER nobody
ENTRYPOINT [ "/protoc-gen-validate-java" ]
