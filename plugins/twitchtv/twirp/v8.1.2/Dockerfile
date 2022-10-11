# syntax=docker/dockerfile:1.4
FROM golang:1.19.2-bullseye AS build
WORKDIR /app
COPY --link . /app
RUN CGO_ENABLED=0 go install -ldflags "-s -w" -trimpath github.com/twitchtv/twirp/protoc-gen-twirp

FROM scratch
COPY --from=build --link /etc/passwd /etc/passwd
COPY --from=build --link --chown=root:root /go/bin/protoc-gen-twirp .
USER nobody
ENTRYPOINT [ "/protoc-gen-twirp" ]
