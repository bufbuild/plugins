# syntax=docker/dockerfile:1.10
FROM python:3.11.10-alpine3.20 AS build
WORKDIR /app
RUN python -mvenv /app
ADD /requirements.txt requirements.txt
RUN source ./bin/activate \
    && pip install --no-cache-dir -r requirements.txt \
    && pip uninstall --yes pip \
    && rm -f requirements.txt bin/activate.fish bin/activate.csh bin/Activate.ps1 \
    && ln -sf /usr/bin/python /app/bin/python

FROM gcr.io/distroless/python3-debian12:latest@sha256:55283e240bb5bd506e711a395092b61acf1fb14e63977ee24336285dd9e68fb8 AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=build --chmod=0755 /app /app
USER nobody
ENTRYPOINT ["/app/bin/protoc-gen-nanopb"]
