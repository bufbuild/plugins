# syntax=docker/dockerfile:1.22
FROM python:3.11.15-bookworm AS build
WORKDIR /app
RUN python -mvenv /app
RUN . ./bin/activate \
 && pip install --no-cache-dir protoc-gen-connect-python==0.9.0 \
 && pip uninstall --yes pip setuptools \
 && rm -f bin/activate.fish bin/activate.csh bin/Activate.ps1

FROM gcr.io/distroless/python3-debian12:latest@sha256:e19b29c8473bed88f8b6e31e49d1133d1bdb2bf9c0af7e82d0e7434f7c036a63 AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=build --chmod=0755 /app /app
USER nobody
ENTRYPOINT [ "/app/bin/protoc-gen-connect-python" ]
