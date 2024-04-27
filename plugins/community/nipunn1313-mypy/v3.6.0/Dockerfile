# syntax=docker/dockerfile:1.4
FROM python:3.12.3-alpine3.19 AS build
WORKDIR /app
RUN python -mvenv /app
ADD /requirements.txt requirements.txt
RUN source ./bin/activate \
 && pip install --no-cache-dir -r requirements.txt \
 && pip uninstall --yes pip setuptools \
 && rm -f requirements.txt bin/activate.fish bin/activate.csh bin/Activate.ps1 \
 && rm -f bin/protoc-gen-mypy_grpc

FROM python:3.12.3-alpine3.19
COPY --from=build --link /app /app
USER nobody
ENTRYPOINT [ "/app/bin/protoc-gen-mypy" ]
