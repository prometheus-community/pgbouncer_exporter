package main

// Elasticsearch Node Stats Structs
import (
	"database/sql"
	"fmt"
	"sync"

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
func (this *columnUsage) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var value string
	if err := unmarshal(&value); err != nil {
		return err
	}

	columnUsage, err := stringTocolumnUsage(value)
	if err != nil {
		return err
	}

	*this = columnUsage
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
}

// Stores the prometheus metric description which a given column will be mapped
// to by the collector
type MetricMap struct {
	discard    bool                 // Should metric be discarded during mapping?
	vtype      prometheus.ValueType // Prometheus valuetype
	namespace  string
	desc       *prometheus.Desc                  // Prometheus descriptor
	conversion func(interface{}) (float64, bool) // Conversion function to turn PG result into float64
}

type ColumnMapping struct {
	usage       columnUsage `yaml:"usage"`
	metric      string      `yaml:"metric"`
	factor      float64     `yaml:"factor"`
	description string      `yaml:"description"`
}

// Exporter collects PgBouncer stats from the given server and exports
// them using the prometheus metrics package.
type Exporter struct {
	connectionString string
	namespace        string
	mutex            sync.RWMutex

	duration, up, error prometheus.Gauge
	totalScrapes        prometheus.Counter

	metricMap map[string]MetricMapNamespace

	db *sql.DB
}
