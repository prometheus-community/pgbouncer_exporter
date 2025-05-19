// Copyright 2020 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"os"
	"strconv"
	"time"
	"unicode/utf8"

	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	metricMaps = map[string]map[string]ColumnMapping{
		"databases": {
			"name":                {LABEL, "N/A", 1, "N/A"},
			"host":                {LABEL, "N/A", 1, "N/A"},
			"port":                {LABEL, "N/A", 1, "N/A"},
			"database":            {LABEL, "N/A", 1, "N/A"},
			"force_user":          {LABEL, "N/A", 1, "N/A"},
			"pool_size":           {GAUGE, "pool_size", 1, "Maximum number of server connections"},
			"reserve_pool":        {GAUGE, "reserve_pool", 1, "Maximum number of additional connections for this database"},
			"pool_mode":           {LABEL, "N/A", 1, "N/A"},
			"max_connections":     {GAUGE, "max_connections", 1, "Maximum number of allowed connections for this database"},
			"current_connections": {GAUGE, "current_connections", 1, "Current number of connections for this database"},
			"paused":              {GAUGE, "paused", 1, "1 if this database is currently paused, else 0"},
			"disabled":            {GAUGE, "disabled", 1, "1 if this database is currently disabled, else 0"},
		},
		"stats_totals": {
			"database":           {LABEL, "N/A", 1, "N/A"},
			"query_count":        {COUNTER, "queries_pooled_total", 1, "Total number of SQL queries pooled"},
			"query_time":         {COUNTER, "queries_duration_seconds_total", 1e-6, "Total number of seconds spent by pgbouncer when actively connected to PostgreSQL, executing queries"},
			"bytes_received":     {COUNTER, "received_bytes_total", 1, "Total volume in bytes of network traffic received by pgbouncer, shown as bytes"},
			"requests":           {COUNTER, "queries_total", 1, "Total number of SQL requests pooled by pgbouncer, shown as requests"},
			"bytes_sent":         {COUNTER, "sent_bytes_total", 1, "Total volume in bytes of network traffic sent by pgbouncer, shown as bytes"},
			"wait_time":          {COUNTER, "client_wait_seconds_total", 1e-6, "Time spent by clients waiting for a server in seconds"},
			"xact_count":         {COUNTER, "sql_transactions_pooled_total", 1, "Total number of SQL transactions pooled"},
			"xact_time":          {COUNTER, "server_in_transaction_seconds_total", 1e-6, "Total number of seconds spent by pgbouncer when connected to PostgreSQL in a transaction, either idle in transaction or executing queries"},
			"client_parse_count": {COUNTER, "client_parses_total", 1, "Total number of prepared statement Parse messages received from clients"},
			"server_parse_count": {COUNTER, "server_parses_total", 1, "Total number of prepared statement Parse messages sent by pgbouncer to PostgreSQL"},
			"bind_count":         {COUNTER, "binds_total", 1, "Total number of prepared statements readied for execution with a Bind message"},
		},
		"pools": {
			"database":              {LABEL, "N/A", 1, "N/A"},
			"user":                  {LABEL, "N/A", 1, "N/A"},
			"cl_active":             {GAUGE, "client_active_connections", 1, "Client connections linked to server connection and able to process queries, shown as connection"},
			"cl_active_cancel_req":  {GAUGE, "client_active_cancel_connections", 1, "Client connections that have forwarded query cancellations to the server and are waiting for the server response"},
			"cl_waiting":            {GAUGE, "client_waiting_connections", 1, "Client connections waiting on a server connection, shown as connection"},
			"cl_waiting_cancel_req": {GAUGE, "client_waiting_cancel_connections", 1, "Client connections that have not forwarded query cancellations to the server yet"},
			"sv_active":             {GAUGE, "server_active_connections", 1, "Server connections linked to a client connection, shown as connection"},
			"sv_active_cancel":      {GAUGE, "server_active_cancel_connections", 1, "Server connections that are currently forwarding a cancel request."},
			"sv_being_canceled":     {GAUGE, "server_being_canceled_connections", 1, "Servers that normally could become idle but are waiting to do so until all in-flight cancel requests have completed that were sent to cancel a query on this server."},
			"sv_idle":               {GAUGE, "server_idle_connections", 1, "Server connections idle and ready for a client query, shown as connection"},
			"sv_used":               {GAUGE, "server_used_connections", 1, "Server connections idle more than server_check_delay, needing server_check_query, shown as connection"},
			"sv_tested":             {GAUGE, "server_testing_connections", 1, "Server connections currently running either server_reset_query or server_check_query, shown as connection"},
			"sv_login":              {GAUGE, "server_login_connections", 1, "Server connections currently in the process of logging in, shown as connection"},
			"maxwait":               {GAUGE, "client_maxwait_seconds", 1, "Age of oldest unserved client connection, shown as second"},
		},
	}

	listsMap = map[string]*(prometheus.Desc){
		"databases": prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "databases"),
			"Count of databases", nil, nil),
		"users": prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "users"),
			"Count of users", nil, nil),
		"pools": prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "pools"),
			"Count of pools", nil, nil),
		"free_clients": prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "free_clients"),
			"Count of free clients", nil, nil),
		"used_clients": prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "used_clients"),
			"Count of used clients", nil, nil),
		"login_clients": prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "login_clients"),
			"Count of clients in login state", nil, nil),
		"free_servers": prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "free_servers"),
			"Count of free servers", nil, nil),
		"used_servers": prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "used_servers"),
			"Count of used servers", nil, nil),
		"dns_names": prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "cached_dns_names"),
			"Count of DNS names in the cache", nil, nil),
		"dns_zones": prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "cached_dns_zones"),
			"Count of DNS zones in the cache", nil, nil),
		"dns_queries": prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "in_flight_dns_queries"),
			"Count of in-flight DNS queries", nil, nil),
	}

	configMap = map[string]*(prometheus.Desc){
		"max_client_conn": prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "config", "max_client_connections"),
			"Config maximum number of client connections", nil, nil),
		"max_user_connections": prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "config", "max_user_connections"),
			"Config maximum number of server connections per user", nil, nil),
	}
)

