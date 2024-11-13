# syntax=docker/dockerfile:1.10
FROM swift:5.10.1-bookworm AS build

WORKDIR /app
RUN apt-get update \
 && apt-get install -y libstdc++-12-dev
RUN git clone --depth 1 --branch 1.0.0 https://github.com/connectrpc/connect-swift
WORKDIR /app/connect-swift
RUN swift build -c release --product protoc-gen-connect-swift --static-swift-stdlib -Xlinker -s

FROM gcr.io/distroless/cc-debian12:latest@sha256:6f05aba4de16e89f8d879bf2a1364de3e41aba04f1dcbba8c75494f6134b4b13 AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=build /app/connect-swift/.build/release/protoc-gen-connect-swift .
USER nobody
ENTRYPOINT [ "/protoc-gen-connect-swift" ]
