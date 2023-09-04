# syntax=docker/dockerfile:1.4
FROM python:3.11.4-alpine3.18 AS build
WORKDIR /app
RUN python -mvenv /app
ADD /requirements.txt requirements.txt
RUN source ./bin/activate \
 && pip install --no-cache-dir -r requirements.txt \
 && pip uninstall --yes pip setuptools \
 && rm -f requirements.txt bin/activate.fish bin/activate.csh bin/Activate.ps1

FROM python:3.11.4-alpine3.18
COPY --from=build --link /app /app
USER nobody
# Plugin uses os.makedirs - needs to run in a directory it can write to
WORKDIR /tmp
ENTRYPOINT [ "/app/bin/protoc-gen-python_betterproto" ]