// Metric descriptors.
var (
	bouncerVersionDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "version", "info"),
		"The pgbouncer version info",
		[]string{"version"}, nil,
	)
	scrapeSuccessDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "up"),
		"The pgbouncer scrape succeeded",
		nil, nil,
	)
)

func NewExporter(connectionString string, namespace string, logger *slog.Logger, filterEmptyPools bool) *Exporter {

	db, err := getDB(connectionString)

	if err != nil {
		logger.Error("error setting up DB connection", "err", err.Error())
		os.Exit(1)
	}

	return &Exporter{
		metricMap:        makeDescMap(metricMaps, namespace, logger),
		db:               db,
		logger:           logger,
		filterEmptyPools: filterEmptyPools,
	}
}

// Query SHOW LISTS, which has a series of rows, not columns.
func queryShowLists(ch chan<- prometheus.Metric, db *sql.DB, logger *slog.Logger) error {
	rows, err := db.Query("SHOW LISTS;")
	if err != nil {
		return fmt.Errorf("error running SHOW LISTS on database: %w", err)
	}
	defer rows.Close()

	columnNames, err := rows.Columns()
	if err != nil || len(columnNames) != 2 {
		return fmt.Errorf("error retrieving columns list from SHOW LISTS: %w", err)
	}

	var list string
	var items sql.RawBytes
	for rows.Next() {
		if err = rows.Scan(&list, &items); err != nil {
			return fmt.Errorf("error retrieving SHOW LISTS rows: %w", err)
		}
		value, err := strconv.ParseFloat(string(items), 64)
		if err != nil {
			return fmt.Errorf("error parsing SHOW LISTS column: %v, error: %w", list, err)
		}
		if metric, ok := listsMap[list]; ok {
			ch <- prometheus.MustNewConstMetric(metric, prometheus.GaugeValue, value)
		} else {
			logger.Debug("SHOW LISTS unknown list", "list", list)
		}
	}
	return nil
}

// Query SHOW CONFIG, which has a series of rows, not columns.
func queryShowConfig(ch chan<- prometheus.Metric, db *sql.DB, logger *slog.Logger) error {
	rows, err := db.Query("SHOW CONFIG;")
	if err != nil {
		return fmt.Errorf("error running SHOW CONFIG on database: %w", err)
	}
	defer rows.Close()

	columnNames, err := rows.Columns()
	numColumns := len(columnNames)
	if err != nil {
		return fmt.Errorf("error retrieving columns list from SHOW CONFIG: %w", err)
	}

	exposedConfig := make(map[string]bool)
	for configKey := range configMap {
		exposedConfig[configKey] = true
	}

	var key string
	var values sql.RawBytes
	var defaultValue sql.RawBytes
	var changeable string
	for rows.Next() {
		switch numColumns {
		case 3:
			if err = rows.Scan(&key, &values, &changeable); err != nil {
				return fmt.Errorf("error retrieving SHOW CONFIG rows: %w", err)
			}
		case 4:
			if err = rows.Scan(&key, &values, &defaultValue, &changeable); err != nil {
				return fmt.Errorf("error retrieving SHOW CONFIG rows: %w", err)
			}
		default:
			return fmt.Errorf("invalid number of SHOW CONFIG  columns: %d", numColumns)
		}

		if !exposedConfig[key] {
			continue
		}

		value, err := strconv.ParseFloat(string(values), 64)
		if err != nil {
			return fmt.Errorf("error parsing SHOW CONFIG column: %v, error: %w ", key, err)
		}
		if metric, ok := configMap[key]; ok {
			ch <- prometheus.MustNewConstMetric(metric, prometheus.GaugeValue, value)
		} else {
			logger.Debug("SHOW CONFIG unknown config", "config", key)
		}
	}
	return nil
}

