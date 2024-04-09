# syntax=docker/dockerfile:1.6
FROM swift:5.10.0-jammy AS build

WORKDIR /app
RUN git clone --depth 1 --branch 0.13.0 https://github.com/connectrpc/connect-swift
WORKDIR /app/connect-swift
RUN swift build -c release --product protoc-gen-connect-swift --static-swift-stdlib -Xlinker -s

FROM gcr.io/distroless/cc-debian12:latest@sha256:7a01d633f75120af59c71489e0911fa8b6512673a3ff0b999522b4221ab4d86a
COPY --from=build --link /app/connect-swift/.build/release/protoc-gen-connect-swift .
USER nobody
ENTRYPOINT [ "/protoc-gen-connect-swift" ]
