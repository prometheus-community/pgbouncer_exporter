# PgBouncer exporter
[![Build Status](https://circleci.com/gh/prometheus-community/pgbouncer_exporter.svg?style=svg)](https://circleci.com/gh/prometheus-community/pgbouncer_exporter)

Prometheus exporter for PgBouncer.
Exports metrics at `9127/metrics`

## Building and running

    make build
    ./pgbouncer_exporter <flags>

To see all available configuration flags:

    ./pgbouncer_exporter -h


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
stats.database | pgbouncer_stats_database | Database name
stats.avg_xact_count | pgbouncer_stats_avg_xact_total | Average transactions per second in last stat period
stats.avg_query_count | pgbouncer_stats_avg_query_total | Average queries per second in last stat period
stats.avg_recv | pgbouncer_stats_avg_recv_bytes | Average received (from clients) bytes per second
stats.avg_sent | pgbouncer_stats_avg_sent_bytes | Average sent (to clients) bytes per second
stats.avg_xact_time | pgbouncer_stats_avg_xact_seconds | Average transaction duration in seconds
stats.avg_query_time | pgbouncer_stats_avg_query_seconds | Average query duration in seconds
stats.avg_wait_time | pgbouncer_stats_avg_wait_seconds | Time spent by clients waiting for a server in seconds (average per second)
pools.cl_active | pgbouncer_pools_client_active_connections | Client connections linked to server connection and able to process queries, shown as connection
pools.cl_waiting | pgbouncer_pools_client_waiting_connections | Client connections waiting on a server connection, shown as connection
pools.sv_active | pgbouncer_pools_server_active_connections | Server connections linked to a client connection, shown as connection
pools.sv_idle | pgbouncer_pools_server_idle_connections | Server connections idle and ready for a client query, shown as connection
pools.sv_used | pgbouncer_pools_server_used_connections | Server connections idle more than server_check_delay, needing server_check_query, shown as connection
pools.sv_tested | pgbouncer_pools_server_testing_connections | Server connections currently running either server_reset_query or server_check_query, shown as connection
pools.sv_login | pgbouncer_pools_server_login_connections | Server connections currently in the process of logging in, shown as connection
pools.maxwait | pgbouncer_pools_client_maxwait_seconds | Age of oldest unserved client connection, shown as second
pools.database | pgbouncer_pools_database | Database name pgbouncer connects to
pools.pool_mode | pgbouncer_pools_pool_mode | The pooling mode in use
databases.name | pgbouncer_databases_name | Name of configured database entry
databases.host | pgbouncer_databases_host | Host pgbouncer connects to
databases.port | pgbouncer_databases_port | Port pgbouncer connects to
databases.database | pgbouncer_databases_database | Actual database name pgbouncer connects to
databases.force_user | pgbouncer_databases_force_user | When user is part of the connection string, the connection between pgbouncer and PostgreSQL is forced to the given user, whatever the client user
databases.pool_size | pgbouncer_databases_pool_size_total | Maximum number of server connections
databases.reserve_pool | pgbouncer_databases_reserve_pool_total | Maximum number of additional connections for this database
databases.pool_mode | pgbouncer_databases_pool_mode | The database's override pool_mode
databases.max_connections | pgbouncer_databases_max_connections_total | Maximum number of allowed connections for this database
databases.current_connections | pgbouncer_databases_current_connections_total | Current number of connections for this database
databases.paused | pgbouncer_databases_paused | 1 if this database is currently paused, else 0
databases.disabled | pgbouncer_databases_disabled | 1 if this database is currently disabled, else 0
lists.databases | pgbouncer_lists_databases_total | Count of databases
lists.users | pgbouncer_lists_users_total | Count of users
lists.pools | pgbouncer_lists_pools_total | Count of pools
lists.free_clients | pgbouncer_lists_free_clients_total | Count of free clients
lists.used_clients | pgbouncer_lists_used_clients_total | Count of used clients
lists.login_clients | pgbouncer_lists_login_clients_total | Count of clients in login state
lists.free_servers | pgbouncer_lists_free_servers_total | Count of free servers
lists.used_servers | pgbouncer_lists_used_servers_total | Count of used servers
lists.dns_names | pgbouncer_lists_cached_dns_names_total | Count of DNS names in the cache
lists.dns_zones | pgbouncer_lists_cached_dns_zones_total | Count of DNS zones in the cache
lists.dns_queries | pgbouncer_lists_in_flight_dns_queries | Count of in-flight DNS queries
lists.dns_pending | pgbouncer_lists_pending_dns_queries | Count of pending DNS queries
mem.name | pgbouncer_mem_name | mem name
mem.size | pgbouncer_mem_size_total | mem size
mem.used | pgbouncer_mem_used_total | mem used
mem.free | pgbouncer_mem_free_total | mem free
mem.memtotal | pgbouncer_mem_memtotal_bytes_total | mem total
