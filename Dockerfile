ARG ARCH="amd64"
ARG OS="linux"
FROM quay.io/prometheus/busybox-${OS}-${ARCH}:latest
LABEL maintainer="The Prometheus Authors <prometheus-developers@googlegroups.com>"

ARG ARCH="amd64"
ARG OS="linux"
COPY .build/${OS}-${ARCH}/pgbouncer_exporter /bin/pgbouncer_exporter
COPY LICENSE                                /LICENSE

USER       nobody
ENTRYPOINT ["/bin/pgbouncer_exporter"]
EXPOSE     9127
