# syntax=docker/dockerfile:1.12

FROM debian:bookworm-20250317 AS build

RUN apt-get update \
 && apt-get install -y build-essential curl libprotobuf-dev libprotoc-dev pkg-config protobuf-compiler

WORKDIR /build
RUN curl -fsSL -o /tmp/protobuf-c.tar.gz https://github.com/protobuf-c/protobuf-c/releases/download/v1.5.2/protobuf-c-1.5.2.tar.gz \
 && tar zxf /tmp/protobuf-c.tar.gz \
 && cd protobuf-c-1.5.2 \
 && ./configure \
 && make LDFLAGS=-all-static -j$(nproc) \
 && make install \
 && strip /usr/local/bin/protoc-gen-c

FROM gcr.io/distroless/cc-debian12:latest@sha256:c1cbcec08d39c81adbefb80cabc51cba285465866f7b5ab15ddb2fcae51a1aed AS base

FROM scratch
COPY --from=base --link / /
COPY --from=build --link --chmod=0755 --chown=root:root /usr/local/bin/protoc-gen-c .
USER nobody
ENTRYPOINT ["/protoc-gen-c"]
