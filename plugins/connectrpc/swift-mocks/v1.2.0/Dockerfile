# syntax=docker/dockerfile:1.19
FROM swift:6.2.0-bookworm AS build

WORKDIR /app
RUN apt-get update \
 && apt-get install -y libstdc++-12-dev
RUN git clone --depth 1 --branch 1.2.0 https://github.com/connectrpc/connect-swift
WORKDIR /app/connect-swift
RUN swift build -c release --product protoc-gen-connect-swift-mocks --static-swift-stdlib -Xlinker -s

FROM gcr.io/distroless/cc-debian12:latest@sha256:14f6999db515330e5d00537bd457289a8968b6456e9197c7a28101ee63a7522f AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=build /app/connect-swift/.build/release/protoc-gen-connect-swift-mocks .
USER nobody
ENTRYPOINT [ "/protoc-gen-connect-swift-mocks" ]
