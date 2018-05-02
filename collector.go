package main

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"
	"unicode/utf8"

	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
)

var (
	metricMaps = map[string]map[string]ColumnMapping{
		"stats": {
			"database":          {LABEL, "N/A", 1, "N/A"},
			"total_query_count": {COUNTER, "queries_pooled_total", 1, "Total number of SQL queries pooled"},
			"total_query_time":  {COUNTER, "queries_duration_seconds", 1e-6, "Total number of seconds spent by pgbouncer when actively connected to PostgreSQL, executing queries"},
			"total_received":    {COUNTER, "received_bytes_total", 1, "Total volume in bytes of network traffic received by pgbouncer, shown as bytes"},
			"total_requests":    {COUNTER, "queries_total", 1, "Total number of SQL requests pooled by pgbouncer, shown as requests"},
			"total_sent":        {COUNTER, "sent_bytes_total", 1, "Total volume in bytes of network traffic sent by pgbouncer, shown as bytes"},
			"total_wait_time":   {COUNTER, "client_wait_seconds", 1e-6, "Time spent by clients waiting for a server in seconds"},
			"total_xact_count":  {COUNTER, "sql_transactions_pooled_total", 1, "Total number of SQL transactions pooled"},
			"total_xact_time":   {COUNTER, "server_in_transaction_seconds", 1e-6, "Total number of seconds spent by pgbouncer when connected to PostgreSQL in a transaction, either idle in transaction or executing queries"},
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
			"maxwait":    {GAUGE, "client_maxwait_seconds", 1, "Age of oldest unserved client connection, shown as second"},
		},
	}
)

func NewExporter(connectionString string, namespace string) *Exporter {

	db, err := getDB(connectionString)

	if err != nil {
		log.Fatal(err)
	}

	return &Exporter{
		metricMap: makeDescMap(metricMaps, namespace),
		namespace: namespace,
		db:        db,
		up: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "up",
			Help:      "Was the PgBouncer instance query successful?",
		}),

		duration: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "last_scrape_duration_seconds",
			Help:      "Duration of the last scrape of metrics from PgBouncer.",
		}),

		totalScrapes: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "scrapes_total",
			Help:      "Total number of times PgBouncer has been scraped for metrics.",
		}),

		error: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "last_scrape_error",
			Help:      "Whether the last scrape of metrics from PgBouncer resulted in an error (1 for error, 0 for success).",
		}),
	}
}

// Query within a namespace mapping and emit metrics. Returns fatal errors if
// the scrape fails, and a slice of errors if they were non-fatal.
func queryNamespaceMapping(ch chan<- prometheus.Metric, db *sql.DB, namespace string, mapping MetricMapNamespace) ([]error, error) {
	query := fmt.Sprintf("SHOW %s;", namespace)

	// Don't fail on a bad scrape of one metric
	rows, err := db.Query(query)
	if err != nil {
		return []error{}, errors.New(fmt.Sprintln("Error running query on database: ", namespace, err))
	}

	defer rows.Close()

	var columnNames []string
	columnNames, err = rows.Columns()
	if err != nil {
		return []error{}, errors.New(fmt.Sprintln("Error retrieving column list for: ", namespace, err))
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
			return []error{}, errors.New(fmt.Sprintln("Error retrieving rows:", namespace, err))
		}

		for i, label := range mapping.labels {
			for idx, columnName := range columnNames {
				if columnName == label {
					log.Debugf("Fetching data for row belonging to %s: %s", columnName, columnData[idx])
					labelValues[i] = columnData[idx].(string)

					// Prometheus will fail hard if the database and usernames are not UTF-8
					if !utf8.ValidString(labelValues[i]) {
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
					nonfatalErrors = append(nonfatalErrors, errors.New(fmt.Sprintln("Unexpected error parsing column: ", namespace, columnName, columnData[idx])))
					continue
				}
				// Generate the metric
				ch <- prometheus.MustNewConstMetric(metricMapping.desc, metricMapping.vtype, value, labelValues...)
			}
		}
	}
	return nonfatalErrors, nil
}

