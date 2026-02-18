# syntax=docker/dockerfile:1.19
FROM swift:6.2.3-bookworm AS build

WORKDIR /app
RUN apt-get update \
 && apt-get install -y libstdc++-12-dev
RUN git clone --depth 1 --branch 1.2.1 https://github.com/connectrpc/connect-swift
WORKDIR /app/connect-swift
RUN swift build -c release --product protoc-gen-connect-swift --static-swift-stdlib -Xlinker -s

FROM gcr.io/distroless/cc-debian12:latest@sha256:329e54034ce498f9c6b345044e8f530c6691f99e94a92446f68c0adf9baa8464 AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=build /app/connect-swift/.build/release/protoc-gen-connect-swift .
USER nobody
ENTRYPOINT [ "/protoc-gen-connect-swift" ]
