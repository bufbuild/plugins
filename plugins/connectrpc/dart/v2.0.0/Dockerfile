# syntax=docker/dockerfile:1.26
FROM dart:3.13.2-sdk@sha256:1f86408456fbcdc5f9c33fa267d3680d86e79c255ec338b9a215459224769770 AS build

WORKDIR /build
RUN git clone --depth=1 --branch v2.0.0 https://github.com/connectrpc/connect-dart.git
RUN cd connect-dart/packages/connect \
 && dart pub get \ 
 && dart compile exe bin/protoc-gen-connect-dart.dart -o /build/protoc-gen-connect-dart

FROM scratch
COPY --from=build --link /etc/passwd /etc/passwd
COPY --from=build --link /runtime/ /
COPY --from=build --link /build/protoc-gen-connect-dart .
USER nobody
ENTRYPOINT [ "/protoc-gen-connect-dart" ]
