# syntax=docker/dockerfile:1.15
FROM dart:3.7.3-sdk AS build

WORKDIR /build
RUN git clone --depth=1 --branch v0.4.0 https://github.com/connectrpc/connect-dart.git
RUN cd connect-dart/packages/connect \
 && dart pub get \ 
 && dart compile exe bin/protoc-gen-connect-dart.dart -o /build/protoc-gen-connect-dart

FROM scratch
COPY --from=build --link /etc/passwd /etc/passwd
COPY --from=build --link /runtime/ /
COPY --from=build --link /build/protoc-gen-connect-dart .
USER nobody
ENTRYPOINT [ "/protoc-gen-connect-dart" ]
