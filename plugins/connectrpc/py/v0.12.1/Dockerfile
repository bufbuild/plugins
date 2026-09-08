# syntax=docker/dockerfile:1.26
FROM python:3.13.15-trixie@sha256:4a0820d8b1b1a1e012a23ff9a1296a3e28b48896a4d27a328a91ee197ce619d9 AS build
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
