# syntax=docker/dockerfile:1.4
ARG DOCKER_ORG=bufbuild
ARG BASE_BUILD_VERSION=latest
FROM ${DOCKER_ORG}/plugins-grpc-base-build:${BASE_BUILD_VERSION}

RUN git clone --depth 1 --branch v1.49.1 --recursive https://github.com/grpc/grpc
WORKDIR /build/grpc
RUN cmake . \
 && cmake --build . --target plugins
