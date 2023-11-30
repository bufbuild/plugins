# syntax=docker/dockerfile:1.6

FROM debian:bookworm-20231120 AS build

RUN apt-get update \
 && apt-get install -y build-essential curl libprotobuf-dev libprotoc-dev pkg-config protobuf-compiler

WORKDIR /build
RUN curl -fsSL -o /tmp/protobuf-c.tar.gz https://github.com/protobuf-c/protobuf-c/releases/download/v1.5.0/protobuf-c-1.5.0.tar.gz \
 && tar zxf /tmp/protobuf-c.tar.gz \
 && cd protobuf-c-1.5.0 \
 && ./configure \
 && make LDFLAGS=-all-static -j$(nproc) \
 && make install \
 && strip /usr/local/bin/protoc-gen-c

FROM gcr.io/distroless/cc-debian12:latest@sha256:a9056d2232d16e3772bec3ef36b93a5ea9ef6ad4b4ed407631e534b85832cf40
COPY --from=build --link --chmod=0755 --chown=root:root /usr/local/bin/protoc-gen-c .
USER nobody
ENTRYPOINT ["/protoc-gen-c"]
