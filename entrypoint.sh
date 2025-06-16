#!/bin/bash
set -e

# Set defaults as fallback if vars are not set
: "${PGBOUNCER_HOST:=127.0.0.1}"
: "${PGBOUNCER_PORT:=6432}"

# Wait for pgbouncer to become available
until pg_isready -h "$PGBOUNCER_HOST" -p "$PGBOUNCER_PORT"; do
  echo "Waiting for PgBouncer on $PGBOUNCER_HOST:$PGBOUNCER_PORT..."
  sleep 2
done

# Launch the exporter
exec /bin/pgbouncer_exporter "$@"
