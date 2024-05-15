# syntax=docker/dockerfile:1.7
FROM python:3.11.9-alpine3.19 AS build
WORKDIR /app
RUN python -mvenv /app
ADD /requirements.txt requirements.txt
RUN source ./bin/activate \
 && pip install --no-cache-dir -r requirements.txt \
 && pip uninstall --yes pip \
 && rm -f requirements.txt bin/activate.fish bin/activate.csh bin/Activate.ps1 \
 && ln -sf /usr/bin/python /app/bin/python

FROM gcr.io/distroless/python3-debian12:latest@sha256:e8e50bc861b16d916f598d7ec920a8cef1e35e99b668a738fe80c032801ceb78
COPY --from=build --link /app /app
USER nobody
ENTRYPOINT [ "/app/bin/protoc-gen-nanopb" ]
