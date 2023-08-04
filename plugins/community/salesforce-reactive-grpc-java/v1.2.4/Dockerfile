FROM debian:bullseye-20230202 AS build

RUN apt-get update && apt-get upgrade -y && apt-get install -y wget openjdk-11-jre-headless
RUN case $(uname -m) in "aarch64") echo linux-aarch_64 ;; "x86_64") echo linux-x86_64;; esac >arch.txt
RUN wget -O protoc-gen-reactor-grpc https://repo1.maven.org/maven2/com/salesforce/servicelibs/reactor-grpc/1.2.4/reactor-grpc-1.2.4-$(cat arch.txt).exe
RUN chmod 0755 protoc-gen-reactor-grpc
USER nobody
CMD ./protoc-gen-reactor-grpc
