# syntax=docker/dockerfile:1.4
FROM debian:bullseye-20221114 AS build

ARG TARGETARCH

RUN apt-get update \
 && apt-get install -y curl
RUN arch=${TARGETARCH}; \
    if [ "${arch}" = "amd64" ]; then arch="x86_64"; elif [ "${arch}" = "arm64" ]; then arch="aarch_64"; fi; \
    curl -fsSL -o /tmp/protobuf-javascript.tar.gz https://github.com/protocolbuffers/protobuf-javascript/releases/download/v3.21.2/protobuf-javascript-3.21.2-linux-${arch}.tar.gz \
 && cd /tmp \
 && tar zxf protobuf-javascript.tar.gz bin/protoc-gen-js \
 && rm -f protobuf-javascript.tar.gz

FROM gcr.io/distroless/cc-debian11
COPY --from=build --link --chmod=0755 /tmp/bin/protoc-gen-js .
USER nobody
ENTRYPOINT ["/protoc-gen-js"]
