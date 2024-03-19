# syntax=docker/dockerfile:1.7
FROM node:20.11.1-alpine3.19 AS build
WORKDIR /app
COPY --link package*.json .
RUN npm ci \
 && find node_modules/typescript ! -name 'typescript.js' ! -name 'package.json' -type f -exec rm -f {} + \
 && find node_modules/typescript -depth -type d -empty -delete \
 && ./node_modules/.bin/esbuild ./node_modules/.bin/protoc-gen-connect-es --bundle --external:typescript --platform=node --outfile=protoc-gen-connect-es.js

FROM gcr.io/distroless/nodejs20-debian12:latest@sha256:e0edb89411e3f93863e9bd97c15cddcbac968d7a16b7a82af6ae6b264da65de9 AS node

FROM gcr.io/distroless/cc-debian12:latest@sha256:e6ae66a5a343d7112167f9117c4e630cfffcd80db44e44302759ec13ddd2d22b
COPY --link --from=node --chmod=0755 /nodejs/bin/node /nodejs/bin/node
COPY --link --from=build --chmod=0755 /app/protoc-gen-connect-es.js /app/protoc-gen-connect-es.js
COPY --link --from=build /app/node_modules/typescript /app/node_modules/typescript
USER nobody
ENTRYPOINT ["/nodejs/bin/node"]
CMD [ "/app/protoc-gen-connect-es.js" ]
