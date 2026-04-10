# syntax=docker/dockerfile:1.19
FROM swift:6.2.4-bookworm AS build

RUN apt-get update \
 && apt-get install -y unzip
WORKDIR /app
RUN git clone --depth 1 --branch 2.2.1 https://github.com/grpc/grpc-swift-protobuf --recursive
WORKDIR /app/grpc-swift-protobuf
COPY --link Package.resolved .
RUN swift package resolve --force-resolved-versions
RUN swift build -c release --static-swift-stdlib --product protoc-gen-grpc-swift-2 -Xlinker -s --force-resolved-versions

FROM gcr.io/distroless/cc-debian12:latest@sha256:329e54034ce498f9c6b345044e8f530c6691f99e94a92446f68c0adf9baa8464 AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=build /app/grpc-swift-protobuf/.build/release/protoc-gen-grpc-swift-2 .
USER nobody
ENTRYPOINT [ "/protoc-gen-grpc-swift-2" ]
