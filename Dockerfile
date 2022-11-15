# syntax=docker/dockerfile:1.3-labs

FROM tonistiigi/binfmt:qemu-v6.2.0 AS binfmt
FROM moby/buildkit:v0.10.6 AS buildkit
FROM registry:2.8.1 AS registry
FROM rancher/klipper-lb:v0.3.5 AS klipper-lb

FROM golang:1.19-alpine AS helper
WORKDIR /usr/src
RUN apk -U add curl
RUN curl -sfL https://github.com/loft-sh/devspace/archive/refs/tags/v5.18.5.tar.gz | tar xvzf - --strip-components=1
RUN --mount=type=cache,target=/go/pkg --mount=type=cache,target=/root/.cache/go-build CGO_ENABLED=0 go build -o /usr/local/bin/acorn-helper -ldflags "-s -w" ./helper

FROM golang:1.19 AS build
COPY / /src
WORKDIR /src
RUN --mount=type=cache,target=/go/pkg --mount=type=cache,target=/root/.cache/go-build make build

FROM nginx:1.23.2-alpine AS base
RUN apk add --no-cache ca-certificates iptables ip6tables fuse3 git openssh pigz xz \
  && ln -s fusermount3 /usr/bin/fusermount
RUN adduser -D acorn
RUN mkdir apiserver.local.config && chown acorn apiserver.local.config
RUN --mount=from=binfmt,src=/usr/bin,target=/usr/src for i in aarch64 x86_64; do if [ -e /usr/src/qemu-$i ]; then cp /usr/src/qemu-$i /usr/bin; fi; done
RUN --mount=from=buildkit,src=/usr/bin,target=/usr/src for i in aarch64 x86_64; do if [ -e /usr/src/buildkit-qemu-$i ]; then cp /usr/src/buildkit-qemu-$i /usr/bin; fi; done
COPY --from=binfmt /usr/bin/binfmt /usr/local/bin
COPY --from=buildkit /usr/bin/buildkitd /usr/bin/buildctl /usr/bin/buildkit-runc /usr/local/bin
COPY --from=registry /etc/docker/registry/config.yml /etc/docker/registry/config.yml
COPY --from=registry /bin/registry /usr/local/bin
COPY --from=klipper-lb /usr/bin/entry /usr/local/bin/klipper-lb
COPY ./scripts/ds-containerd-config-path-entry /usr/local/bin
COPY ./scripts/setup-binfmt /usr/local/bin
COPY --from=helper /usr/local/bin/acorn-helper /usr/local/bin/
VOLUME /var/lib/buildkit

# NOTE: Requires buildkit: DOCKER_BUILDKIT=1
COPY <<EOF /usr/local/bin/acorn-helper-init
#!/bin/sh
cp -f /usr/local/bin/acorn-helper /.acorn/acorn-helper
EOF
RUN chmod +x /usr/local/bin/acorn-helper-init
CMD []
ENTRYPOINT ["/usr/local/bin/acorn"]

FROM base AS goreleaser
COPY acorn /usr/local/bin/acorn

FROM base
COPY --from=build /src/bin/acorn /usr/local/bin/acorn
