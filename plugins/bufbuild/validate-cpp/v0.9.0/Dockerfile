# syntax=docker/dockerfile:1.4
FROM golang:1.19.3-bullseye AS build
RUN CGO_ENABLED=0 go install -ldflags "-s -w" -trimpath github.com/envoyproxy/protoc-gen-validate/cmd/protoc-gen-validate-cpp@v0.9.0

FROM scratch
COPY --from=build --link /etc/passwd /etc/passwd
COPY --from=build --link --chown=root:root /go/bin/protoc-gen-validate-cpp .
USER nobody
ENTRYPOINT [ "/protoc-gen-validate-cpp" ]
