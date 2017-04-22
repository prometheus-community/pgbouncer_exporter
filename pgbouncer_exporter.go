package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
	"github.com/prometheus/common/version"
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
		showVersion             = flag.Bool("version", false, "Print version information.")
		listenAddress           = flag.String("web.listen-address", ":9124", "Address on which to expose metrics and web interface.")
		connectionStringPointer = flag.String("pgBouncer.connectionString", "postgres://postgres:@localhost:6543/pgbouncer?sslmode=disable", "Address on which to expose metrics and web interface.")
		metricsPath             = flag.String("web.telemetry-path", "/metrics", "Path under which to expose metrics.")
	)

	flag.Parse()

	if *showVersion {
		fmt.Fprintln(os.Stdout, version.Print("pgbouncer_exporter"))
		os.Exit(0)
	}

	connectionString := *connectionStringPointer
	exporter := NewExporter(connectionString)
	prometheus.MustRegister(exporter)

	log.Infoln("Starting pgbouncer exporter version: ", version.Info())

	http.Handle(*metricsPath, prometheus.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(fmt.Sprintf(indexHTML, *metricsPath)))
	})

	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}
