# syntax=docker/dockerfile:1.4
FROM swift:5.6.2-focal AS build

ARG PLUGIN_VERSION
RUN test -n "${PLUGIN_VERSION}"
ENV PLUGIN_VERSION=${PLUGIN_VERSION}

WORKDIR /app
RUN ["/bin/bash", "-c", "git clone --depth 1 --branch ${PLUGIN_VERSION#v} https://github.com/apple/swift-protobuf --recursive"]
WORKDIR /app/swift-protobuf
RUN swift build -c release --static-swift-stdlib

FROM debian:bullseye-20220822-slim
WORKDIR /app
COPY --from=build /app/swift-protobuf/.build/release/protoc-gen-swift .
USER nobody
ENTRYPOINT [ "/app/protoc-gen-swift" ]
