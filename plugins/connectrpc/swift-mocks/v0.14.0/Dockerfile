# syntax=docker/dockerfile:1.7
FROM swift:5.10.1-jammy AS build

WORKDIR /app
RUN git clone --depth 1 --branch 0.14.0 https://github.com/connectrpc/connect-swift
WORKDIR /app/connect-swift
RUN swift build -c release --product protoc-gen-connect-swift-mocks --static-swift-stdlib -Xlinker -s

FROM gcr.io/distroless/cc-debian12:latest@sha256:e1065a1d58800a7294f74e67c32ec4146d09d6cbe471c1fa7ed456b2d2bf06e0 AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=build /app/connect-swift/.build/release/protoc-gen-connect-swift-mocks .
USER nobody
ENTRYPOINT [ "/protoc-gen-connect-swift-mocks" ]
