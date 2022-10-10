# syntax=docker/dockerfile:1.4
FROM dart:2.18.2-sdk as build

WORKDIR /build
RUN curl -fsSL https://pub.dartlang.org/packages/protoc_plugin/versions/20.0.1.tar.gz --compressed -o protoc_plugin.tar.gz \
 && tar -xvf protoc_plugin.tar.gz \
 && dart pub get \
 && dart compile exe bin/protoc_plugin.dart -o protoc-gen-dart

FROM scratch
COPY --from=build --link /etc/passwd /etc/passwd
COPY --from=build --link /runtime/ /
COPY --from=build --link /build/protoc-gen-dart .
USER nobody
ENTRYPOINT [ "/protoc-gen-dart" ]