func hasActiveConnections(columnData []interface{}, columnIdx map[string]int) (bool, error) {
	// Connection metrics to check
	metricsToCheck := []string{
		"cl_active",
		"cl_waiting",
		"sv_active",
		"sv_idle",
		"sv_used",
		"sv_tested",
		"sv_login",
	}

	sum := float64(0)

	for _, metric := range metricsToCheck {
		if idx, ok := columnIdx[metric]; ok {
			if value, ok := dbToFloat64(columnData[idx], 1.0); ok {
				sum += value
			}
		}
	}

	return sum > 0, nil
}

// Query within a namespace mapping and emit metrics. Returns fatal errors if
// the scrape fails, and a slice of errors if they were non-fatal.
func queryNamespaceMapping(ch chan<- prometheus.Metric, db *sql.DB, namespace string, mapping MetricMapNamespace, filterEmptyPools bool, logger *slog.Logger) ([]error, error) {
	query := fmt.Sprintf("SHOW %s;", namespace)

	// Don't fail on a bad scrape of one metric
	rows, err := db.Query(query)
	if err != nil {
		return []error{}, fmt.Errorf("error running query on database: %v, error: %w", namespace, err)
	}

	defer rows.Close()

	var columnNames []string
	columnNames, err = rows.Columns()
	if err != nil {
		return []error{}, fmt.Errorf("error retrieving column list for: %v, error: %w", namespace, err)
	}

	// Make a lookup map for the column indices
	var columnIdx = make(map[string]int, len(columnNames))
	for i, n := range columnNames {
		columnIdx[n] = i
	}

	var columnData = make([]interface{}, len(columnNames))
	var scanArgs = make([]interface{}, len(columnNames))
	for i := range columnData {
		scanArgs[i] = &columnData[i]
	}

	nonfatalErrors := []error{}

	for rows.Next() {
		labelValues := make([]string, len(mapping.labels))
		err = rows.Scan(scanArgs...)
		if err != nil {
			return []error{}, fmt.Errorf("error retrieving rows: %v, error: %w", namespace, err)
		}

		if namespace == "pools" && filterEmptyPools {
			hasConnections, err := hasActiveConnections(columnData, columnIdx)
			if err != nil {
				nonfatalErrors = append(nonfatalErrors, fmt.Errorf("error checking active connections: %w", err))
				continue
			}
			if !hasConnections {
				continue // Skip this pool
			}
		}

		for i, label := range mapping.labels {
			for idx, columnName := range columnNames {
				if columnName == label {
					switch v := columnData[idx].(type) {
					case int:
						labelValues[i] = strconv.Itoa(columnData[idx].(int))
					case int64:
						labelValues[i] = strconv.Itoa(int(columnData[idx].(int64)))
					case float64:
						labelValues[i] = fmt.Sprintf("%f", columnData[idx].(float64))
					case string:
						labelValues[i] = columnData[idx].(string)
					case nil:
						labelValues[i] = ""
					default:
						nonfatalErrors = append(nonfatalErrors, fmt.Errorf("column %s in %s has an unhandled type %v for label: %s ", columnName, namespace, v, columnData[idx]))
						labelValues[i] = "<invalid>"
						continue
					}

					// Prometheus will fail hard if the database and usernames are not UTF-8
					if !utf8.ValidString(labelValues[i]) {
						nonfatalErrors = append(nonfatalErrors, fmt.Errorf("column %s in %s has an invalid UTF-8 for a label: %s ", columnName, namespace, columnData[idx]))
						labelValues[i] = "<invalid>"
						continue
					}
				}
			}
		}

		// Loop over column names, and match to scan data. Unknown columns
		// will be filled with an untyped metric number *if* they can be
		// converted to float64s. NULLs are allowed and treated as NaN.
		for idx, columnName := range columnNames {
			if metricMapping, ok := mapping.columnMappings[columnName]; ok {
				// Is this a metricy metric?
				if metricMapping.discard {
					continue
				}

				value, ok := metricMapping.conversion(columnData[idx])
				if !ok {
					nonfatalErrors = append(nonfatalErrors, fmt.Errorf("unexpected error parsing namespace: %v, column: %v, index: %v", namespace, columnName, columnData[idx]))
					continue
				}
				// Generate the metric
				ch <- prometheus.MustNewConstMetric(metricMapping.desc, metricMapping.vtype, value, labelValues...)
			}
		}
	}
	if err := rows.Err(); err != nil {
		logger.Error("Failed scaning all rows", "err", err.Error())
		nonfatalErrors = append(nonfatalErrors, fmt.Errorf("failed to consume all rows due to: %w", err))
	}
	return nonfatalErrors, nil
}

