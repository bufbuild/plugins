# syntax=docker/dockerfile:1.15
FROM swift:6.1.2-bookworm AS build

WORKDIR /app
RUN git clone --depth 1 --branch 2.0.0 https://github.com/grpc/grpc-swift-protobuf --recursive
WORKDIR /app/grpc-swift-protobuf
RUN swift build -c release --static-swift-stdlib --product protoc-gen-grpc-swift-2 -Xlinker -s

FROM gcr.io/distroless/cc-debian12:latest@sha256:c53c9416a1acdbfd6e09abba720442444a3d1a6338b8db850e5e198b59af5570 AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=build /app/grpc-swift-protobuf/.build/release/protoc-gen-grpc-swift-2 .
USER nobody
ENTRYPOINT [ "/protoc-gen-grpc-swift-2" ]
