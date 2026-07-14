# syntax=docker/dockerfile:1.24
FROM python:3.13.14-trixie@sha256:705ee9b6af4891eaae0098008a8a99bb9b1246e3fa8b687519b87f8e355c3e0a AS build
WORKDIR /app
RUN python -mvenv /app
ADD /requirements.txt requirements.txt
RUN . ./bin/activate \
 && pip install --no-cache-dir -r requirements.txt \
 && pip uninstall --yes pip setuptools \
 && rm -f requirements.txt bin/activate.fish bin/activate.csh bin/Activate.ps1 \
 && ln -sf /usr/bin/python /app/bin/python

FROM gcr.io/distroless/python3-debian13:latest@sha256:393cdf69ec7a5e217f837f2ff9b2123e06545d89c6e718c14ad020451fcb1900 AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=build --chmod=0755 /app /app
USER nobody
CMD [ "/app/bin/protoc-gen-py" ]