func getDB(conn string) (*sql.DB, error) {
	// Open the database connection
	db, err := sql.Open("postgres", conn)
	if err != nil {
		return nil, err
	}

	// Configure connection pool settings
	db.SetMaxOpenConns(1)                  // Limit to single connection since we're dealing with PgBouncer
	db.SetMaxIdleConns(1)                  // Keep one idle connection ready
	db.SetConnMaxLifetime(5 * time.Minute) // Recycle connections every 5 minutes to prevent staleness
	db.SetConnMaxIdleTime(1 * time.Minute) // Close idle connections after 1 minute

	// Verify the connection is working
	rows, err := db.Query("SHOW STATS")
	if err != nil {
		db.Close() // Clean up if connection test fails
		return nil, fmt.Errorf("error pinging pgbouncer: %w", err)
	}
	defer rows.Close()

	return db, nil
}

// Convert database.sql types to float64s for Prometheus consumption. Null types are mapped to NaN. string and []byte
// types are mapped as NaN and !ok
func dbToFloat64(t interface{}, factor float64) (float64, bool) {
	switch v := t.(type) {
	case int64:
		return float64(v) * factor, true
	case float64:
		return v * factor, true
	case time.Time:
		return float64(v.Unix()), true
	case []byte:
		// Try and convert to string and then parse to a float64
		strV := string(v)
		result, err := strconv.ParseFloat(strV, 64)
		if err != nil {
			return math.NaN(), false
		}
		return result * factor, true
	case string:
		result, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return math.NaN(), false
		}
		return result * factor, true
	case nil:
		return math.NaN(), true
	default:
		return math.NaN(), false
	}
}

// Iterate through all the namespace mappings in the exporter and run their queries.
func (e *Exporter) queryNamespaceMappings(ch chan<- prometheus.Metric, db *sql.DB, metricMap map[string]MetricMapNamespace) map[string]error {
	// Return a map of namespace -> errors
	namespaceErrors := make(map[string]error)

	for namespace, mapping := range metricMap {
		e.logger.Debug("Querying namespace", "namespace", namespace)
		nonFatalErrors, err := queryNamespaceMapping(ch, db, namespace, mapping, e.filterEmptyPools, e.logger)
		// Serious error - a namespace disappeared
		if err != nil {
			namespaceErrors[namespace] = err
			e.logger.Info("namespace disappeared", "err", err.Error())
		}
		// Non-serious errors - likely version or parsing problems.
		if len(nonFatalErrors) > 0 {
			for _, err := range nonFatalErrors {
				e.logger.Info("error parsing", "err", err.Error())
			}
		}
	}

	return namespaceErrors
}

// Gather the pgbouncer version info.
func queryVersion(ch chan<- prometheus.Metric, db *sql.DB) error {
	rows, err := db.Query("SHOW VERSION;")
	if err != nil {
		return fmt.Errorf("error getting pgbouncer version: %w", err)
	}
	defer rows.Close()

	var columnNames []string
	columnNames, err = rows.Columns()
	if err != nil {
		return fmt.Errorf("error retrieving column list for version: %w", err)
	}
	if len(columnNames) != 1 || columnNames[0] != "version" {
		return errors.New("show version didn't return version column")
	}

	var bouncerVersion string

	for rows.Next() {
		err := rows.Scan(&bouncerVersion)
		if err != nil {
			return err
		}
		ch <- prometheus.MustNewConstMetric(
			bouncerVersionDesc,
			prometheus.GaugeValue,
			1.0,
			bouncerVersion,
		)
	}

	return nil
}

