# syntax=docker/dockerfile:1.27
FROM swift:6.4.0-bookworm@sha256:7dac8560dab0523028235ce7aa79e59fdc68b861c6aca6c709f57c3187dd7f66 AS build

RUN apt-get update \
 && apt-get install -y libstdc++-12-dev unzip
COPY --link extramoduleimports.patch /app/extramoduleimports.patch
WORKDIR /app
RUN git clone --depth 1 --branch 1.39.0 https://github.com/apple/swift-protobuf --recursive
WORKDIR /app/swift-protobuf
RUN git apply /app/extramoduleimports.patch
RUN swift build -c release --build-system native --static-swift-stdlib -Xlinker -s

FROM gcr.io/distroless/cc-debian13:latest@sha256:159783207c2cd44c2aa5715961d13c8612368ac9bd450f887e3f08fc8ea461e3 AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=build /app/swift-protobuf/.build/release/protoc-gen-swift .
USER nobody
ENTRYPOINT [ "/protoc-gen-swift" ]
