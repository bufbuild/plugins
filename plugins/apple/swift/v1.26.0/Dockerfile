# syntax=docker/dockerfile:1.7
FROM swift:5.10.0-jammy AS build

COPY extramoduleimports.patch /app/extramoduleimports.patch
WORKDIR /app
RUN git clone --depth 1 --branch 1.26.0 https://github.com/apple/swift-protobuf --recursive
WORKDIR /app/swift-protobuf
RUN git apply /app/extramoduleimports.patch
RUN swift build -c release --static-swift-stdlib -Xlinker -s

FROM gcr.io/distroless/cc-debian12:latest@sha256:e6ae66a5a343d7112167f9117c4e630cfffcd80db44e44302759ec13ddd2d22b
COPY --from=build --link /app/swift-protobuf/.build/release/protoc-gen-swift .
USER nobody
ENTRYPOINT [ "/protoc-gen-swift" ]
