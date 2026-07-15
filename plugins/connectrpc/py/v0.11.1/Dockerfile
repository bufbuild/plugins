# syntax=docker/dockerfile:1.24
FROM python:3.13.14-trixie@sha256:c68b61ac211bc17bda0dfabff372246fd7dbb7dceaeabdb01cd6042a197c53c2 AS build
WORKDIR /app
RUN python -mvenv /app
ADD /requirements.txt requirements.txt
RUN . ./bin/activate \
 && pip install --no-cache-dir -r requirements.txt \
 && pip uninstall --yes pip setuptools \
 && rm -f requirements.txt bin/activate.fish bin/activate.csh bin/Activate.ps1 \
 && ln -sf /usr/bin/python /app/bin/python

 FROM gcr.io/distroless/python3-debian13:latest@sha256:178dd00f2da3271f3819df5cd327472754946c7430d82197b247e95e839a3d55 AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=build --chmod=0755 /app /app
USER nobody
ENTRYPOINT [ "/app/bin/protoc-gen-connectrpc" ]
