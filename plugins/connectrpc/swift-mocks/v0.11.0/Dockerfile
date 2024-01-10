# syntax=docker/dockerfile:1.6
FROM swift:5.9.2-jammy AS build

WORKDIR /app
RUN git clone --depth 1 --branch 0.11.0 https://github.com/connectrpc/connect-swift
WORKDIR /app/connect-swift
RUN swift build -c release --product protoc-gen-connect-swift-mocks --static-swift-stdlib -Xlinker -s

FROM gcr.io/distroless/cc-debian12:latest@sha256:6714977f9f02632c31377650c15d89a7efaebf43bab0f37c712c30fc01edb973
COPY --from=build --link /app/connect-swift/.build/release/protoc-gen-connect-swift-mocks .
USER nobody
ENTRYPOINT [ "/protoc-gen-connect-swift-mocks" ]
