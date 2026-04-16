# syntax=docker/dockerfile:1.23
FROM swift:6.3.0-bookworm@sha256:6caf4864fc41f2162aee7189d8e4d6cd79e799540b6b197d0bcf396bf34b94df AS build

RUN apt-get update \
 && apt-get install -y unzip
WORKDIR /app
RUN git clone --depth 1 --branch 2.3.0 https://github.com/grpc/grpc-swift-protobuf --recursive
WORKDIR /app/grpc-swift-protobuf
COPY --link Package.resolved .
RUN swift build -c release --static-swift-stdlib --product protoc-gen-grpc-swift-2 -Xlinker -s --force-resolved-versions

FROM gcr.io/distroless/cc-debian13:latest@sha256:56aaf20ab2523a346a67c8e8f8e8dabe447447d0788b82284d14ad79cd5f93cc AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=build /app/grpc-swift-protobuf/.build/release/protoc-gen-grpc-swift-2 .
USER nobody
ENTRYPOINT [ "/protoc-gen-grpc-swift-2" ]
