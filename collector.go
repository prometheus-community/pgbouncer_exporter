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
	"math"
	"os"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
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
			"reserve_pool:":       {GAUGE, "reserve_pool", 1, "Maximum number of additional connections for this database"},
			"pool_mode":           {LABEL, "N/A", 1, "N/A"},
			"max_connections":     {GAUGE, "max_connections", 1, "Maximum number of allowed connections for this database"},
			"current_connections": {GAUGE, "current_connections", 1, "Current number of connections for this database"},
			"paused":              {GAUGE, "paused", 1, "1 if this database is currently paused, else 0"},
			"disabled":            {GAUGE, "disabled", 1, "1 if this database is currently disabled, else 0"},
		},
		"stats": {
			"database":          {LABEL, "N/A", 1, "N/A"},
			"total_query_count": {COUNTER, "queries_pooled_total", 1, "Total number of SQL queries pooled"},
			"total_query_time":  {COUNTER, "queries_duration_seconds_total", 1e-6, "Total number of seconds spent by pgbouncer when actively connected to PostgreSQL, executing queries"},
			"total_received":    {COUNTER, "received_bytes_total", 1, "Total volume in bytes of network traffic received by pgbouncer, shown as bytes"},
			"total_requests":    {COUNTER, "queries_total", 1, "Total number of SQL requests pooled by pgbouncer, shown as requests"},
			"total_sent":        {COUNTER, "sent_bytes_total", 1, "Total volume in bytes of network traffic sent by pgbouncer, shown as bytes"},
			"total_wait_time":   {COUNTER, "client_wait_seconds_total", 1e-6, "Time spent by clients waiting for a server in seconds"},
			"total_xact_count":  {COUNTER, "sql_transactions_pooled_total", 1, "Total number of SQL transactions pooled"},
			"total_xact_time":   {COUNTER, "server_in_transaction_seconds_total", 1e-6, "Total number of seconds spent by pgbouncer when connected to PostgreSQL in a transaction, either idle in transaction or executing queries"},
		},
		"pools": {
			"database":   {LABEL, "N/A", 1, "N/A"},
			"user":       {LABEL, "N/A", 1, "N/A"},
			"cl_active":  {GAUGE, "client_active_connections", 1, "Client connections linked to server connection and able to process queries, shown as connection"},
			"cl_waiting": {GAUGE, "client_waiting_connections", 1, "Client connections waiting on a server connection, shown as connection"},
			"sv_active":  {GAUGE, "server_active_connections", 1, "Server connections linked to a client connection, shown as connection"},
			"sv_idle":    {GAUGE, "server_idle_connections", 1, "Server connections idle and ready for a client query, shown as connection"},
			"sv_used":    {GAUGE, "server_used_connections", 1, "Server connections idle more than server_check_delay, needing server_check_query, shown as connection"},
			"sv_tested":  {GAUGE, "server_testing_connections", 1, "Server connections currently running either server_reset_query or server_check_query, shown as connection"},
			"sv_login":   {GAUGE, "server_login_connections", 1, "Server connections currently in the process of logging in, shown as connection"},
			"maxwait":    {GAUGE, "client_maxwait_seconds", 1, "Age of oldest unserved client connection in full seconds"},
			"maxwait_us": {GAUGE, "client_maxwait_fractional", 1e-6, "The fractional part of client_maxwait_seconds"},
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

func NewExporter(connectionString string, namespace string, logger log.Logger) *Exporter {

	db, err := getDB(connectionString)

	if err != nil {
		level.Error(logger).Log("msg", "error setting up DB connection", "err", err)
		os.Exit(1)
	}

	return &Exporter{
		metricMap: makeDescMap(metricMaps, namespace, logger),
		db:        db,
		logger:    logger,
	}
}

// Query SHOW LISTS, which has a series of rows, not columns.
func queryShowLists(ch chan<- prometheus.Metric, db *sql.DB, logger log.Logger) error {
	rows, err := db.Query("SHOW LISTS;")
	if err != nil {
		return errors.New(fmt.Sprintln("error running SHOW LISTS on database: ", err))
	}
	defer rows.Close()

	columnNames, err := rows.Columns()
	if err != nil || len(columnNames) != 2 {
		return errors.New(fmt.Sprintln("error retrieving columns list from SHOW LISTS: ", err))
	}

	var list string
	var items sql.RawBytes
	for rows.Next() {
		if err = rows.Scan(&list, &items); err != nil {
			return errors.New(fmt.Sprintln("error retrieving SHOW LISTS rows:", err))
		}
		value, err := strconv.ParseFloat(string(items), 64)
		if err != nil {
			return errors.New(fmt.Sprintln("error parsing SHOW LISTS column: ", list, err))
		}
		if metric, ok := listsMap[list]; ok {
			ch <- prometheus.MustNewConstMetric(metric, prometheus.GaugeValue, value)
		} else {
			level.Debug(logger).Log("msg", "SHOW LISTS unknown list", "list", list)
		}
	}
	return nil
}

// Query within a namespace mapping and emit metrics. Returns fatal errors if
// the scrape fails, and a slice of errors if they were non-fatal.
func queryNamespaceMapping(ch chan<- prometheus.Metric, db *sql.DB, namespace string, mapping MetricMapNamespace, logger log.Logger) ([]error, error) {
	query := fmt.Sprintf("SHOW %s;", namespace)

	// Don't fail on a bad scrape of one metric
	rows, err := db.Query(query)
	if err != nil {
		return []error{}, errors.New(fmt.Sprintln("error running query on database: ", namespace, err))
	}

	defer rows.Close()

	var columnNames []string
	columnNames, err = rows.Columns()
	if err != nil {
		return []error{}, errors.New(fmt.Sprintln("error retrieving column list for: ", namespace, err))
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
			return []error{}, errors.New(fmt.Sprintln("error retrieving rows:", namespace, err))
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
					nonfatalErrors = append(nonfatalErrors, errors.New(fmt.Sprintln("unexpected error parsing column: ", namespace, columnName, columnData[idx])))
					continue
				}
				// Generate the metric
				ch <- prometheus.MustNewConstMetric(metricMapping.desc, metricMapping.vtype, value, labelValues...)
			}
		}
	}
	if err := rows.Err(); err != nil {
		level.Error(logger).Log("msg", "Failed scaning all rows", "err", err)
		nonfatalErrors = append(nonfatalErrors, fmt.Errorf("Failed to consume all rows due to: %s", err))
	}
	return nonfatalErrors, nil
}

