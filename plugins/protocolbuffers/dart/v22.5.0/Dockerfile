# syntax=docker/dockerfile:1.15
FROM dart:3.8.2-sdk AS build

WORKDIR /build
RUN git clone --depth 1 --branch protoc_plugin-v22.5.0 https://github.com/google/protobuf.dart.git \
 && cd protobuf.dart/protoc_plugin \
 && dart pub get \
 && dart compile exe bin/protoc_plugin.dart -o /build/protoc-gen-dart

FROM scratch
COPY --from=build --link /etc/passwd /etc/passwd
COPY --from=build --link /runtime/ /
COPY --from=build --link /build/protoc-gen-dart .
USER nobody
ENTRYPOINT [ "/protoc-gen-dart" ]
