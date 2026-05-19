# syntax=docker/dockerfile:1.23
FROM swift:6.3.2-bookworm@sha256:40918972df286d15aa47f4686c98a211f0a6152ca1d48b3dbccd6f56a3c34815 AS build

RUN apt-get update \
 && apt-get install -y unzip
WORKDIR /app
RUN git clone --depth 1 --branch 2.4.0 https://github.com/grpc/grpc-swift-protobuf --recursive
WORKDIR /app/grpc-swift-protobuf
COPY --link Package.resolved .
RUN swift build -c release --static-swift-stdlib --product protoc-gen-grpc-swift-2 -Xlinker -s --force-resolved-versions

FROM gcr.io/distroless/cc-debian13:latest@sha256:8b5d1db6d2253036a53cb8362d3e3fa82a7caf84c247772c46a023166c64e977 AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=build /app/grpc-swift-protobuf/.build/release/protoc-gen-grpc-swift-2 .
USER nobody
ENTRYPOINT [ "/protoc-gen-grpc-swift-2" ]
