# syntax=docker/dockerfile:1.17
FROM swift:6.1.2-bookworm AS build

RUN apt-get update \
 && apt-get install -y unzip
WORKDIR /app
RUN git clone --depth 1 --branch 2.1.1 https://github.com/grpc/grpc-swift-protobuf --recursive
WORKDIR /app/grpc-swift-protobuf
COPY --link Package.resolved .
RUN swift build -c release --static-swift-stdlib --product protoc-gen-grpc-swift-2 -Xlinker -s --force-resolved-versions

FROM gcr.io/distroless/cc-debian12:latest@sha256:00cc20b928afcc8296b72525fa68f39ab332f758c4f2a9e8d90845d3e06f1dc4 AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=build /app/grpc-swift-protobuf/.build/release/protoc-gen-grpc-swift-2 .
USER nobody
ENTRYPOINT [ "/protoc-gen-grpc-swift-2" ]
