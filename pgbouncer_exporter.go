package main

import (
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
	"github.com/prometheus/common/version"
	"gopkg.in/alecthomas/kingpin.v2"
)

const (
	namespace = "pgbouncer"
	indexHTML = `
	<html>
		<head>
			<title>PgBouncer Exporter</title>
		</head>
		<body>
			<h1>PgBouncer Exporter</h1>
			<p>
			<a href='%s'>Metrics</a>
			</p>
		</body>
	</html>`
)

func main() {
	var (
		connectionStringPointer = kingpin.Flag("pgBouncer.connectionString", "Connection string for accessing pgBouncer.").Default("postgres://postgres:@localhost:6543/pgbouncer?sslmode=disable").String()
		listenAddress           = kingpin.Flag("web.listen-address", "Address on which to expose metrics and web interface.").Default(":9127").String()
		metricsPath             = kingpin.Flag("web.telemetry-path", "Path under which to expose metrics.").Default("/metrics").String()
	)

	log.AddFlags(kingpin.CommandLine)
	kingpin.Version(version.Print("pgbouncer_exporter"))
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	connectionString := *connectionStringPointer
	exporter := NewExporter(connectionString, namespace)
	prometheus.MustRegister(exporter)
	prometheus.MustRegister(version.NewCollector("pgbouncer_exporter"))

	log.Infoln("Starting pgbouncer exporter version: ", version.Info())

	http.Handle(*metricsPath, promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(fmt.Sprintf(indexHTML, *metricsPath)))
	})

	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}
