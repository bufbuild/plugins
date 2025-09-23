# syntax=docker/dockerfile:1.18
FROM python:3.11.13-bookworm AS build
WORKDIR /app
RUN python -mvenv /app
RUN . ./bin/activate \
 && pip install --no-cache-dir protoc-gen-connect-python==0.5.0 \
 && pip uninstall --yes pip setuptools \
 && rm -f bin/activate.fish bin/activate.csh bin/Activate.ps1

FROM gcr.io/distroless/python3-debian12:latest@sha256:a0ded8676ed10e194cce1b8fdf9171672cda95ad1cabba4ece7b01f4db2e7471 AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=build --chmod=0755 /app /app
USER nobody
ENTRYPOINT [ "/app/bin/protoc-gen-connect-python" ]
