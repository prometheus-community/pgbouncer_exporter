# PgBouncer exporter
[![Build Status](https://github.com/prometheus-community/pgbouncer_exporter/actions/workflows/ci.yml/badge.svg)](https://github.com/prometheus-community/pgbouncer_exporter/actions/workflows/ci.yml)

Prometheus exporter for PgBouncer.
Exports metrics at `9127/metrics`

## Requirements

- PgBouncer 1.8 or higher, since PgBouncer exporter 0.11.0

## Building and running

    make build
    ./pgbouncer_exporter <flags>

To see all available configuration flags:

    ./pgbouncer_exporter -h

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--pgBouncer.connectionString` | `postgres://postgres:@localhost:6543/pgbouncer?sslmode=disable` | Connection string for accessing PgBouncer. Can also be set with the `PGBOUNCER_EXPORTER_CONNECTION_STRING` environment variable. |
| `--pgBouncer.pid-file` | `""` | Path to the PgBouncer pid file. If set, the standard process metrics of the PgBouncer process are exported as `pgbouncer_process_...`. The exporter needs read access to the PgBouncer process files in `/proc`. |
| `--web.listen-address` | `:9127` | Addresses on which to expose metrics and web interface. Repeatable for multiple addresses. |
| `--web.telemetry-path` | `/metrics` | Path under which to expose metrics. |
| `--web.config.file` | `""` | Path to a configuration file that can enable TLS or authentication. See [TLS and basic authentication](#tls-and-basic-authentication). |
| `--web.systemd-socket` | `false` | Use systemd socket activation listeners instead of port listeners (Linux only). |
| `--log.level` | `info` | Only log messages with the given severity or above. One of: `debug`, `info`, `warn`, `error`. |
| `--log.format` | `logfmt` | Output format of log messages. One of: `logfmt`, `json`. |

## PGBouncer configuration

The pgbouncer\_exporter requires a configuration change to pgbouncer to ignore a PostgreSQL driver connection parameter. In the `pgbouncer.ini` please include this option:

    ignore_startup_parameters = extra_float_digits

## Run with docker

```
docker run prometheuscommunity/pgbouncer-exporter <flags>
```

## Metrics

Metrics from `SHOW DATABASES` are labelled with `name`, `host`, `port`, `database`, `force_user` and `pool_mode`.
Metrics from `SHOW STATS_TOTALS` are labelled with `database`. Metrics from `SHOW POOLS` are labelled with `database` and `user`.

| PgBouncer source | Prometheus metric | Description |
|------------------|-------------------|-------------|
| `SHOW DATABASES` `pool_size` | `pgbouncer_databases_pool_size` | Maximum number of server connections |
| `SHOW DATABASES` `reserve_pool / reserve_pool_size` | `pgbouncer_databases_reserve_pool` | Maximum number of additional connections for this database |
| `SHOW DATABASES` `max_connections` | `pgbouncer_databases_max_connections` | Maximum number of allowed connections for this database |
| `SHOW DATABASES` `current_connections` | `pgbouncer_databases_current_connections` | Current number of connections for this database |
| `SHOW DATABASES` `paused` | `pgbouncer_databases_paused` | 1 if this database is currently paused, else 0 |
| `SHOW DATABASES` `disabled` | `pgbouncer_databases_disabled` | 1 if this database is currently disabled, else 0 |
| `SHOW STATS_TOTALS` `query_count` | `pgbouncer_stats_totals_queries_pooled_total` | Total number of SQL queries pooled |
| `SHOW STATS_TOTALS` `query_time` | `pgbouncer_stats_totals_queries_duration_seconds_total` | Total number of seconds spent by pgbouncer when actively connected to PostgreSQL, executing queries |
| `SHOW STATS_TOTALS` `bytes_received` | `pgbouncer_stats_totals_received_bytes_total` | Total volume in bytes of network traffic received by pgbouncer, shown as bytes |
| `SHOW STATS_TOTALS` `bytes_sent` | `pgbouncer_stats_totals_sent_bytes_total` | Total volume in bytes of network traffic sent by pgbouncer, shown as bytes |
| `SHOW STATS_TOTALS` `wait_time` | `pgbouncer_stats_totals_client_wait_seconds_total` | Time spent by clients waiting for a server in seconds |
| `SHOW STATS_TOTALS` `xact_count` | `pgbouncer_stats_totals_sql_transactions_pooled_total` | Total number of SQL transactions pooled |
| `SHOW STATS_TOTALS` `xact_time` | `pgbouncer_stats_totals_server_in_transaction_seconds_total` | Total number of seconds spent by pgbouncer when connected to PostgreSQL in a transaction, either idle in transaction or executing queries |
| `SHOW STATS_TOTALS` `client_parse_count` | `pgbouncer_stats_totals_client_parses_total` | Total number of prepared statement Parse messages received from clients |
| `SHOW STATS_TOTALS` `server_parse_count` | `pgbouncer_stats_totals_server_parses_total` | Total number of prepared statement Parse messages sent by pgbouncer to PostgreSQL |
| `SHOW STATS_TOTALS` `bind_count` | `pgbouncer_stats_totals_binds_total` | Total number of prepared statements readied for execution with a Bind message |
| `SHOW STATS_TOTALS` `server_assignment_count` | `pgbouncer_stats_totals_server_assignments_total` | Total number of client connections which have been served since process start |
| `SHOW POOLS` `cl_active` | `pgbouncer_pools_client_active_connections` | Client connections linked to server connection and able to process queries, shown as connection |
| `SHOW POOLS` `cl_active_cancel_req` | `pgbouncer_pools_client_active_cancel_connections` | Client connections that have forwarded query cancellations to the server and are waiting for the server response |
| `SHOW POOLS` `cl_waiting` | `pgbouncer_pools_client_waiting_connections` | Client connections waiting on a server connection, shown as connection |
| `SHOW POOLS` `cl_waiting_cancel_req` | `pgbouncer_pools_client_waiting_cancel_connections` | Client connections that have not forwarded query cancellations to the server yet |
| `SHOW POOLS` `sv_active` | `pgbouncer_pools_server_active_connections` | Server connections linked to a client connection, shown as connection |
| `SHOW POOLS` `sv_active_cancel` | `pgbouncer_pools_server_active_cancel_connections` | Server connections that are currently forwarding a cancel request. |
| `SHOW POOLS` `sv_being_canceled` | `pgbouncer_pools_server_being_canceled_connections` | Servers that normally could become idle but are waiting to do so until all in-flight cancel requests have completed that were sent to cancel a query on this server. |
| `SHOW POOLS` `sv_idle` | `pgbouncer_pools_server_idle_connections` | Server connections idle and ready for a client query, shown as connection |
| `SHOW POOLS` `sv_used` | `pgbouncer_pools_server_used_connections` | Server connections idle more than server_check_delay, needing server_check_query, shown as connection |
| `SHOW POOLS` `sv_tested` | `pgbouncer_pools_server_testing_connections` | Server connections currently running either server_reset_query or server_check_query, shown as connection |
| `SHOW POOLS` `sv_login` | `pgbouncer_pools_server_login_connections` | Server connections currently in the process of logging in, shown as connection |
| `SHOW POOLS` `maxwait` | `pgbouncer_pools_client_maxwait_seconds` | Age of oldest unserved client connection, shown as second |
| `SHOW LISTS` `databases` | `pgbouncer_databases` | Count of databases |
| `SHOW LISTS` `users` | `pgbouncer_users` | Count of users |
| `SHOW LISTS` `pools` | `pgbouncer_pools` | Count of pools |
| `SHOW LISTS` `free_clients` | `pgbouncer_free_clients` | Count of free clients |
| `SHOW LISTS` `used_clients` | `pgbouncer_used_clients` | Count of used clients |
| `SHOW LISTS` `login_clients` | `pgbouncer_login_clients` | Count of clients in login state |
| `SHOW LISTS` `free_servers` | `pgbouncer_free_servers` | Count of free servers |
| `SHOW LISTS` `used_servers` | `pgbouncer_used_servers` | Count of used servers |
| `SHOW LISTS` `dns_names` | `pgbouncer_cached_dns_names` | Count of DNS names in the cache |
| `SHOW LISTS` `dns_zones` | `pgbouncer_cached_dns_zones` | Count of DNS zones in the cache |
| `SHOW LISTS` `dns_queries` | `pgbouncer_in_flight_dns_queries` | Count of in-flight DNS queries |
| `SHOW CONFIG` `max_client_conn` | `pgbouncer_config_max_client_connections` | Configured maximum number of client connections |
| `SHOW CONFIG` `max_user_connections` | `pgbouncer_config_max_user_connections` | Configured maximum number of server connections per user |
| `SHOW CLIENTS` | `pgbouncer_client_connections` | Number of client connections grouped by database, user, application name, and state |
| `SHOW VERSION` | `pgbouncer_version_info` | The PgBouncer version info |
| N/A | `pgbouncer_up` | The PgBouncer scrape succeeded |

## TLS and basic authentication

The pgbouncer exporter supports TLS and basic authentication.

To use TLS and/or basic authentication, you need to pass a configuration file
using the `--web.config.file` parameter. The format of the file is described
[in the exporter-toolkit repository](https://github.com/prometheus/exporter-toolkit/blob/master/docs/web-configuration.md).
