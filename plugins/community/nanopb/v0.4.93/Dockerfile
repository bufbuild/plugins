# syntax=docker/dockerfile:1.27
FROM python:3.13.18-trixie@sha256:51db6babcbd039687e5362a676db5444b5926739abb03b10c8c6a8144872611f AS build
WORKDIR /app
RUN python -mvenv /app
ADD /requirements.txt requirements.txt
RUN . ./bin/activate \
    && pip install --no-cache-dir -r requirements.txt \
    && pip uninstall --yes pip \
    && rm -f requirements.txt bin/activate.fish bin/activate.csh bin/Activate.ps1 \
    && ln -sf /usr/bin/python /app/bin/python

FROM gcr.io/distroless/python3-debian13:latest@sha256:5bac3c3ab33426c460db63504cd1a267b3326b9f804d21c63359492858d1beda AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=build --chmod=0755 /app /app
USER nobody
ENTRYPOINT ["/app/bin/protoc-gen-nanopb"]
