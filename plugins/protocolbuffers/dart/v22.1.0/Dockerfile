# syntax=docker/dockerfile:1.15
FROM dart:3.7.3-sdk AS build

WORKDIR /build
RUN curl -fsSL https://pub.dartlang.org/packages/protoc_plugin/versions/22.1.0.tar.gz --compressed -o protoc_plugin.tar.gz \
 && tar -xvf protoc_plugin.tar.gz \
 && dart pub get \
 && dart compile exe bin/protoc_plugin.dart -o protoc-gen-dart

FROM scratch
COPY --from=build --link /etc/passwd /etc/passwd
COPY --from=build --link /runtime/ /
COPY --from=build --link /build/protoc-gen-dart .
USER nobody
ENTRYPOINT [ "/protoc-gen-dart" ]
