# syntax=docker/dockerfile:1.19
FROM python:3.11.13-bookworm AS build
WORKDIR /app
RUN python -mvenv /app
ADD /requirements.txt requirements.txt
RUN . ./bin/activate \
 && pip install --no-cache-dir -r requirements.txt \
 && pip uninstall --yes pip setuptools \
 && rm -f requirements.txt bin/activate.fish bin/activate.csh bin/Activate.ps1 \
 && rm -f bin/protoc-gen-mypy \
 && ln -sf /usr/bin/python /app/bin/python

FROM gcr.io/distroless/python3-debian12:latest@sha256:afdfc6df9eb96cf65576d4d59af48c4b5dac29ad62066aee879f12b303298aac AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=build /app /app
USER nobody
ENTRYPOINT [ "/app/bin/protoc-gen-mypy_grpc" ]
