// Copyright 2024 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific langu
package main

import (
	"errors"
	"github.com/google/go-cmp/cmp"
	"io/fs"
	"maps"
	"strings"
	"testing"
)

func TestDefaultConfig(t *testing.T) {

	config := NewDefaultConfig()

	MustConnectOnStartupWant := true
	if config.MustConnectOnStartup != MustConnectOnStartupWant {
		t.Errorf("MustConnectOnStartup does not match. Want: %v, Got: %v", MustConnectOnStartupWant, config.MustConnectOnStartup)
	}

	MetricsPathWant := "/metrics"
	if config.MetricsPath != MetricsPathWant {
		t.Errorf("MustConnectOnStartup does not match. Want: %v, Got: %v", MetricsPathWant, config.MustConnectOnStartup)
	}

}

func TestUnHappyFileConfig(t *testing.T) {

	var config *Config
	var err error

	config, err = NewConfigFromFile("")
	if config != nil || err != nil {
		t.Errorf("NewConfigFromFile should return nil for config and error if path is empty. Got: %v", err)
	}

	_, err = NewConfigFromFile("./testdata/i-do-not-exist.yaml")
	if errors.Is(err, fs.ErrNotExist) == false {
		t.Errorf("NewConfigFromFile should return fs.ErrNotExist error. Got: %v", err)
	}

	_, err = NewConfigFromFile("./testdata/parse_error.yaml")
	if err != nil && strings.Contains(err.Error(), "yaml: line") == false {
		t.Errorf("NewConfigFromFile should return yaml parse error. Got: %v", err)
	}

	_, err = NewConfigFromFile("./testdata/empty.yaml")
	if errors.Is(err, ErrNoPgbouncersConfigured) == false {
		t.Errorf("NewConfigFromFile should return ErrNoPgbouncersConfigured error. Got: %v", err)
	}

	_, err = NewConfigFromFile("./testdata/no-dsn.yaml")
	if errors.Is(err, ErrEmptyPgbouncersDSN) == false {
		t.Errorf("NewConfigFromFile should return ErrEmptyPgbouncersDSN error. Got: %v", err)
	}

}

func TestFileConfig(t *testing.T) {

	var config *Config
	var err error

	config, err = NewConfigFromFile("./testdata/config.yaml")
	if err != nil {
		t.Errorf("NewConfigFromFile() should not throw an error: %v", err)
	}

	MustConnectOnStartupWant := false
	if config.MustConnectOnStartup != MustConnectOnStartupWant {
		t.Errorf("MustConnectOnStartup does not match. Want: %v, Got: %v", MustConnectOnStartupWant, config.MustConnectOnStartup)
	}

	MetricsPathWant := "/prom"
	if config.MetricsPath != MetricsPathWant {
		t.Errorf("MustConnectOnStartup does not match. Want: %v, Got: %v", MetricsPathWant, config.MustConnectOnStartup)
	}

	CommonExtraLabelsWant := map[string]string{"environment": "sandbox"}
	if maps.Equal(config.ExtraLabels, CommonExtraLabelsWant) == false {
		t.Errorf("ExtraLabels does not match. Want: %v, Got: %v", CommonExtraLabelsWant, config.ExtraLabels)
	}

	pgWants := []PgBouncerConfig{
		{
			DSN:         "postgres://postgres:@localhost:6543/pgbouncer?sslmode=disable",
			PidFile:     "/var/run/pgbouncer1.pid",
			ExtraLabels: map[string]string{"pgbouncer_instance": "set1-0", "environment": "prod"},
		},
		{
			DSN:         "postgres://postgres:@localhost:6544/pgbouncer?sslmode=disable",
			PidFile:     "/var/run/pgbouncer2.pid",
			ExtraLabels: map[string]string{"pgbouncer_instance": "set1-1", "environment": "prod"},
		},
	}

	for i := range pgWants {
		if cmp.Equal(config.PgBouncers[i], pgWants[i]) == false {
			t.Errorf("PGBouncer config %d does not match. Want: %v, Got: %v", i, pgWants[i], config.PgBouncers[i])
		}
	}

	err = config.ValidateLabels()
	if err != nil {
		t.Errorf("ValidateLabels() throws an unexpected error: %v", err)
	}

}

func TestNotUniqueLabels(t *testing.T) {

	config := NewDefaultConfig()

	config.AddPgbouncerConfig("", "", map[string]string{"pgbouncer_instance": "set1-0", "environment": "prod"})
	config.AddPgbouncerConfig("", "", map[string]string{"pgbouncer_instance": "set1-0", "environment": "prod"})

	err := config.ValidateLabels()
	if err == nil {
		t.Errorf("ValidateLabels() did not throw an error ")
	}
	errorWant := "Every pgbouncer instance must have unique label values, found the following label=value combination multiple times: 'environment=prod,pgbouncer_instance=set1-0'"
	if err.Error() != errorWant {
		t.Errorf("ValidateLabels() did not throw the expected error.\n- Want: %s\n- Got: %s", errorWant, err.Error())
	}

}

func TestMissingLabels(t *testing.T) {

	config := NewDefaultConfig()

	config.AddPgbouncerConfig("", "", map[string]string{"pgbouncer_instance": "set1-0", "environment": "prod"})
	config.AddPgbouncerConfig("", "", map[string]string{"pgbouncer_instance": "set1-0"})

	err := config.ValidateLabels()
	if err == nil {
		t.Errorf("ValidateLabels() did not throw an error ")
	}
	errorWant := "Every pgbouncer instance must define the same extra labels, the label 'environment' is only found on 1 of the 2 instances"
	if err.Error() != errorWant {
		t.Errorf("ValidateLabels() did not throw the expected error.\n- Want: %s\n- Got: %s", errorWant, err.Error())
	}

}
