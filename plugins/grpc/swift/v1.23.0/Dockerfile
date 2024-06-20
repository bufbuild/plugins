# syntax=docker/dockerfile:1.7
FROM swift:5.10.1-bookworm AS build

RUN apt-get update \
 && apt-get install -y libstdc++-12-dev
WORKDIR /app
RUN git clone --depth 1 --branch 1.23.0 https://github.com/grpc/grpc-swift --recursive
WORKDIR /app/grpc-swift
RUN swift build -c release --static-swift-stdlib --product protoc-gen-grpc-swift -Xlinker -s

FROM gcr.io/distroless/cc-debian12:latest@sha256:e1065a1d58800a7294f74e67c32ec4146d09d6cbe471c1fa7ed456b2d2bf06e0 AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=build /app/grpc-swift/.build/release/protoc-gen-grpc-swift .
USER nobody
ENTRYPOINT [ "/protoc-gen-grpc-swift" ]
