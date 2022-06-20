FROM node:18-buster as src
COPY / /usr/src
WORKDIR /usr/src
RUN yarn install
RUN yarn build

FROM nginx as static
COPY --from=src /usr/src/build /usr/share/nginx/html

FROM src as dynamic
CMD yarn start --host=0.0.0.0