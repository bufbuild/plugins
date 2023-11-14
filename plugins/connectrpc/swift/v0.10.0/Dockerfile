# syntax=docker/dockerfile:1.6
FROM swift:5.9.1-focal AS build

WORKDIR /app
RUN git clone --depth 1 --branch 0.10.0 https://github.com/connectrpc/connect-swift
WORKDIR /app/connect-swift
RUN swift build -c release --product protoc-gen-connect-swift --static-swift-stdlib -Xlinker -s

FROM gcr.io/distroless/cc-debian11
COPY --from=build --link /app/connect-swift/.build/release/protoc-gen-connect-swift .
USER nobody
ENTRYPOINT [ "/protoc-gen-connect-swift" ]
