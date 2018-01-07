# PgBouncer exporter

Prometheus exporter for PgBouncer.
Exports metrics at `9127/metrics`

## Building and running

    make
    ./pgbouncer_exporter <flags>

To see all available configuration flags:

    ./pgbouncer_exporter -h


## Metrics

Metric     | Description
---------|-------------
stats.avg_query_count | Average queries per second in last stat period
stats.avg_query | The average query duration, shown as microsecond
stats.avg_query_time | Average query duration in microseconds
stats.avg_recv | Average received (from clients) bytes per second
stats.avg_req | The average number of requests per second in last stat period, shown as request/second
stats.avg_sent | Average sent (to clients) bytes per second
stats.avg_wait_time | Time spent by clients waiting for a server in microseconds (average per second)
stats.avg_xact_count | Average transactions per second in last stat period
stats.avg_xact_time | Average transaction duration in microseconds
stats.bytes_received_per_second | The total network traffic received, shown as byte/second
stats.bytes_sent_per_second | The total network traffic sent, shown as byte/second
stats.total_query_count | Total number of SQL queries pooled
stats.total_query_time | Total number of microseconds spent by pgbouncer when actively connected to PostgreSQL, executing queries
stats.total_received | Total volume in bytes of network traffic received by pgbouncer, shown as bytes
stats.total_requests | Total number of SQL requests pooled by pgbouncer, shown as requests
stats.total_sent | Total volume in bytes of network traffic sent by pgbouncer, shown as bytes
stats.total_wait_time | Time spent by clients waiting for a server in microseconds
stats.total_xact_count | Total number of SQL transactions pooled
stats.total_xact_time | Total number of microseconds spent by pgbouncer when connected to PostgreSQL in a transaction, either idle in transaction or executing queries
pools.cl_active | Client connections linked to server connection and able to process queries, shown as connection
pools.cl_waiting | Client connections waiting on a server connection, shown as connection
pools.sv_active | Server connections linked to a client connection, shown as connection
pools.sv_idle | Server connections idle and ready for a client query, shown as connection
pools.sv_used | Server connections idle more than server_check_delay, needing server_check_query, shown as connection
pools.sv_tested | Server connections currently running either server_reset_query or server_check_query, shown as connection
pools.sv_login | Server connections currently in the process of logging in, shown as connection
pools.maxwait | Age of oldest unserved client connection, shown as second
