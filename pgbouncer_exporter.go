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
