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
	"net/http"
	"net/url"
	"os"
)

const namespace = "pgbouncer"

const (
	_ = iota
	ExitCodeWebServerError
	ExitConfigFileReadError
	ExitConfigFileContentError
	ExitConfigError
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
		metricsPath = kingpin.Flag("web.telemetry-path", "Path under which to expose metrics.").Default(config.MetricsPath).String()
		probePath   = kingpin.Flag("web.probe-path", "Path under which to expose probe metrics.").Default(config.ProbePath).String()

		connectionStringPointer = kingpin.Flag("pgBouncer.connectionString", "Connection string for accessing pgBouncer.").Default("postgres://postgres:@localhost:6543/pgbouncer?sslmode=disable").Envar("PGBOUNCER_EXPORTER_CONNECTION_STRING").String()
		pidFilePath             = kingpin.Flag("pgBouncer.pid-file", pidFileHelpText).Default("").String()

		configFilePath = kingpin.Flag("config.file", cfgFileHelpText).Default("").String()
		err            error
	)

	toolkitFlags := kingpinflag.AddFlags(kingpin.CommandLine, ":9127")

	kingpin.Version(version.Print("pgbouncer_exporter"))
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	logger := promslog.New(promslogConfig)

	// Update config with values from the command-line parameters
	if metricsPath != nil && *metricsPath != "" {
		config.MetricsPath = *metricsPath
	}
	if probePath != nil && *probePath != "" {
		config.ProbePath = *probePath
	}
	if connectionStringPointer != nil && *connectionStringPointer != "" {
		config.DSN = *connectionStringPointer
	}
	if pidFilePath != nil && *pidFilePath != "" {
		config.PidFile = *pidFilePath
	}

	// Read and apply config from config file.
	// When using a config file legacy_mode is disabled by default.
	if configFilePath != nil && *configFilePath != "" {
		err = config.ReadFromFile(*configFilePath)
		if err != nil {
			logger.Error("Error reading config file", "file", *configFilePath, "err", err)
			os.Exit(ExitConfigFileReadError)
		}
	}

	if config.MetricsPath == config.ProbePath {
		logger.Error("Metrics and probe paths cannot be the same path", "metrics", config.MetricsPath, "probe", config.ProbePath)
		os.Exit(ExitConfigError)
	}

	if config.LegacyMode {

		logger.Info("Running in legacy mode")

		// In legacy mode start the exporter in single-target mode with one scrape endpoint en no probe option
		exporter := NewExporter(config.DSN, namespace, logger, config.MustConnectOnStartup)
		prometheus.MustRegister(exporter)

		if config.PidFile != "" {
			procExporter := collectors.NewProcessCollector(
				collectors.ProcessCollectorOpts{
					PidFn:     prometheus.NewPidFileFn(config.PidFile),
					Namespace: namespace,
				},
			)
			prometheus.MustRegister(procExporter)
		}
	} else {
		logger.Info("Running in multi-target mode")
		http.HandleFunc(config.ProbePath, func(w http.ResponseWriter, r *http.Request) {
			params := r.URL.Query()

			// Get DSN
			dsn := params.Get("dsn")
			dsnURL, err := url.Parse(dsn)
			if err != nil {
				logger.Warn("Error parsing dsn", "dsn", dsn, "err", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			// Apply Credential
			cred := params.Get("cred")
			if cred != "" {
				creds, err := config.GetCredentials(cred)
				if err != nil {
					logger.Error("Error getting credentials", "cred", cred, "err", err)
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				creds.UpdateDSN(dsnURL)
			}

			registry := prometheus.NewRegistry()
			err = registry.Register(NewExporter(dsnURL.String(), namespace, logger, false))
			if err != nil {
				logger.Error("Error registering exporter", "err", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
			h.ServeHTTP(w, r)
		})
	}

	prometheus.MustRegister(versioncollector.NewCollector("pgbouncer_exporter"))

	logger.Info("Starting pgbouncer_exporter", "version", version.Info())
	logger.Info("Build context", "build_context", version.BuildContext())

	http.Handle(config.MetricsPath, promhttp.Handler())

	if config.MetricsPath != "/" && config.MetricsPath != "" && config.ProbePath != "/" && config.ProbePath != "" {
		landingConfig := web.LandingConfig{
			Name:        "PgBouncer Exporter",
			Description: "Prometheus Exporter for PgBouncer servers",
			Version:     version.Info(),
			Links: []web.LandingLinks{
				{
					Address: config.MetricsPath,
					Text:    "Metrics",
				},
				{
					Address: config.ProbePath,
					Text:    "Probe",
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