func getDB(conn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", conn)
	if err != nil {
		return nil, err
	}
	err = db.Ping()
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	return db, nil
}

// Convert database.sql to string for Prometheus labels. Null types are mapped to empty strings.
func dbToString(t interface{}) (string, bool) {
	switch v := t.(type) {
	case int64:
		return fmt.Sprintf("%v", v), true
	case float64:
		return fmt.Sprintf("%v", v), true
	case time.Time:
		return fmt.Sprintf("%v", v.Unix()), true
	case nil:
		return "", true
	case []byte:
		// Try and convert to string
		return string(v), true
	case string:
		return v, true
	default:
		return "", false
	}
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
		return result, true
	case string:
		result, err := strconv.ParseFloat(v, 64)
		if err != nil {
			log.Infoln("Could not parse string:", err)
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
func queryNamespaceMappings(ch chan<- prometheus.Metric, db *sql.DB, metricMap map[string]MetricMapNamespace) map[string]error {
	// Return a map of namespace -> errors
	namespaceErrors := make(map[string]error)

	for namespace, mapping := range metricMap {
		log.Debugln("Querying namespace: ", namespace)
		nonFatalErrors, err := queryNamespaceMapping(ch, db, namespace, mapping)
		// Serious error - a namespace disappeard
		if err != nil {
			namespaceErrors[namespace] = err
			log.Infoln(err)
		}
		// Non-serious errors - likely version or parsing problems.
		if len(nonFatalErrors) > 0 {
			for _, err := range nonFatalErrors {
				log.Infoln(err.Error())
			}
		}
	}

	return namespaceErrors
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
	e.scrape(ch)
	ch <- e.duration
	ch <- e.up
	ch <- e.totalScrapes
	ch <- e.error
}

func (e *Exporter) scrape(ch chan<- prometheus.Metric) {
	defer func(begun time.Time) {
		e.duration.Set(time.Since(begun).Seconds())
	}(time.Now())
	log.Info("Starting scrape")

	e.error.Set(0)
	e.totalScrapes.Inc()

	e.mutex.RLock()
	defer e.mutex.RUnlock()

	errMap := queryNamespaceMappings(ch, e.db, e.metricMap)
	if len(errMap) > 0 {
		log.Fatal(errMap)
		e.error.Set(1)
	}
}

// Turn the MetricMap column mapping into a prometheus descriptor mapping.
func makeDescMap(metricMaps map[string]map[string]ColumnMapping, namespace string) map[string]MetricMapNamespace {
	var metricMap = make(map[string]MetricMapNamespace)
	var labels = make([]string, 1)

	for metricNamespace, mappings := range metricMaps {
		thisMap := make(map[string]MetricMap)

		// First collect all the labels since the metrics will need them
		for columnName, columnMapping := range mappings {
			if columnMapping.usage == LABEL {
				labels = append(labels, columnName)
			}
		}

		for columnName, columnMapping := range mappings {
			// Determine how to convert the column based on its usage.
			switch columnMapping.usage {
			case COUNTER:
				thisMap[columnName] = MetricMap{
					vtype: prometheus.CounterValue,
					desc:  prometheus.NewDesc(fmt.Sprintf("%s_%s_%s", namespace, metricNamespace, columnMapping.metric), columnMapping.description, labels, nil),
					conversion: func(in interface{}) (float64, bool) {
						return dbToFloat64(in, columnMapping.factor)
					},
				}
			case GAUGE:
				thisMap[columnName] = MetricMap{
					vtype: prometheus.GaugeValue,
					desc:  prometheus.NewDesc(fmt.Sprintf("%s_%s_%s", namespace, metricNamespace, columnMapping.metric), columnMapping.description, labels, nil),
					conversion: func(in interface{}) (float64, bool) {
						return dbToFloat64(in, columnMapping.factor)
					},
				}
			}
		}

		metricMap[metricNamespace] = MetricMapNamespace{thisMap, labels}
	}

	return metricMap
}
