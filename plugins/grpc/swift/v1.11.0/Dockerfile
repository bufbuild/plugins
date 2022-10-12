# syntax=docker/dockerfile:1.4
FROM swift:5.7.0-focal AS build

WORKDIR /app
RUN git clone --depth 1 --branch 1.11.0 https://github.com/grpc/grpc-swift --recursive
WORKDIR /app/grpc-swift
RUN swift build -c release --static-swift-stdlib --product protoc-gen-grpc-swift

FROM debian:bullseye-20221004-slim
COPY --from=build --link /app/grpc-swift/.build/release/protoc-gen-grpc-swift .
USER nobody
ENTRYPOINT [ "/protoc-gen-grpc-swift" ]
