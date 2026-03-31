# syntax=docker/dockerfile:1.22
FROM swift:6.3.0-bookworm@sha256:943bb41624442cef8ccca256f1ff7830ce25209cbad88ebfede0aaef268a8a80 AS build

WORKDIR /app
RUN apt-get update \
 && apt-get install -y libstdc++-12-dev
RUN git clone --depth 1 --branch 1.2.2 https://github.com/connectrpc/connect-swift
WORKDIR /app/connect-swift
RUN swift build -c release --product protoc-gen-connect-swift-mocks --static-swift-stdlib -Xlinker -s

FROM gcr.io/distroless/cc-debian13:latest@sha256:e1cc90d06703f5dc30ae869fbfce78fce688f21a97efecd226375233a882e62f AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=build /app/connect-swift/.build/release/protoc-gen-connect-swift-mocks .
USER nobody
ENTRYPOINT [ "/protoc-gen-connect-swift-mocks" ]
