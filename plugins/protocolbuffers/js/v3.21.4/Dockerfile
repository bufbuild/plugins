# syntax=docker/dockerfile:1.8
FROM debian:bookworm-20240701 AS build

ARG TARGETARCH

RUN apt-get update \
 && apt-get install -y curl
RUN arch=${TARGETARCH}; \
    if [ "${arch}" = "amd64" ]; then arch="x86_64"; elif [ "${arch}" = "arm64" ]; then arch="aarch_64"; fi; \
    curl -fsSL -o /tmp/protobuf-javascript.tar.gz https://github.com/protocolbuffers/protobuf-javascript/releases/download/v3.21.4/protobuf-javascript-3.21.4-linux-${arch}.tar.gz \
 && cd /tmp \
 && tar zxf protobuf-javascript.tar.gz bin/protoc-gen-js \
 && rm -f protobuf-javascript.tar.gz

FROM gcr.io/distroless/cc-debian12:latest@sha256:3b75fdd33932d16e53a461277becf57c4f815c6cee5f6bc8f52457c095e004c8 AS base

FROM scratch
COPY --from=base --link / /
COPY --from=build --link --chmod=0755 /tmp/bin/protoc-gen-js .
USER nobody
ENTRYPOINT ["/protoc-gen-js"]
