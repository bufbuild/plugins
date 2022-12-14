# syntax=docker/dockerfile:1.4
FROM swift:5.7.1-focal AS build

WORKDIR /app
RUN git clone --depth 1 --branch 1.13.1 https://github.com/grpc/grpc-swift --recursive
WORKDIR /app/grpc-swift
RUN swift build -c release --static-swift-stdlib --product protoc-gen-grpc-swift -Xlinker -s

FROM gcr.io/distroless/cc-debian11
COPY --from=build --link /app/grpc-swift/.build/release/protoc-gen-grpc-swift .
USER nobody
ENTRYPOINT [ "/protoc-gen-grpc-swift" ]
