# syntax=docker/dockerfile:1.19
FROM debian:bookworm-20251117 AS build

ARG TARGETARCH

RUN apt-get update \
 && apt-get install -y curl
RUN arch=${TARGETARCH}; \
    if [ "${arch}" = "amd64" ]; then arch="x86_64"; elif [ "${arch}" = "arm64" ]; then arch="aarch_64"; fi; \
    curl -fsSL -o /tmp/protobuf-javascript.tar.gz https://github.com/protocolbuffers/protobuf-javascript/releases/download/v4.0.1/protobuf-javascript-4.0.1-linux-${arch}.tar.gz \
 && cd /tmp \
 && tar zxf protobuf-javascript.tar.gz bin/protoc-gen-js \
 && rm -f protobuf-javascript.tar.gz

FROM gcr.io/distroless/cc-debian12:latest@sha256:0000f9dc0290f8eaf0ecceafbc35e803649087ea7879570fbc78372df7ac649b AS base

FROM scratch
COPY --from=base --link / /
COPY --from=build --link --chmod=0755 /tmp/bin/protoc-gen-js .
USER nobody
ENTRYPOINT ["/protoc-gen-js"]
