# syntax=docker/dockerfile:1.19
FROM swift:6.2.3-bookworm AS build

RUN apt-get update \
 && apt-get install -y libstdc++-12-dev
WORKDIR /app
RUN git clone --depth 1 --branch 1.27.1 https://github.com/grpc/grpc-swift --recursive
WORKDIR /app/grpc-swift
RUN swift build -c release --static-swift-stdlib --product protoc-gen-grpc-swift -Xlinker -s

FROM gcr.io/distroless/cc-debian12:latest@sha256:66d87e170bc2c5e2b8cf853501141c3c55b4e502b8677595c57534df54a68cc5 AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=build /app/grpc-swift/.build/release/protoc-gen-grpc-swift .
USER nobody
ENTRYPOINT [ "/protoc-gen-grpc-swift" ]
