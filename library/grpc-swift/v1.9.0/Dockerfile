# syntax=docker/dockerfile:1.4
FROM swift:5.6.2-focal AS build

ARG PLUGIN_VERSION
RUN test -n "${PLUGIN_VERSION}"
ENV PLUGIN_VERSION=${PLUGIN_VERSION}

WORKDIR /app
RUN git clone --depth 1 --branch ${PLUGIN_VERSION#v} https://github.com/grpc/grpc-swift --recursive
WORKDIR /app/grpc-swift
RUN swift build -c release --static-swift-stdlib --product protoc-gen-grpc-swift

FROM debian:bullseye-20220822-slim
WORKDIR /app
COPY --from=build /app/grpc-swift/.build/release/protoc-gen-grpc-swift .
USER nobody
ENTRYPOINT [ "/app/protoc-gen-grpc-swift" ]
