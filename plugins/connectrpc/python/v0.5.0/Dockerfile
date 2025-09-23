# syntax=docker/dockerfile:1.18
FROM python:3.11.13-bookworm AS build
WORKDIR /app
RUN python -mvenv /app
RUN . ./bin/activate \
 && pip install --no-cache-dir protoc-gen-connect-python==0.5.0 \
 && pip uninstall --yes pip setuptools \
 && rm -f bin/activate.fish bin/activate.csh bin/Activate.ps1

FROM python:3.11.13-bookworm
COPY --from=build --link /app /app
USER nobody
ENTRYPOINT [ "/app/bin/protoc-gen-connect-python" ]
