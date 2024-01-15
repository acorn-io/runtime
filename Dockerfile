# syntax=docker/dockerfile:1.3-labs

FROM ghcr.io/acorn-io/images-mirror/tonistiigi/binfmt:qemu-v8.1.4 AS binfmt
FROM ghcr.io/acorn-io/images-mirror/coredns/coredns:1.10.1 AS coredns
FROM ghcr.io/acorn-io/images-mirror/moby/buildkit:v0.12.4 AS buildkit
FROM ghcr.io/acorn-io/images-mirror/registry:2.8.3 AS registry
FROM ghcr.io/acorn-io/images-mirror/traefik:2.10.7 AS traefik
FROM ghcr.io/acorn-io/images-mirror/rancher/k3s:v1.29.0-k3s1 AS k3s
FROM ghcr.io/acorn-io/images-mirror/rancher/klipper-lb:v0.4.5 AS klipper-lb
FROM ghcr.io/acorn-io/sleep:latest AS sleep

FROM ghcr.io/acorn-io/images-mirror/golang:1.21-alpine AS helper
WORKDIR /usr/src
RUN apk -U add curl
RUN curl -sfL https://github.com/loft-sh/devspace/archive/refs/tags/v6.3.2.tar.gz | tar xzf - --strip-components=1
RUN --mount=type=cache,target=/go/pkg --mount=type=cache,target=/root/.cache/go-build CGO_ENABLED=0 go build -o /usr/local/bin/acorn-helper -ldflags "-s -w" ./helper

FROM cgr.dev/chainguard/wolfi-base AS pause
RUN apk add -U crane
RUN crane pull --platform=linux/amd64 --platform=linux/arm64 --format=oci rancher/mirrored-pause:3.6 /out
RUN tar cvf /pause.tar -C /out .

FROM ghcr.io/acorn-io/images-mirror/golang:1.21-alpine AS loglevel
WORKDIR /usr/src
RUN apk -U add curl && rm -rf /var/cache/apk/*
RUN curl -sfL https://github.com/acorn-io/loglevel/archive/refs/tags/v0.1.6.tar.gz | tar xzf - --strip-components=1
RUN --mount=type=cache,target=/go/pkg --mount=type=cache,target=/root/.cache/go-build CGO_ENABLED=0 go build -o /usr/local/bin/loglevel -ldflags "-s -w"

FROM ghcr.io/acorn-io/images-mirror/golang:1.21 AS build
COPY / /src
WORKDIR /src
COPY --from=sleep /sleep /src/pkg/controller/appdefinition/embed/acorn-sleep
RUN --mount=type=cache,target=/go/pkg --mount=type=cache,target=/root/.cache/go-build GO_TAGS=netgo,image make build

FROM ghcr.io/acorn-io/images-mirror/nginx:1.23.2-alpine AS base
RUN apk add --no-cache ca-certificates iptables ip6tables fuse3 git openssh pigz xz busybox-static \
  && ln -s fusermount3 /usr/bin/fusermount
RUN adduser -D acorn
RUN mkdir /wd && \
    chown acorn /wd && \
    mkdir /etc/coredns
RUN --mount=from=binfmt,src=/usr/bin,target=/usr/src for i in aarch64 x86_64; do if [ -e /usr/src/qemu-$i ]; then cp /usr/src/qemu-$i /usr/bin; fi; done
RUN --mount=from=buildkit,src=/usr/bin,target=/usr/src for i in aarch64 x86_64; do if [ -e /usr/src/buildkit-qemu-$i ]; then cp /usr/src/buildkit-qemu-$i /usr/bin; fi; done
COPY --from=binfmt /usr/bin/binfmt /usr/local/bin
COPY --from=buildkit /usr/bin/buildkitd /usr/bin/buildctl /usr/bin/buildkit-runc /usr/local/bin/
COPY --from=registry /etc/docker/registry/config.yml /etc/docker/registry/config.yml
COPY --from=registry /bin/registry /usr/local/bin
COPY --from=klipper-lb /usr/bin/entry /usr/local/bin/klipper-lb
COPY --from=coredns /coredns /usr/local/bin/coredns
COPY --from=traefik /usr/local/bin/traefik /usr/local/bin/traefik
COPY --from=pause /pause.tar /var/lib/rancher/k3s/agent/images/
RUN --mount=from=k3s,target=/k3s tar cf - -C /k3s bin | tar xvf -
COPY ./scripts/ds-containerd-config-path-entry /usr/local/bin
COPY ./scripts/setup-binfmt /usr/local/bin
COPY ./scripts/40-copy-resolv-nameserver.sh /docker-entrypoint.d/
COPY --from=helper /usr/local/bin/acorn-helper /usr/local/bin/
COPY --from=loglevel /usr/local/bin/loglevel /usr/local/bin/

COPY /scripts/acorn-helper-init /usr/local/bin
COPY /scripts/acorn-busybox-init /usr/local/bin
COPY /scripts/acorn-job-helper-init /usr/local/bin
COPY /scripts/acorn-job-helper-shutdown /usr/local/bin
COPY /scripts/acorn-job-get-output /usr/local/bin
COPY /scripts/k3s-config.yaml /etc/rancher/k3s/config.yaml
CMD []
WORKDIR /wd
VOLUME /var/lib/buildkit
VOLUME /var/lib/rancher/k3s
STOPSIGNAL SIGTERM
ENTRYPOINT ["/usr/local/bin/acorn"]

FROM base AS goreleaser
COPY acorn /usr/local/bin/acorn

FROM base
COPY --from=build /src/bin/acorn /usr/local/bin/acorn
