FROM golang:1.18 AS build
COPY / /src
WORKDIR /src
RUN --mount=type=cache,target=/go/pkg --mount=type=cache,target=/root/.cache/go-build make build

FROM alpine:3.15.4 AS base
RUN apk add --no-cache ca-certificates 
RUN adduser -D acorn
RUN mkdir apiserver.local.config && chown acorn apiserver.local.config
USER acorn
ENTRYPOINT ["/usr/local/bin/acorn"]

FROM base AS goreleaser
COPY acorn /usr/local/bin/acorn

FROM base
COPY --from=build /src/bin/acorn /usr/local/bin/acorn
