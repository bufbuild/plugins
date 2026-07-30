# syntax=docker/dockerfile:1.24
FROM python:3.13.14-trixie@sha256:54e7879ce487f34f8ee85fc37e7547e843a2358bbcb19ecc273869e9d9fcc7dc AS build
WORKDIR /app
RUN python -mvenv /app
ADD /requirements.txt requirements.txt
RUN . ./bin/activate \
 && pip install --no-cache-dir -r requirements.txt \
 && pip uninstall --yes pip setuptools \
 && rm -f requirements.txt bin/activate.fish bin/activate.csh bin/Activate.ps1 \
 && ln -sf /usr/bin/python /app/bin/python

FROM gcr.io/distroless/python3-debian13:latest@sha256:3f056d8b0189540a15b5a3a125cfd28fb0960af0af2c56cf8bcec613fd771a00 AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=build --chmod=0755 /app /app
USER nobody
CMD [ "/app/bin/protoc-gen-grpc-py" ]
