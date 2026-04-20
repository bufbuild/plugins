# syntax=docker/dockerfile:1.23
FROM swift:6.3.0-bookworm@sha256:6caf4864fc41f2162aee7189d8e4d6cd79e799540b6b197d0bcf396bf34b94df AS build

RUN apt-get update \
 && apt-get install -y libstdc++-12-dev unzip
COPY --link extramoduleimports.patch /app/extramoduleimports.patch
WORKDIR /app
RUN git clone --depth 1 --branch 1.37.0 https://github.com/apple/swift-protobuf --recursive
WORKDIR /app/swift-protobuf
RUN git apply /app/extramoduleimports.patch
RUN swift build -c release --static-swift-stdlib -Xlinker -s

FROM gcr.io/distroless/cc-debian13:latest@sha256:56aaf20ab2523a346a67c8e8f8e8dabe447447d0788b82284d14ad79cd5f93cc AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=build /app/swift-protobuf/.build/release/protoc-gen-swift .
USER nobody
ENTRYPOINT [ "/protoc-gen-swift" ]
