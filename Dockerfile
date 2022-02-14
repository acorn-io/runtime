FROM golang:1.17 AS src
COPY / /src
WORKDIR /src
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o /init /src/cmd/appimageinit

FROM scratch AS app-image-init
COPY --from=src /init /
ENTRYPOINT ["/init"]

