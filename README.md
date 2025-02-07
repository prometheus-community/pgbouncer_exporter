# PgBouncer exporter
[![Build Status](https://circleci.com/gh/prometheus-community/pgbouncer_exporter.svg?style=svg)](https://circleci.com/gh/prometheus-community/pgbouncer_exporter)

Prometheus exporter for PgBouncer.
Exports metrics at `9127/metrics`

## Building and running

    make build
    ./pgbouncer_exporter <flags>

## Exporter configuration

### Command line flags
To see all available configuration flags:

    ./pgbouncer_exporter -h

### Config file
The exporter can be configured using a config file using the `--config.file` flag.  
When using a config file the default operation changes from single-target to multi-target mode. <sup>Can be set by `legacy_mode`</sup> 

For more information about the possibilities and requirements see [the example config.yaml file within this repo](config.yaml)

    ./pgbouncer_exporter --config.file config.yaml

### Multi-Target mode

In multi-target mode this exporter adheres to the https://prometheus.io/docs/guides/multi-target-exporter/ pattern.
The probe endpoints accepts 2 parameters:

- `dsn`: the postgresql connection string
- `cred`: credential reference to credentials stored in the config file.

When `cred` is used the dsn will be updated with the credential values from the config file. 
If the DSN and credentials config define the same parameter the latter takes precedence.    
With the example below the exporter will scrape `"postgres://username:password@localhost:6543/pgbouncer?sslmode=disable"`.

*mtls with client certificates is also supported through the use of credentials config* 

`prometheus.yaml`:
```yaml
- job_name: pgbouncer-exporter
  metrics_path: /probe
  params:
    dsn: "postgres://localhost:6543/pgbouncer?sslmode=disable"
    cred: "monitoring"
```

`config.yaml`
```yaml
credentials:
  - key: monitoring 
    username: username
    password: password
```

## PGBouncer configuration

The pgbouncer\_exporter requires a configuration change to pgbouncer to ignore a PostgreSQL driver connection parameter. In the `pgbouncer.ini` please include this option:

    ignore_startup_parameters = extra_float_digits

## Run with docker

```
docker run prometheuscommunity/pgbouncer-exporter <flags>
```

## Metrics

|PgBouncer column|Prometheus Metric|Description|
|----------------|-----------------|-----------|
stats_total_query_count | pgbouncer_stats_queries_pooled_total | Total number of SQL queries pooled
stats.total_query_time | pgbouncer_stats_queries_duration_seconds_total | Total number of seconds spent by pgbouncer when actively connected to PostgreSQL, executing queries
stats.total_received | pgbouncer_stats_received_bytes_total | Total volume in bytes of network traffic received by pgbouncer, shown as bytes
stats.total_requests | pgbouncer_stats_queries_total | Total number of SQL requests pooled by pgbouncer, shown as requests
stats.total_sent | pgbouncer_stats_sent_bytes_total | Total volume in bytes of network traffic sent by pgbouncer, shown as bytes
stats.total_wait_time | pgbouncer_stats_client_wait_seconds_total | Time spent by clients waiting for a server in seconds
stats.total_xact_count | pgbouncer_stats_sql_transactions_pooled_total | Total number of SQL transactions pooled
stats.total_xact_time | pgbouncer_stats_server_in_transaction_seconds_total | Total number of seconds spent by pgbouncer when connected to PostgreSQL in a transaction, either idle in transaction or executing queries
pools.cl_active | pgbouncer_pools_client_active_connections | Client connections linked to server connection and able to process queries, shown as connection
pools.cl_waiting | pgbouncer_pools_client_waiting_connections | Client connections waiting on a server connection, shown as connection
pools.sv_active | pgbouncer_pools_server_active_connections | Server connections linked to a client connection, shown as connection
pools.sv_idle | pgbouncer_pools_server_idle_connections | Server connections idle and ready for a client query, shown as connection
pools.sv_used | pgbouncer_pools_server_used_connections | Server connections idle more than server_check_delay, needing server_check_query, shown as connection
pools.sv_tested | pgbouncer_pools_server_testing_connections | Server connections currently running either server_reset_query or server_check_query, shown as connection
pools.sv_login | pgbouncer_pools_server_login_connections | Server connections currently in the process of logging in, shown as connection
pools.maxwait | pgbouncer_pools_client_maxwait_seconds | Age of oldest unserved client connection, shown as second
config.max_client_conn | pgbouncer_config_max_client_connections | Configured maximum number of client connections
config.max_user_connections | pgbouncer_config_max_user_connections | Configured maximum number of server connections per user

## TLS and basic authentication

The pgbouncer exporter supports TLS and basic authentication.

To use TLS and/or basic authentication, you need to pass a configuration file
using the `--web.config.file` parameter. The format of the file is described
[in the exporter-toolkit repository](https://github.com/prometheus/exporter-toolkit/blob/master/docs/web-configuration.md).
