# syntax=docker/dockerfile:1

### build app
FROM golang:1.24.1 AS builder

ARG NEPTUNUS_VERSION

WORKDIR /build

COPY . . 

RUN go build -ldflags="-X 'main.Version=$NEPTUNUS_VERSION'" -o /neptunus ./cmd/neptunus

### create final image
FROM alpine:3

EXPOSE 9600/tcp

RUN addgroup --gid 9600 neptunus && \
    adduser --uid 9600 --ingroup neptunus --no-create-home --disabled-password neptunus

COPY --from=builder --chmod=555 neptunus /bin/neptunus

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

    [runtime]
      gcpercent = "75%"
      memlimit = "75%"

    [engine]
      storage = "fs"
      [engine.fs]
        directory = "/etc/neptunus/conf/pipelines"
        extention = "toml"
EOT

USER 9600:9600

ENTRYPOINT [ "/bin/neptunus" ]
CMD [ "run", "--config", "/etc/neptunus/conf/config.toml" ]
