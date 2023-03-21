# syntax=docker/dockerfile:1.4
FROM golang:1.20.2-bullseye AS build
RUN CGO_ENABLED=0 go install -ldflags "-s -w" -trimpath github.com/envoyproxy/protoc-gen-validate/cmd/protoc-gen-validate-java@v0.10.1

FROM scratch
COPY --from=build --link /etc/passwd /etc/passwd
COPY --from=build --link --chown=root:root /go/bin/protoc-gen-validate-java .
USER nobody
ENTRYPOINT [ "/protoc-gen-validate-java" ]
