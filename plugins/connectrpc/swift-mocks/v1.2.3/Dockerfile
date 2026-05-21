# syntax=docker/dockerfile:1.23
FROM swift:6.3.2-bookworm@sha256:de037fec606b26cee2bf7f93dba8eab7a617f067f94cf78669f929e5a0c6146a AS build

WORKDIR /app
RUN apt-get update \
 && apt-get install -y libstdc++-12-dev
RUN git clone --depth 1 --branch 1.2.3 https://github.com/connectrpc/connect-swift
WORKDIR /app/connect-swift
RUN swift build -c release --product protoc-gen-connect-swift-mocks --static-swift-stdlib -Xlinker -s

FROM gcr.io/distroless/cc-debian13:latest@sha256:8b5d1db6d2253036a53cb8362d3e3fa82a7caf84c247772c46a023166c64e977 AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=build /app/connect-swift/.build/release/protoc-gen-connect-swift-mocks .
USER nobody
ENTRYPOINT [ "/protoc-gen-connect-swift-mocks" ]
