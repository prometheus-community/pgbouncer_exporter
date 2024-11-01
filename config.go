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
	"errors"
	"fmt"
	"gopkg.in/yaml.v3"
	"maps"
	"os"
	"slices"
	"strings"
)

var (
	ErrNoPgbouncersConfigured = errors.New("no pgbouncer instances configured")
	ErrEmptyPgbouncersDSN     = errors.New("atleast one pgbouncer instance has an empty dsn configured")
)

func NewDefaultConfig() *Config {
	return &Config{
		MustConnectOnStartup: true,
		ExtraLabels:          map[string]string{},
		MetricsPath:          "/metrics",
		PgBouncers:           []PgBouncerConfig{},
	}
}

func NewConfigFromFile(path string) (*Config, error) {
	var err error
	var data []byte
	if path == "" {
		return nil, nil
	}
	config := NewDefaultConfig()

	data, err = os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(data, config)
	if err != nil {
		return nil, err
	}

	if len(config.PgBouncers) == 0 {
		return nil, ErrNoPgbouncersConfigured
	}

	for _, instance := range config.PgBouncers {
		if strings.TrimSpace(instance.DSN) == "" {
			return nil, ErrEmptyPgbouncersDSN
		}
	}

	return config, nil

}

type Config struct {
	MustConnectOnStartup bool              `yaml:"must_connect_on_startup"`
	ExtraLabels          map[string]string `yaml:"extra_labels"`
	PgBouncers           []PgBouncerConfig `yaml:"pgbouncers"`
	MetricsPath          string            `yaml:"metrics_path"`
}
type PgBouncerConfig struct {
	DSN         string            `yaml:"dsn"`
	PidFile     string            `yaml:"pid-file"`
	ExtraLabels map[string]string `yaml:"extra_labels"`
}

func (p *Config) AddPgbouncerConfig(dsn string, pidFilePath string, extraLabels map[string]string) {
	p.PgBouncers = append(
		p.PgBouncers,
		PgBouncerConfig{
			DSN:         dsn,
			PidFile:     pidFilePath,
			ExtraLabels: extraLabels,
		},
	)
}

func (p *Config) MergedExtraLabels(extraLabels map[string]string) map[string]string {
	mergedLabels := make(map[string]string)
	maps.Copy(mergedLabels, p.ExtraLabels)
	maps.Copy(mergedLabels, extraLabels)

	return mergedLabels
}

func (p Config) ValidateLabels() error {

	var labels = make(map[string]int)
	var keys = make(map[string]int)
	for _, cfg := range p.PgBouncers {

		var slabels []string

		for k, v := range p.MergedExtraLabels(cfg.ExtraLabels) {
			slabels = append(slabels, fmt.Sprintf("%s=%s", k, v))
			keys[k]++
		}
		slices.Sort(slabels)
		hash := strings.Join(slabels, ",")
		if _, ok := labels[hash]; ok {
			return fmt.Errorf("Every pgbouncer instance must have unique label values,"+
				" found the following label=value combination multiple times: '%s'", hash)
		}
		labels[hash] = 1
	}

	for k, amount := range keys {
		if amount != len(p.PgBouncers) {
			return fmt.Errorf("Every pgbouncer instance must define the same extra labels,"+
				" the label '%s' is only found on %d of the %d instances", k, amount, len(p.PgBouncers))
		}
	}

	return nil

}