func getDB(conn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", conn)
	if err != nil {
		return nil, err
	}
	rows, err := db.Query("SHOW STATS")
	if err != nil {
		return nil, fmt.Errorf("error pinging pgbouncer: %q", err)
	}
	defer rows.Close()

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

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
func queryNamespaceMappings(ch chan<- prometheus.Metric, db *sql.DB, metricMap map[string]MetricMapNamespace, logger log.Logger) map[string]error {
	// Return a map of namespace -> errors
	namespaceErrors := make(map[string]error)

	for namespace, mapping := range metricMap {
		level.Debug(logger).Log("msg", "Querying namespace", "namespace", namespace)
		nonFatalErrors, err := queryNamespaceMapping(ch, db, namespace, mapping, logger)
		// Serious error - a namespace disappeard
		if err != nil {
			namespaceErrors[namespace] = err
			level.Info(logger).Log("msg", "namespace disappeard", "err", err)
		}
		// Non-serious errors - likely version or parsing problems.
		if len(nonFatalErrors) > 0 {
			for _, err := range nonFatalErrors {
				level.Info(logger).Log("msg", "error parsing", "err", err.Error())
			}
		}
	}

	return namespaceErrors
}

// Gather the pgbouncer version info.
func queryVersion(ch chan<- prometheus.Metric, db *sql.DB) error {
	rows, err := db.Query("SHOW VERSION;")
	if err != nil {
		return fmt.Errorf("error getting pgbouncer version: %v", err)
	}
	defer rows.Close()

	var columnNames []string
	columnNames, err = rows.Columns()
	if err != nil {
		return fmt.Errorf("error retrieving column list for version: %v", err)
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
	level.Info(e.logger).Log("msg", "Starting scrape")

	var up float64 = 1.0

	err := queryVersion(ch, e.db)
	if err != nil {
		level.Error(e.logger).Log("msg", "error getting version", "err", err)
		up = 0
	}

	if err = queryShowLists(ch, e.db, e.logger); err != nil {
		level.Error(e.logger).Log("msg", "error getting SHOW LISTS", "err", err)
		up = 0
	}

	errMap := queryNamespaceMappings(ch, e.db, e.metricMap, e.logger)
	if len(errMap) > 0 {
		level.Error(e.logger).Log("msg", "error querying namespace mappings", "err", errMap)
		up = 0
	}

	if len(errMap) == len(e.metricMap) {
		up = 0
	}
	ch <- prometheus.MustNewConstMetric(scrapeSuccessDesc, prometheus.GaugeValue, up)
}

// Turn the MetricMap column mapping into a prometheus descriptor mapping.
func makeDescMap(metricMaps map[string]map[string]ColumnMapping, namespace string, logger log.Logger) map[string]MetricMapNamespace {
	var metricMap = make(map[string]MetricMapNamespace)

	for metricNamespace, mappings := range metricMaps {
		thisMap := make(map[string]MetricMap)
		var labels = make([]string, 0)

		// First collect all the labels since the metrics will need them
		for columnName, columnMapping := range mappings {
			if columnMapping.usage == LABEL {
				level.Debug(logger).Log("msg", "Adding label", "column_name", columnName, "metric_namespace", metricNamespace)
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
