FROM node:18-buster as src
COPY / /usr/src
WORKDIR /usr/src
RUN yarn install

FROM src as bin
RUN yarn build

FROM nginx as static
COPY --from=bin /usr/src/build /usr/share/nginx/html

FROM src as dynamic
CMD yarn start --host=0.0.0.0
