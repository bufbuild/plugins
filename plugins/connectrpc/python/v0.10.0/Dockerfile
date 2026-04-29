# syntax=docker/dockerfile:1.23
FROM python:3.13.13-trixie@sha256:d52f7ef162003e17330f0dd9c3e0b4ac6775b39ea988be0c23de6d1461191569 AS build
WORKDIR /app
RUN python -mvenv /app
RUN . ./bin/activate \
 && pip install --no-cache-dir protoc-gen-connectrpc==0.10.0 \
 && pip uninstall --yes pip setuptools \
 && rm -f bin/activate.fish bin/activate.csh bin/Activate.ps1

FROM gcr.io/distroless/python3-debian13:latest@sha256:ed3a4beb46f8f8baac068743ba1b1f95ea3f793422129cf6dd23967f779b6018 AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=build --chmod=0755 /app /app
USER nobody
ENTRYPOINT [ "/app/bin/protoc-gen-connectrpc" ]
