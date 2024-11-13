# syntax=docker/dockerfile:1.9
FROM golang:1.22.5-bookworm AS build
RUN CGO_ENABLED=0 go install -ldflags "-s -w" -trimpath github.com/envoyproxy/protoc-gen-validate/cmd/protoc-gen-validate-cpp@v1.1.0

FROM scratch
COPY --from=build --link /etc/passwd /etc/passwd
COPY --from=build --link --chown=root:root /go/bin/protoc-gen-validate-cpp .
USER nobody
ENTRYPOINT [ "/protoc-gen-validate-cpp" ]
