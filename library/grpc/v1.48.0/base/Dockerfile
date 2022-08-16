# syntax=docker/dockerfile:1.4
FROM bufbuild/plugins-grpc-base-build:latest

ARG GRPC_VERSION
RUN test -n "${GRPC_VERSION}"

RUN git clone --depth 1 --branch ${GRPC_VERSION} --recursive https://github.com/grpc/grpc
WORKDIR /build/grpc
RUN cmake . \
 && cmake --build . --target plugins
