# syntax=docker/dockerfile:1.24
FROM python:3.13.14-trixie@sha256:69719dbce933473eb3632ee1c3b25788e71752d615ddee02536475be4d9e310b AS build
WORKDIR /app
RUN python -mvenv /app
ADD /requirements.txt requirements.txt
RUN . ./bin/activate \
 && pip install --no-cache-dir -r requirements.txt \
 && pip uninstall --yes pip setuptools \
 && rm -f requirements.txt bin/activate.fish bin/activate.csh bin/Activate.ps1 \
 && ln -sf /usr/bin/python /app/bin/python

FROM gcr.io/distroless/python3-debian13:latest@sha256:325c8639176419afb9c3b50b420fbbd65d608f85f1e3f38bc4b8ba80aeac2e8d AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=build --chmod=0755 /app /app
USER nobody
CMD [ "/app/bin/protoc-gen-grpc-py" ]
