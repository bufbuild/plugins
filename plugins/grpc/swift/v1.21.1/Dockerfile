# syntax=docker/dockerfile:1.4
FROM swift:5.9.2-jammy AS build

WORKDIR /app
RUN git clone --depth 1 --branch 1.21.1 https://github.com/grpc/grpc-swift --recursive
WORKDIR /app/grpc-swift
RUN swift build -c release --static-swift-stdlib --product protoc-gen-grpc-swift -Xlinker -s

FROM gcr.io/distroless/cc-debian12:latest@sha256:4049e8f163161818a52e028c3c110ee0ba9d71a14760ad2838aabba52b3f9782
COPY --from=build --link /app/grpc-swift/.build/release/protoc-gen-grpc-swift .
USER nobody
ENTRYPOINT [ "/protoc-gen-grpc-swift" ]
