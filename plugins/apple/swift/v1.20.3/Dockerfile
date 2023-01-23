# syntax=docker/dockerfile:1.4
FROM swift:5.7.3-focal AS build

WORKDIR /app
RUN git clone --depth 1 --branch 1.20.3 https://github.com/apple/swift-protobuf --recursive
WORKDIR /app/swift-protobuf
RUN swift build -c release --static-swift-stdlib -Xlinker -s

FROM gcr.io/distroless/cc-debian11
COPY --from=build --link /app/swift-protobuf/.build/release/protoc-gen-swift .
USER nobody
ENTRYPOINT [ "/protoc-gen-swift" ]
