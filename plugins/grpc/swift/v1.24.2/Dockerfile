# syntax=docker/dockerfile:1.10
FROM swift:5.10.1-bookworm AS build

RUN apt-get update \
 && apt-get install -y libstdc++-12-dev
WORKDIR /app
RUN git clone --depth 1 --branch 1.24.2 https://github.com/grpc/grpc-swift --recursive
WORKDIR /app/grpc-swift
RUN swift build -c release --static-swift-stdlib --product protoc-gen-grpc-swift -Xlinker -s

FROM gcr.io/distroless/cc-debian12:latest@sha256:6f05aba4de16e89f8d879bf2a1364de3e41aba04f1dcbba8c75494f6134b4b13 AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=build /app/grpc-swift/.build/release/protoc-gen-grpc-swift .
USER nobody
ENTRYPOINT [ "/protoc-gen-grpc-swift" ]
