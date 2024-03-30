
# syntax=docker/dockerfile:1

### build app
FROM golang:1.21.0 as builder

ARG NEPTUNUS_VERSION

WORKDIR /build

COPY . . 

RUN go build -ldflags="-X 'main.Version=$NEPTUNUS_VERSION'" -o /neptunus ./cmd/neptunus

### create final image
FROM alpine:3

EXPOSE 9600/tcp

COPY --from=builder --chmod=u+x neptunus /bin/neptunus

# musl and glibc compability
RUN mkdir /lib64 && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2

RUN mkdir -p /etc/neptunus/conf/pipelines

COPY <<-EOT /etc/neptunus/conf/config.toml
    [common]
      log_level = "info"
      log_format = "logfmt"
      http_port = ":9600"
      [common.log_fields]
        runner = "local"

    [engine]
      storage = "fs"
      [engine.fs]
        directory = "/etc/neptunus/conf/pipelines"
        extention = "toml"
EOT

ENTRYPOINT [ "/bin/neptunus" ]
CMD [ "run", "--config", "/etc/neptunus/conf/config.toml" ]
