# syntax=docker/dockerfile:1.4
FROM dart:3.5.4-sdk AS build

WORKDIR /build
RUN curl -fsSL https://github.com/connectrpc/connect-dart/archive/refs/tags/v0.1.0.tar.gz --compressed -o connect-dart.tar.gz \
 && tar -xvf connect-dart.tar.gz \
 && cd connect-dart-0.1.0/packages/connect \
 && dart pub get \ 
 && dart compile exe bin/protoc-gen-connect-dart.dart -o /build/protoc-gen-connect-dart

FROM scratch
COPY --from=build --link /etc/passwd /etc/passwd
COPY --from=build --link /runtime/ /
COPY --from=build --link /build/protoc-gen-connect-dart .
USER nobody
ENTRYPOINT [ "/protoc-gen-connect-dart" ]
