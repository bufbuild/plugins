# syntax=docker/dockerfile:1.4
FROM dart:2.17.6-sdk as build

ARG PLUGIN_VERSION
RUN test -n "${PLUGIN_VERSION}"
ENV PLUGIN_VERSION=${PLUGIN_VERSION}

WORKDIR /build
RUN ["/bin/bash", "-c", "curl -fsSL https://pub.dartlang.org/packages/protoc_plugin/versions/${PLUGIN_VERSION#v}.tar.gz --compressed -o protoc_plugin.tar.gz"]
RUN tar -xvf protoc_plugin.tar.gz
RUN dart pub get
RUN dart compile exe bin/protoc_plugin.dart -o protoc-gen-dart

FROM scratch
WORKDIR /build
COPY --from=build /etc/passwd /etc/passwd
COPY --from=build /runtime/ /
COPY --from=build /build/protoc-gen-dart .
USER nobody
ENTRYPOINT [ "./protoc-gen-dart" ]
