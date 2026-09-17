# syntax=docker/dockerfile:1.26
FROM debian:trixie-20260824@sha256:f324c7ff54321e8d9c588493a20244965938ce0aa50bbd1022d38010e9ffc4b1 AS build

ARG TARGETARCH

RUN apt-get update \
 && apt-get install -y curl
RUN arch=${TARGETARCH}; \
    if [ "${arch}" = "amd64" ]; then arch="x86_64"; elif [ "${arch}" = "arm64" ]; then arch="aarch_64"; fi; \
    curl -fsSL -o /tmp/protobuf-javascript.tar.gz https://github.com/protocolbuffers/protobuf-javascript/releases/download/v4.0.3/protobuf-javascript-4.0.3-linux-${arch}.tar.gz \
 && cd /tmp \
 && tar zxf protobuf-javascript.tar.gz bin/protoc-gen-js \
 && rm -f protobuf-javascript.tar.gz

FROM gcr.io/distroless/cc-debian13:latest@sha256:4594d59540d1948417f6ca2829ddd9294493a7c68b7528f4dd459de7f203a750 AS base

FROM scratch
COPY --from=base --link / /
COPY --from=build --link --chmod=0755 /tmp/bin/protoc-gen-js .
USER nobody
ENTRYPOINT ["/protoc-gen-js"]
