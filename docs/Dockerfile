FROM ghcr.io/acorn-io/images-mirror/node:19-buster as src
COPY / /usr/src
WORKDIR /usr/src
RUN yarn install

FROM src as bin
RUN yarn build

FROM ghcr.io/acorn-io/images-mirror/nginx:latest as static
COPY --from=bin /usr/src/build /usr/share/nginx/html

FROM src as dynamic
CMD yarn start --host=0.0.0.0
