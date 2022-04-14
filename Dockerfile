FROM golang:1.18 AS src
COPY / /src
WORKDIR /src
RUN --mount=type=cache,target=/go/pkg --mount=type=cache,target=/root/.cache/go-build CGO_ENABLED=0 go build -ldflags "-s -w" -o /init /src/cmd/appimageinit

FROM scratch AS app-image-init
COPY --from=src /init /
ENTRYPOINT ["/init"]