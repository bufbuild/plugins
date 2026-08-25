# syntax=docker/dockerfile:1.26
FROM python:3.13.15-trixie@sha256:785ba3c54682e15fd75186dd1732d88389884e9b6f7311871674c18e5d32a1f1 AS build
WORKDIR /app
RUN python -mvenv /app
ADD /requirements.txt requirements.txt
RUN . ./bin/activate \
    && pip install --no-cache-dir -r requirements.txt \
    && pip uninstall --yes pip \
    && rm -f requirements.txt bin/activate.fish bin/activate.csh bin/Activate.ps1 \
    && ln -sf /usr/bin/python /app/bin/python

FROM gcr.io/distroless/python3-debian13:latest@sha256:84e04b1c8a7904c81563ea125db995ba321fd330f3731f0a841e2f90757c4667 AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=build --chmod=0755 /app /app
USER nobody
ENTRYPOINT ["/app/bin/protoc-gen-nanopb"]
