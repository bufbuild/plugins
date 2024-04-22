# syntax=docker/dockerfile:1.7
FROM swift:5.10.0-jammy AS build

WORKDIR /app
RUN git clone --depth 1 --branch 1.23.0 https://github.com/grpc/grpc-swift --recursive
WORKDIR /app/grpc-swift
RUN swift build -c release --static-swift-stdlib --product protoc-gen-grpc-swift -Xlinker -s

FROM gcr.io/distroless/cc-debian12:latest@sha256:7a01d633f75120af59c71489e0911fa8b6512673a3ff0b999522b4221ab4d86a
COPY --from=build --link /app/grpc-swift/.build/release/protoc-gen-grpc-swift .
USER nobody
ENTRYPOINT [ "/protoc-gen-grpc-swift" ]
