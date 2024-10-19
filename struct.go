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

// Elasticsearch Node Stats Structs
import (
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/blang/semver/v4"
	"github.com/prometheus/client_golang/prometheus"
)

type columnUsage int

// convert a string to the corresponding columnUsage
func stringTocolumnUsage(s string) (u columnUsage, err error) {
	switch s {
	case "DISCARD":
		u = DISCARD

	case "LABEL":
		u = LABEL

	case "COUNTER":
		u = COUNTER

	case "GAUGE":
		u = GAUGE

	case "MAPPEDMETRIC":
		u = MAPPEDMETRIC

	case "DURATION":
		u = DURATION

	default:
		err = fmt.Errorf("wrong columnUsage given : %s", s)
	}

	return
}

// Implements the yaml.Unmarshaller interface
func (cu *columnUsage) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var value string
	if err := unmarshal(&value); err != nil {
		return err
	}

	columnUsage, err := stringTocolumnUsage(value)
	if err != nil {
		return err
	}

	*cu = columnUsage
	return nil
}

const (
	DISCARD      columnUsage = iota // Ignore this column
	LABEL        columnUsage = iota // Use this column as a label
	COUNTER      columnUsage = iota // Use this column as a counter
	GAUGE        columnUsage = iota // Use this column as a gauge
	MAPPEDMETRIC columnUsage = iota // Use this column with the supplied mapping of text values
	DURATION     columnUsage = iota // This column should be interpreted as a text duration (and converted to milliseconds)
)

// Groups metric maps under a shared set of labels
type MetricMapNamespace struct {
	columnMappings map[string]MetricMap // Column mappings in this namespace
	labels         []string
}

// Stores the prometheus metric description which a given column will be mapped
// to by the collector
type MetricMap struct {
	discard    bool                              // Should metric be discarded during mapping?
	vtype      prometheus.ValueType              // Prometheus valuetype
	desc       *prometheus.Desc                  // Prometheus descriptor
	conversion func(interface{}) (float64, bool) // Conversion function to turn PG result into float64
}

type ColumnMapping struct {
	usage       columnUsage    `yaml:"usage"`
	metric      string         `yaml:"metric"`
	factor      float64        `yaml:"factor"`
	description string         `yaml:"description"`
	minVersion  semver.Version `yaml:"min_version"`
}

// Exporter collects PgBouncer stats from the given server and exports
// them using the prometheus metrics package.
type Exporter struct {
	metricMap map[string]MetricMapNamespace

	db *sql.DB

	logger *slog.Logger

	version semver.Version
}
