# syntax=docker/dockerfile:1.4
ARG DOCKER_ORG=bufbuild
ARG BASE_BUILD_VERSION=latest
FROM ${DOCKER_ORG}/plugins-grpc-base-build:${BASE_BUILD_VERSION}

ARG VERSION
RUN test -n "${VERSION}"

RUN git clone --depth 1 --branch ${VERSION} --recursive https://github.com/grpc/grpc
WORKDIR /build/grpc
RUN cmake . \
 && cmake --build . --target plugins
