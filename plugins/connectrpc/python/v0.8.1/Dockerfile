# syntax=docker/dockerfile:1.19
FROM python:3.11.14-bookworm AS build
WORKDIR /app
RUN python -mvenv /app
RUN . ./bin/activate \
 && pip install --no-cache-dir protoc-gen-connect-python==0.8.1 \
 && pip uninstall --yes pip setuptools \
 && rm -f bin/activate.fish bin/activate.csh bin/Activate.ps1

FROM gcr.io/distroless/python3-debian12:latest@sha256:2971dbfa56eb4f2e33e94709579783406c1e7165b2d92edaa5657da5e8040617 AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=build --chmod=0755 /app /app
USER nobody
ENTRYPOINT [ "/app/bin/protoc-gen-connect-python" ]
