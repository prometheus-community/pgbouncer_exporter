## master / unreleased

## 0.5.1 / 2022-10-03

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
