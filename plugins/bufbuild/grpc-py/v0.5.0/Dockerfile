# syntax=docker/dockerfile:1.27
FROM python:3.13.17-trixie@sha256:d5a186865324088bd10d1671801b42297a8fe3f8059143fb22b2e2b776ffa6dd AS build
WORKDIR /app
RUN python -mvenv /app
ADD /requirements.txt requirements.txt
RUN . ./bin/activate \
 && pip install --no-cache-dir -r requirements.txt \
 && pip uninstall --yes pip setuptools \
 && rm -f requirements.txt bin/activate.fish bin/activate.csh bin/Activate.ps1 \
 && ln -sf /usr/bin/python /app/bin/python

FROM gcr.io/distroless/python3-debian13:latest@sha256:0db804f397ca31a1498c9d94c3c1df53fe49ac8007e5942b1cb547c1c3355c09 AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=build --chmod=0755 /app /app
USER nobody
CMD [ "/app/bin/protoc-gen-grpc-py" ]
