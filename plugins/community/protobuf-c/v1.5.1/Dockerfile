# syntax=docker/dockerfile:1.12

FROM debian:bookworm-20250113 AS build

RUN apt-get update \
 && apt-get install -y build-essential curl libprotobuf-dev libprotoc-dev pkg-config protobuf-compiler

WORKDIR /build
RUN curl -fsSL -o /tmp/protobuf-c.tar.gz https://github.com/protobuf-c/protobuf-c/releases/download/v1.5.1/protobuf-c-1.5.1.tar.gz \
 && tar zxf /tmp/protobuf-c.tar.gz \
 && cd protobuf-c-1.5.1 \
 && ./configure \
 && make LDFLAGS=-all-static -j$(nproc) \
 && make install \
 && strip /usr/local/bin/protoc-gen-c

FROM gcr.io/distroless/cc-debian12:latest@sha256:b7550f0b15838de14c564337eef2b804ba593ae55d81ca855421bd52f19bb480
COPY --from=build --link --chmod=0755 --chown=root:root /usr/local/bin/protoc-gen-c .
USER nobody
ENTRYPOINT ["/protoc-gen-c"]
