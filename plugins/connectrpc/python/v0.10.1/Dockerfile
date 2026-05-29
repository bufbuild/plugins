# syntax=docker/dockerfile:1.24
FROM python:3.13.13-trixie@sha256:e6220f3ff717d352ea275569e0aea7a4d18e8cd681075ae195125969b0e163f4 AS build
WORKDIR /app
RUN python -mvenv /app
RUN . ./bin/activate \
 && pip install --no-cache-dir protoc-gen-connectrpc==0.10.1 \
 && pip uninstall --yes pip setuptools \
 && rm -f bin/activate.fish bin/activate.csh bin/Activate.ps1

FROM gcr.io/distroless/python3-debian13:latest@sha256:0e300647f5a9d51fb686e9167c97248e0419cd6e5186efc50b642748aab8d8be AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=build --chmod=0755 /app /app
USER nobody
ENTRYPOINT [ "/app/bin/protoc-gen-connectrpc" ]
