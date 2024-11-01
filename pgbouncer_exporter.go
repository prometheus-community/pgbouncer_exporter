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
	"net/http"
	"os"

	"github.com/alecthomas/kingpin/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	versioncollector "github.com/prometheus/client_golang/prometheus/collectors/version"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promslog"
	"github.com/prometheus/common/promslog/flag"
	"github.com/prometheus/common/version"
	"github.com/prometheus/exporter-toolkit/web"
	"github.com/prometheus/exporter-toolkit/web/kingpinflag"
)

const namespace = "pgbouncer"

const (
	_ = iota
	ExitCodeWebServerError
	ExitConfigFileReadError
	ExitConfigFileContentError
)

func main() {
	const pidFileHelpText = `Path to PgBouncer pid file.

	If provided, the standard process metrics get exported for the PgBouncer
	process, prefixed with 'pgbouncer_process_...'. The pgbouncer_process exporter
	needs to have read access to files owned by the PgBouncer process. Depends on
	the availability of /proc.

	https://prometheus.io/docs/instrumenting/writing_clientlibs/#process-metrics.`

	const cfgFileHelpText = `Path to config file for multiple pgbouncer instances .

	If provided, the standard pgbouncer parameters, 'pgBouncer.connectionString' 
	and 'pgBouncer.pid-file', will be ignored and read from the config file`

	config := NewDefaultConfig()

	promslogConfig := &promslog.Config{}
	flag.AddFlags(kingpin.CommandLine, promslogConfig)

	var (
		connectionStringPointer = kingpin.Flag("pgBouncer.connectionString", "Connection string for accessing pgBouncer.").Default("postgres://postgres:@localhost:6543/pgbouncer?sslmode=disable").Envar("PGBOUNCER_EXPORTER_CONNECTION_STRING").String()
		metricsPath             = kingpin.Flag("web.telemetry-path", "Path under which to expose metrics.").Default(config.MetricsPath).String()
		pidFilePath             = kingpin.Flag("pgBouncer.pid-file", pidFileHelpText).Default("").String()

		configFilePath = kingpin.Flag("config.file", cfgFileHelpText).Default("").String()
		err            error
	)

	toolkitFlags := kingpinflag.AddFlags(kingpin.CommandLine, ":9127")

	kingpin.Version(version.Print("pgbouncer_exporter"))
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	logger := promslog.New(promslogConfig)

	// If config file is used, read config from file
	// Else use the legacy command-line parameters
	if configFilePath != nil && *configFilePath != "" {
		config, err = NewConfigFromFile(*configFilePath)
		if err != nil {
			logger.Error("Error reading config file", "file", *configFilePath, "err", err)
			os.Exit(ExitConfigFileReadError)
		}
	} else {
		config.AddPgbouncerConfig(*connectionStringPointer, *pidFilePath, nil)
		config.MetricsPath = *metricsPath
	}

	// When running multiple connection every connection must have the same labels but a unique value combination
	if err = config.ValidateLabels(); err != nil {
		logger.Error("Error while validating labels: ", "file", *configFilePath, "err", err)
		os.Exit(ExitConfigFileContentError)
	}

	// Add an exporter for each connection with the extra labels merged
	reg := prometheus.DefaultRegisterer
	for _, pgbouncer := range config.PgBouncers {

		// Merge the comment extra_labels with the extra_labels per connection
		extraLabels := config.MergedExtraLabels(pgbouncer.ExtraLabels)

		// Add base exporter
		exporter := NewExporter(pgbouncer.DSN, namespace, logger, config.MustConnectOnStartup)
		prometheus.WrapRegistererWith(extraLabels, reg).MustRegister(exporter)

		// Add process exporter
		if pgbouncer.PidFile != "" {
			procExporter := collectors.NewProcessCollector(
				collectors.ProcessCollectorOpts{
					PidFn:     prometheus.NewPidFileFn(pgbouncer.PidFile),
					Namespace: namespace,
				},
			)
			prometheus.WrapRegistererWith(extraLabels, reg).MustRegister(procExporter)
		}
	}

	prometheus.MustRegister(versioncollector.NewCollector("pgbouncer_exporter"))

	logger.Info("Starting pgbouncer_exporter", "version", version.Info())
	logger.Info("Build context", "build_context", version.BuildContext())

	http.Handle(config.MetricsPath, promhttp.Handler())
	if config.MetricsPath != "/" && config.MetricsPath != "" {
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
			logger.Error("Error creating landing page", "err", err)
			os.Exit(1)
		}
		http.Handle("/", landingPage)
	}

	srv := &http.Server{}
	if err := web.ListenAndServe(srv, toolkitFlags, logger); err != nil {
		logger.Error("Error starting server", "err", err)
		os.Exit(ExitCodeWebServerError)
	}
}
