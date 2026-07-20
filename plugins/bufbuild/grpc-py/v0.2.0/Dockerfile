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

FROM gcr.io/distroless/python3-debian13:latest@sha256:d7658786080d0ad01686028fdf6f2fea1c15485ff48f74f3af80b8d46ba6fe39 AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=build --chmod=0755 /app /app
USER nobody
CMD [ "/app/bin/protoc-gen-grpc-py" ]