// Describe implements prometheus.Collector.
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	// We cannot know in advance what metrics the exporter will generate
	// from Postgres. So we use the poor man's describe method: Run a collect
	// and send the descriptors of all the collected metrics. The problem
	// here is that we need to connect to the Postgres DB. If it is currently
	// unavailable, the descriptors will be incomplete. Since this is a
	// stand-alone exporter and not used as a library within other code
	// implementing additional metrics, the worst that can happen is that we
	// don't detect inconsistent metrics created by this exporter
	// itself. Also, a change in the monitored Postgres instance may change the
	// exported metrics during the runtime of the exporter.

	metricCh := make(chan prometheus.Metric)
	doneCh := make(chan struct{})

	go func() {
		for m := range metricCh {
			ch <- m.Desc()
		}
		close(doneCh)
	}()

	e.Collect(metricCh)
	close(metricCh)
	<-doneCh
}

// Collect implements prometheus.Collector.
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	start := time.Now()
	e.logger.Info("Starting scrape", "timestamp", start)

	var up = 1.0

	// Version query timing
	versionStart := time.Now()
	err := queryVersion(ch, e.db)
	if err != nil {
		e.logger.Error("error getting version", "err", err.Error())
		up = 0
	}
	e.logger.Info("Version query completed",
		"duration_ms", time.Since(versionStart).Milliseconds(),
		"timestamp", time.Now())

	// SHOW LISTS timing
	listsStart := time.Now()
	if err = queryShowLists(ch, e.db, e.logger); err != nil {
		e.logger.Error("error getting SHOW LISTS", "err", err.Error())
		up = 0
	}
	e.logger.Info("SHOW LISTS query completed",
		"duration_ms", time.Since(listsStart).Milliseconds(),
		"timestamp", time.Now())

	// SHOW CONFIG timing
	configStart := time.Now()
	if err = queryShowConfig(ch, e.db, e.logger); err != nil {
		e.logger.Error("error getting SHOW CONFIG", "err", err.Error())
		up = 0
	}
	e.logger.Info("SHOW CONFIG query completed",
		"duration_ms", time.Since(configStart).Milliseconds(),
		"timestamp", time.Now())

	// Namespace mappings timing
	mappingsStart := time.Now()
	errMap := e.queryNamespaceMappings(ch, e.db, e.metricMap)
	if len(errMap) > 0 {
		e.logger.Error("error querying namespace mappings", "err", errMap)
		up = 0
	}
	e.logger.Info("Namespace mappings completed",
		"duration_ms", time.Since(mappingsStart).Milliseconds(),
		"timestamp", time.Now())

	if len(errMap) == len(e.metricMap) {
		up = 0
	}

	ch <- prometheus.MustNewConstMetric(scrapeSuccessDesc, prometheus.GaugeValue, up)

	e.logger.Info("Scrape completed",
		"total_duration_ms", time.Since(start).Milliseconds(),
		"timestamp", time.Now())
}

// Turn the MetricMap column mapping into a prometheus descriptor mapping.
func makeDescMap(metricMaps map[string]map[string]ColumnMapping, namespace string, logger *slog.Logger) map[string]MetricMapNamespace {
	var metricMap = make(map[string]MetricMapNamespace)

	for metricNamespace, mappings := range metricMaps {
		thisMap := make(map[string]MetricMap)
		var labels = make([]string, 0)

		// First collect all the labels since the metrics will need them
		for columnName, columnMapping := range mappings {
			if columnMapping.usage == LABEL {
				logger.Debug("Adding label", "column_name", columnName, "metric_namespace", metricNamespace)
				labels = append(labels, columnName)
			}
		}

		for columnName, columnMapping := range mappings {
			factor := columnMapping.factor

			// Determine how to convert the column based on its usage.
			switch columnMapping.usage {
			case COUNTER:
				thisMap[columnName] = MetricMap{
					vtype: prometheus.CounterValue,
					desc:  prometheus.NewDesc(fmt.Sprintf("%s_%s_%s", namespace, metricNamespace, columnMapping.metric), columnMapping.description, labels, nil),
					conversion: func(in interface{}) (float64, bool) {
						return dbToFloat64(in, factor)
					},
				}
			case GAUGE:
				thisMap[columnName] = MetricMap{
					vtype: prometheus.GaugeValue,
					desc:  prometheus.NewDesc(fmt.Sprintf("%s_%s_%s", namespace, metricNamespace, columnMapping.metric), columnMapping.description, labels, nil),
					conversion: func(in interface{}) (float64, bool) {
						return dbToFloat64(in, factor)
					},
				}
			}
		}

		metricMap[metricNamespace] = MetricMapNamespace{thisMap, labels}
	}

	return metricMap
}
