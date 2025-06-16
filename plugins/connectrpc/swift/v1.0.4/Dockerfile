# syntax=docker/dockerfile:1.15
FROM swift:6.1.2-bookworm AS build

WORKDIR /app
RUN apt-get update \
 && apt-get install -y libstdc++-12-dev
RUN git clone --depth 1 --branch 1.0.4 https://github.com/connectrpc/connect-swift
WORKDIR /app/connect-swift
RUN swift build -c release --product protoc-gen-connect-swift --static-swift-stdlib -Xlinker -s

FROM gcr.io/distroless/cc-debian12:latest@sha256:eccec5274132c1be0ce5d2c8e6fe41033e64af5e987ccee9007826e4c012069d AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=build /app/connect-swift/.build/release/protoc-gen-connect-swift .
USER nobody
ENTRYPOINT [ "/protoc-gen-connect-swift" ]
