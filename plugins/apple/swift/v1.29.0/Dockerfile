# syntax=docker/dockerfile:1.12
FROM swift:6.0.3-bookworm AS build

RUN apt-get update \
 && apt-get install -y libstdc++-12-dev
COPY --link extramoduleimports.patch /app/extramoduleimports.patch
WORKDIR /app
RUN git clone --depth 1 --branch 1.29.0 https://github.com/apple/swift-protobuf --recursive
WORKDIR /app/swift-protobuf
RUN git apply /app/extramoduleimports.patch
RUN swift build -c release --static-swift-stdlib -Xlinker -s

FROM gcr.io/distroless/cc-debian12:latest@sha256:b7550f0b15838de14c564337eef2b804ba593ae55d81ca855421bd52f19bb480 AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=build /app/swift-protobuf/.build/release/protoc-gen-swift .
USER nobody
ENTRYPOINT [ "/protoc-gen-swift" ]
