FROM  quay.io/prometheus/busybox:latest
LABEL maintainer="The Prometheus Authors <prometheus-developers@googlegroups.com>"

COPY pgbouncer_exporter /bin/pgbouncer_exporter

ENTRYPOINT ["/bin/pgbouncer_exporter"]
EXPOSE     9127
