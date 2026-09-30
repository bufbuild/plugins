# syntax=docker/dockerfile:1.26
FROM python:3.13.15-trixie@sha256:82c46c08c991d3d3ff10476ac5e386c2c28f27bbd3985a02c5e39fe99ab272eb AS build
WORKDIR /app
RUN python -mvenv /app
ADD /requirements.txt requirements.txt
RUN . ./bin/activate \
 && pip install --no-cache-dir -r requirements.txt \
 && pip uninstall --yes pip setuptools \
 && rm -f requirements.txt bin/activate.fish bin/activate.csh bin/Activate.ps1 \
 && ln -sf /usr/bin/python /app/bin/python

FROM gcr.io/distroless/python3-debian13:latest@sha256:e866e2f9fbaa19f63e579cf2f61b7f76cd8a74c43a62a0af95d48678d9878436 AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=build --chmod=0755 /app /app
USER nobody
CMD [ "/app/bin/protoc-gen-py" ]
