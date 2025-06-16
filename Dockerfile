ARG ARCH="amd64"
ARG OS="linux"
FROM quay.io/prometheus/busybox-${OS}-${ARCH}:latest
LABEL maintainer="The Prometheus Authors <prometheus-developers@googlegroups.com>"

RUN apk add --no-cache postgresql-client

ARG ARCH="amd64"
ARG OS="linux"
COPY .build/${OS}-${ARCH}/pgbouncer_exporter /bin/pgbouncer_exporter
COPY LICENSE                                /LICENSE

COPY entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

USER       nobody
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
EXPOSE     9127
