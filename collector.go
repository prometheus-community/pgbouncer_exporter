package main

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
)

var (
	metricMaps = map[string]map[string]ColumnMapping{
		"stats": {
			"requests_per_second":       {GAUGE, "The request rate, shown as request/second", nil},
			"bytes_received_per_second": {GAUGE, "The total network traffic received, shown as byte/second", nil},
			"bytes_sent_per_second":     {GAUGE, "The total network traffic sent, shown as byte/second", nil},
			"total_query_time":          {GAUGE, "Time spent by pgbouncer actively querying PostgreSQL, shown as microsecond", nil},
			"avg_req":                   {GAUGE, "The average number of requests per second in last stat period, shown as request/second", nil},
			"avg_recv":                  {GAUGE, "The client network traffic received, shown as byte/second", nil},
			"avg_sent":                  {GAUGE, "The client network traffic sent, shown as byte/second", nil},
			"avg_query":                 {GAUGE, "The average query duration, shown as microsecond", nil},
		},
		"pools": {
			"cl_active":  {GAUGE, "Client connections linked to server connection and able to process queries, shown as connection", nil},
			"cl_waiting": {GAUGE, "Client connections waiting on a server connection, shown as connection", nil},
			"sv_active":  {GAUGE, "Server connections linked to a client connection, shown as connection", nil},
			"sv_idle":    {GAUGE, "Server connections idle and ready for a client query, shown as connection", nil},
			"sv_used":    {GAUGE, "Server connections idle more than server_check_delay, needing server_check_query, shown as connection", nil},
			"sv_tested":  {GAUGE, "Server connections currently running either server_reset_query or server_check_query, shown as connection", nil},
			"sv_login":   {GAUGE, "Server connections currently in the process of logging in, shown as connection", nil},
			"maxwait":    {GAUGE, "Age of oldest unserved client connection, shown as second", nil},
		},
	}
)

// NewExporter returns an initialized Exporter.
func NewExporter(pgBouncerConnectionString string) *Exporter {

	// Init our exporter.
	return &Exporter{
		up: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "up",
			Help:      "Was the PgBouncer instance query successful?",
		}),

		duration: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "last_scrape_duration_seconds",
			Help:      "Duration of the last scrape of metrics from PostgresSQL.",
		}),

		totalScrapes: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "scrapes_total",
			Help:      "Total number of times PostgresSQL was scraped for metrics.",
		}),

		error: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "last_scrape_error",
			Help:      "Whether the last scrape of metrics from PostgreSQL resulted in an error (1 for error, 0 for success).",
		}),
	}
}

// Query within a namespace mapping and emit metrics. Returns fatal errors if
// the scrape fails, and a slice of errors if they were non-fatal.
func queryNamespaceMapping(ch chan<- prometheus.Metric, db *sql.DB, namespace string, mapping MetricMapNamespace) ([]error, error) {
	// Check for a query override for this namespace
	query, found := queryOverrides[namespace]
	if !found {
		// No query override - do a simple * search.
		query = fmt.Sprintf("SELECT * FROM %s;", namespace)
	}

	// Was this query disabled (i.e. nothing sensible can be queried on this
	// version of PostgreSQL?
	if query == "" {
		// Return success (no pertinent data)
		return []error{}, nil
	}

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
		err = rows.Scan(scanArgs...)
		if err != nil {
			return []error{}, errors.New(fmt.Sprintln("Error retrieving rows:", namespace, err))
		}

		// Get the label values for this row
		var labels = make([]string, len(mapping.labels))
		for idx, columnName := range mapping.labels {
			labels[idx], _ = dbToString(columnData[columnIdx[columnName]])
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
				// Generate the metric
				ch <- prometheus.MustNewConstMetric(metricMapping.desc, metricMapping.vtype, value, labels...)
			} else {
				// Unknown metric. Report as untyped if scan to float64 works, else note an error too.
				desc := prometheus.NewDesc(fmt.Sprintf("%s_%s", namespace, columnName), fmt.Sprintf("Unknown metric from %s", namespace), nil, nil)

				// Its not an error to fail here, since the values are
				// unexpected anyway.
				log.Debugln(columnName, labels)
				ch <- prometheus.MustNewConstMetric(desc, prometheus.UntypedValue, value, labels...)
			}
		}
	}
	return nonfatalErrors, nil
}

// Iterate through all the namespace mappings in the exporter and run their
// queries.
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
	ch <- e.totalScrapes
	ch <- e.error
}

func (e *Exporter) scrape(ch chan<- prometheus.Metric) {
	defer func(begun time.Time) {
		e.duration.Set(time.Since(begun).Seconds())
	}(time.Now())

	e.error.Set(0)
	e.totalScrapes.Inc()

	// Lock the exporter maps
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	errMap := queryNamespaceMappings(ch, e.db, e.metricMap)
	if len(errMap) > 0 {
		e.error.Set(1)
	}
}
