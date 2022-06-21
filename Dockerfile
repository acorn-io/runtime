# syntax=docker/dockerfile:1.3-labs

FROM golang:1.18-alpine AS helper
WORKDIR /usr/src
RUN apk -U add curl
RUN curl -sfL https://github.com/loft-sh/devspace/archive/refs/tags/v5.18.5.tar.gz | tar xvzf - --strip-components=1
RUN --mount=type=cache,target=/go/pkg --mount=type=cache,target=/root/.cache/go-build CGO_ENABLED=0 go build -o /usr/local/bin/acorn-helper -ldflags "-s -w" ./helper

FROM golang:1.18 AS build
COPY / /src
WORKDIR /src
RUN --mount=type=cache,target=/go/pkg --mount=type=cache,target=/root/.cache/go-build make build

FROM alpine:3.16.0 AS base
RUN apk add --no-cache ca-certificates 
RUN adduser -D acorn
RUN mkdir apiserver.local.config && chown acorn apiserver.local.config
COPY --from=helper /usr/local/bin/acorn-helper /usr/local/bin/
COPY <<EOF /usr/local/bin/acorn-helper-init
#!/bin/sh
cp -f /usr/local/bin/acorn-helper /.acorn/acorn-helper
EOF
RUN chmod +x /usr/local/bin/acorn-helper-init
USER acorn
ENTRYPOINT ["/usr/local/bin/acorn"]

FROM base AS goreleaser
COPY acorn /usr/local/bin/acorn

FROM base
COPY --from=build /src/bin/acorn /usr/local/bin/acorn
