# syntax=docker/dockerfile:1.27
FROM python:3.13.16-trixie@sha256:98daa3227dee3c38b8ee9e020e3785ec4274b1affd854c9f40ead46b6fb31f55 AS build
WORKDIR /app
RUN python -mvenv /app
ADD /requirements.txt requirements.txt
RUN . ./bin/activate \
 && pip install --no-cache-dir -r requirements.txt \
 && pip uninstall --yes pip setuptools \
 && rm -f requirements.txt bin/activate.fish bin/activate.csh bin/Activate.ps1 \
 && ln -sf /usr/bin/python /app/bin/python

FROM gcr.io/distroless/python3-debian13:latest@sha256:96234eaf03bba491856ead9142eccdda8cf9d09ce418770a260b84f529d81848 AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=build --chmod=0755 /app /app
USER nobody
CMD [ "/app/bin/protoc-gen-py" ]
