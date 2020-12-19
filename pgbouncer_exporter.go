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
	"os"

	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promlog"
	"github.com/prometheus/common/promlog/flag"
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
	const pidFileHelpText = `Path to PgBouncer pid file.

	If provided, the standard process metrics get exported for the PgBouncer
	process, prefixed with 'pgbouncer_process_...'. The pgbouncer_process exporter
	needs to have read access to files owned by the PgBouncer process. Depends on
	the availability of /proc.

	https://prometheus.io/docs/instrumenting/writing_clientlibs/#process-metrics.`

	promlogConfig := &promlog.Config{}
	flag.AddFlags(kingpin.CommandLine, promlogConfig)

	var (
		connectionStringPointer = kingpin.Flag("pgBouncer.connectionString", "Connection string for accessing pgBouncer.").Default("postgres://postgres:@localhost:6543/pgbouncer?sslmode=disable").String()
		listenAddress           = kingpin.Flag("web.listen-address", "Address on which to expose metrics and web interface.").Default(":9127").String()
		metricsPath             = kingpin.Flag("web.telemetry-path", "Path under which to expose metrics.").Default("/metrics").String()
		pidFilePath             = kingpin.Flag("pgBouncer.pid-file", pidFileHelpText).Default("").String()
	)

	kingpin.Version(version.Print("pgbouncer_exporter"))
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	logger := promlog.New(promlogConfig)

	connectionString := *connectionStringPointer
	exporter := NewExporter(connectionString, namespace, logger)
	prometheus.MustRegister(exporter)
	prometheus.MustRegister(version.NewCollector("pgbouncer_exporter"))

	level.Info(logger).Log("msg", "Starting stackdriver_exporter", "version", version.Info())
	level.Info(logger).Log("msg", "Build context", "build_context", version.BuildContext())

	if *pidFilePath != "" {
		procExporter := prometheus.NewProcessCollector(
			prometheus.ProcessCollectorOpts{
				PidFn:     prometheus.NewPidFileFn(*pidFilePath),
				Namespace: namespace,
			},
		)
		prometheus.MustRegister(procExporter)
	}

	http.Handle(*metricsPath, promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(fmt.Sprintf(indexHTML, *metricsPath)))
	})

	level.Info(logger).Log("msg", "Listening on", "address", *listenAddress)
	if err := http.ListenAndServe(*listenAddress, nil); err != nil {
		level.Error(logger).Log("err", err)
		os.Exit(1)
	}
}
