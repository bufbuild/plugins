# syntax=docker/dockerfile:1.4
FROM debian:bullseye-20221004 AS build

RUN apt-get update \
 && apt-get install -y git cmake build-essential

WORKDIR /build

RUN git clone --depth 1 --branch v1.49.1 --recursive https://github.com/grpc/grpc
WORKDIR /build/grpc
RUN cmake . \
 && cmake --build . --target plugins

FROM debian:bullseye-20221004-slim
COPY --from=build --link --chmod=0755 /build/grpc/grpc_objective_c_plugin .
USER nobody
ENTRYPOINT ["/grpc_objective_c_plugin"]
