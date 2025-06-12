## master / unreleased

## 0.11.0 / 2025-06-13

Notes:
- This version changes the connection behavior to create a new client
connection for each scrape. Previously a connection was created at startup.
- This version now requires PgBouncer >= 1.8.

* [CHANGE] Make connections per-scrape #214
* [FEATURE] Add prepared statement metrics #194
* [FEATURE] Add total_server_assignment_count metric #208

## 0.10.2 / 2024-10-18

* [BUGFIX] Fix wrong logging level of "Starting scrape" message #175

## 0.10.1 / 2024-10-14

* [BUGFIX] Revert auth_type guage #173

## 0.10.0 / 2024-10-07

* [CHANGE] Switch logging to slog #167
* [ENHANCEMENT] Add auth_type to config collector #169

## 0.9.0 / 2024-08-01

* [FEATURE] Allow connection config via environment variable #159

## 0.8.0 / 2024-04-02

* [ENHANCEMENT] Publish server/client cancel statistics. #1144

## 0.7.0 / 2023-06-29

* [CHANGE] Require Go 1.19 and update CI with Go 1.20 #120
* [CHANGE] Synchronize common files from prometheus/prometheus #123

## 0.6.0 / 2023-01-27

* [FEATURE] Add config metrics #93
* [FEATURE] Add TLS and Basic auth to the metrics endpoint #101

## 0.5.1 / 2022-10-03

* No changes, just retagging due to a VERSION fix.

## 0.5.0 / 2022-10-03

* [CHANGE] Update Go to 1.18.
* [CHANGE] Update upstream dependencies.

## 0.4.1 / 2022-01-27

* [BUGFIX] Fix startup log message typo #50
* [BUGFIX] Fix typo in reserve_pool metric #67

## 0.4.0 / 2020-07-09

Counter names have been updated to match Prometheus naming conventions.
* `pgbouncer_stats_queries_duration_seconds` -> `pgbouncer_stats_queries_duration_seconds_total`
* `pgbouncer_stats_client_wait_seconds` -> `pgbouncer_stats_client_wait_seconds_total`
* `pgbouncer_stats_server_in_transaction_seconds` -> `pgbouncer_stats_server_in_transaction_seconds_total`

* [CHANGE] Cleanup exporter metrics #33
* [CHANGE] Update counter metric names #35
* [FEATURE] Add support for SHOW LISTS metrics #36

## 0.3.0 / 2020-05-27

* [CHANGE] Switch logging to promlog #29
* [FEATURE] Add pgbouncer process metrics #27

## 0.2.0 / 2020-04-29

* [BUGFIX] Fix byte slice values not receiving conversion factor #18

Initial prometheus-community release.
