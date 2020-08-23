ARG ARCH="amd64"
ARG OS="linux"
FROM golang:alpine as builder

# ENV GO111MODULE=on

RUN apk update && apk add --no-cache git make bash curl

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . ./

RUN make update-go-deps
RUN make common-build

FROM quay.io/prometheus/busybox-${OS}-${ARCH}:latest
LABEL maintainer="The Prometheus Authors <prometheus-developers@googlegroups.com>"

ARG DB_HOST=odyssey
ARG DB_PORT=6432
ARG DB_NAME=db
ARG CONN_STR=postgres://postgres@$DB_HOST:$DB_PORT?dbname=$DB_NAME&sslmode=disable
ENV ENV_CONN_STR=${CONN_STR}

COPY --from=builder /app/pgbouncer_exporter ./bin/
COPY --from=builder /app/LICENSE .

RUN echo "[INFO] EXPORTER CONNECTION STRING $ENV_CONN_STR"

USER       nobody
ENTRYPOINT ./bin/pgbouncer_exporter --pgBouncer.connectionString="$ENV_CONN_STR"
EXPOSE     9127
