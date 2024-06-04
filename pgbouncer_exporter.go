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
	"encoding/json"
	"net/http"
	"os"

	"github.com/alecthomas/kingpin/v2"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	versioncollector "github.com/prometheus/client_golang/prometheus/collectors/version"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promlog"
	"github.com/prometheus/common/promlog/flag"
	"github.com/prometheus/common/version"
	"github.com/prometheus/exporter-toolkit/web"
	"github.com/prometheus/exporter-toolkit/web/kingpinflag"
)

const namespace = "pgbouncer"

type Config struct {
	ConnectionString string `json:"pgBouncer.connectionString"`
	MetricsPath      string `json:"web.telemetry-path"`
	PidFilePath      string `json:"pgBouncer.pid-file"`
}

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
		metricsPath             = kingpin.Flag("web.telemetry-path", "Path under which to expose metrics.").Default("/metrics").String()
		pidFilePath             = kingpin.Flag("pgBouncer.pid-file", pidFileHelpText).Default("").String()
		configFilePath          = kingpin.Flag("config.file", "Path to config file.").Default("").String()
	)

	toolkitFlags := kingpinflag.AddFlags(kingpin.CommandLine, ":9127")

	kingpin.Version(version.Print("pgbouncer_exporter"))
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	logger := promlog.New(promlogConfig)

	// Read configuration from file if configFilePath is provided
	if *configFilePath != "" {
		file, err := os.ReadFile(*configFilePath)
		if err != nil {
			level.Error(logger).Log("err", "Error reading config file", "file", *configFilePath, "err", err)
			os.Exit(1)
		}
		var config Config
		if err := json.Unmarshal(file, &config); err != nil {
			level.Error(logger).Log("err", "Error parsing config file", "file", *configFilePath, "err", err)
			os.Exit(1)
		}
		// Override flags with config file values
		if config.ConnectionString != "" {
			*connectionStringPointer = config.ConnectionString
		}
		if config.MetricsPath != "" {
			*metricsPath = config.MetricsPath
		}
		if config.PidFilePath != "" {
			*pidFilePath = config.PidFilePath
		}
	}

	connectionString := *connectionStringPointer
	exporter := NewExporter(connectionString, namespace, logger)
	prometheus.MustRegister(exporter)
	prometheus.MustRegister(versioncollector.NewCollector("pgbouncer_exporter"))

	level.Info(logger).Log("msg", "Starting pgbouncer_exporter", "version", version.Info())
	level.Info(logger).Log("msg", "Build context", "build_context", version.BuildContext())

	if *pidFilePath != "" {
		procExporter := collectors.NewProcessCollector(
			collectors.ProcessCollectorOpts{
				PidFn:     prometheus.NewPidFileFn(*pidFilePath),
				Namespace: namespace,
			},
		)
		prometheus.MustRegister(procExporter)
	}

	http.Handle(*metricsPath, promhttp.Handler())
	if *metricsPath != "/" && *metricsPath != "" {
		landingConfig := web.LandingConfig{
			Name:        "PgBouncer Exporter",
			Description: "Prometheus Exporter for PgBouncer servers",
			Version:     version.Info(),
			Links: []web.LandingLinks{
				{
					Address: *metricsPath,
					Text:    "Metrics",
				},
			},
		}
		landingPage, err := web.NewLandingPage(landingConfig)
		if err != nil {
			level.Error(logger).Log("err", err)
			os.Exit(1)
		}
		http.Handle("/", landingPage)
	}

	srv := &http.Server{}
	if err := web.ListenAndServe(srv, toolkitFlags, logger); err != nil {
		level.Error(logger).Log("err", err)
		os.Exit(1)
	}
}
