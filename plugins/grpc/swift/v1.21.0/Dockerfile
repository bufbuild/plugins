# syntax=docker/dockerfile:1.4
FROM swift:5.9.1-jammy AS build

WORKDIR /app
RUN git clone --depth 1 --branch 1.21.0 https://github.com/grpc/grpc-swift --recursive
WORKDIR /app/grpc-swift
RUN swift build -c release --static-swift-stdlib --product protoc-gen-grpc-swift -Xlinker -s

FROM gcr.io/distroless/cc-debian12:latest@sha256:a9056d2232d16e3772bec3ef36b93a5ea9ef6ad4b4ed407631e534b85832cf40
COPY --from=build --link /app/grpc-swift/.build/release/protoc-gen-grpc-swift .
USER nobody
ENTRYPOINT [ "/protoc-gen-grpc-swift" ]
