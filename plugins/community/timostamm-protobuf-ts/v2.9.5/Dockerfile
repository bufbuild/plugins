# syntax=docker/dockerfile:1.12
FROM node:22.14.0-bookworm AS build
WORKDIR /app
COPY --link package*.json .
RUN npm ci \
 && ./node_modules/.bin/esbuild ./node_modules/.bin/protoc-gen-ts --bundle --external:typescript --platform=node --outfile=protoc-gen-ts.js

FROM gcr.io/distroless/nodejs22-debian12:latest@sha256:881157f8399d3ab71c54068f148c25296f7f9bee6d36279febad5a6f46f41c2b AS node

FROM gcr.io/distroless/cc-debian12:latest@sha256:b7550f0b15838de14c564337eef2b804ba593ae55d81ca855421bd52f19bb480 AS base

FROM scratch
COPY --link --from=base / /
COPY --link --from=node --chmod=0755 /nodejs/bin/node /nodejs/bin/node
COPY --link --from=build --chmod=0755 /app/protoc-gen-ts.js /app/protoc-gen-ts.js
COPY --link --from=build /app/node_modules/typescript /app/node_modules/typescript
USER nobody
ENTRYPOINT ["/nodejs/bin/node"]
CMD [ "/app/protoc-gen-ts.js" ]
