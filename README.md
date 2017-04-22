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
stats.requests_per_second | The request rate, shown as request/second
stats.bytes_received_per_second | The total network traffic received, shown as byte/second
stats.bytes_sent_per_second | The total network traffic sent, shown as byte/second
stats.total_query_time  | Time spent by pgbouncer actively querying PostgreSQL, shown as microsecond
stats.avg_req | The average number of requests per second in last stat period, shown as request/second
stats.avg_recv | The client network traffic received, shown as byte/second
stats.avg_sent | The client network traffic sent, shown as byte/second
stats.total_received | Total volume in bytes of network traffic received by pgbouncer, shown as bytes
stats.total_requests | Total number of SQL requests pooled by pgbouncer, shown as requests
stats.total_sent" | Total volume in bytes of network traffic sent by pgbouncer, shown as bytes
stats.avg_query | The average query duration, shown as microsecond
pools.cl_active | Client connections linked to server connection and able to process queries, shown as connection
pools.cl_waiting | Client connections waiting on a server connection, shown as connection
pools.sv_active | Server connections linked to a client connection, shown as connection
pools.sv_idle | Server connections idle and ready for a client query, shown as connection
pools.sv_used | Server connections idle more than server_check_delay, needing server_check_query, shown as connection
pools.sv_tested | Server connections currently running either server_reset_query or server_check_query, shown as connection
pools.sv_login | Server connections currently in the process of logging in, shown as connection
pools.maxwait | Age of oldest unserved client connection, shown as second
