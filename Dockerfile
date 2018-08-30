FROM golang:alpine AS build-env
RUN apk add --no-cache make git gcc musl-dev
ADD . /go/src/github.com/stanhu/pgbouncer_exporter
WORKDIR /go/src/github.com/stanhu/pgbouncer_exporter
RUN PREFIX=/go/bin/ make


FROM quay.io/prometheus/busybox
WORKDIR /app
COPY --from=build-env /go/bin/pgbouncer_exporter /app/
EXPOSE 9127
ENTRYPOINT [ "/app/pgbouncer_exporter" ]
