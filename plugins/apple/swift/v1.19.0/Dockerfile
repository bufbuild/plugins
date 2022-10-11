# syntax=docker/dockerfile:1.4
FROM swift:5.7.0-focal AS build

WORKDIR /app
RUN git clone --depth 1 --branch 1.19.0 https://github.com/apple/swift-protobuf --recursive
WORKDIR /app/swift-protobuf
RUN swift build -c release --static-swift-stdlib

FROM debian:bullseye-20221004-slim
COPY --from=build --link /app/swift-protobuf/.build/release/protoc-gen-swift .
USER nobody
ENTRYPOINT [ "/protoc-gen-swift" ]
