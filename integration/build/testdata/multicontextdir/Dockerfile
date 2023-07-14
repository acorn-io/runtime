FROM ghcr.io/acorn-io/images-mirror/busybox:latest AS not-default
COPY test.sh /test.sh
COPY --from=other-context /token /
RUN test -f /test.sh && sh /test.sh
