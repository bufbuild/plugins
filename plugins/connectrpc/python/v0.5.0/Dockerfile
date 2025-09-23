# syntax=docker/dockerfile:1.18
FROM python:3.11.13-bookworm AS build
WORKDIR /app
RUN python -mvenv /app
ADD /requirements.txt requirements.txt
RUN . ./bin/activate \
 && pip install --no-cache-dir -r requirements.txt \
 && pip uninstall --yes pip setuptools \
 && rm -f requirements.txt bin/activate.fish bin/activate.csh bin/Activate.ps1

FROM python:3.11.13-bookworm
COPY --from=build --link /app /app
USER nobody
ENTRYPOINT [ "/app/bin/protoc-gen-connect_python" ]
