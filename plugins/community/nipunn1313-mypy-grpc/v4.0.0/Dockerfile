# syntax=docker/dockerfile:1.19
FROM python:3.11.14-bookworm AS build
WORKDIR /app
RUN python -mvenv /app
ADD /requirements.txt requirements.txt
RUN . ./bin/activate \
 && pip install --no-cache-dir -r requirements.txt \
 && pip uninstall --yes pip setuptools \
 && rm -f requirements.txt bin/activate.fish bin/activate.csh bin/Activate.ps1 \
 && rm -f bin/protoc-gen-mypy \
 && ln -sf /usr/bin/python /app/bin/python

FROM gcr.io/distroless/python3-debian12:latest@sha256:8ce6bba3f793ba7d834467dfe18983c42f9b223604970273e9e3a22b1891fc27 AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=build /app /app
USER nobody
ENTRYPOINT [ "/app/bin/protoc-gen-mypy_grpc" ]
