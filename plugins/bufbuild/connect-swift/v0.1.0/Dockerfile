# syntax=docker/dockerfile:1.4
FROM swift:5.7.1-focal AS build

WORKDIR /app
RUN git clone --branch v0.1.0 https://github.com/bufbuild/connect-swift
WORKDIR /app/connect-swift
RUN swift build -c release --product protoc-gen-connect-swift --static-swift-stdlib -Xlinker -s

FROM gcr.io/distroless/cc-debian11
COPY --from=build --link /app/connect-swift/.build/release/protoc-gen-connect-swift .
USER nobody
ENTRYPOINT [ "/protoc-gen-connect-swift" ]
