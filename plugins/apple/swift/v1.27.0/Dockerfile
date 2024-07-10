# syntax=docker/dockerfile:1.8
FROM swift:5.10.1-bookworm AS build

RUN apt-get update \
 && apt-get install -y libstdc++-12-dev
COPY --link extramoduleimports.patch /app/extramoduleimports.patch
WORKDIR /app
RUN git clone --depth 1 --branch 1.27.0 https://github.com/apple/swift-protobuf --recursive
WORKDIR /app/swift-protobuf
RUN git apply /app/extramoduleimports.patch
RUN swift build -c release --static-swift-stdlib -Xlinker -s

FROM gcr.io/distroless/cc-debian12:latest@sha256:3b75fdd33932d16e53a461277becf57c4f815c6cee5f6bc8f52457c095e004c8 AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=build /app/swift-protobuf/.build/release/protoc-gen-swift .
USER nobody
ENTRYPOINT [ "/protoc-gen-swift" ]
