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
	"os"
)

var (
	ErrorNoConfigFileGiven = errors.New("File path cannot be an empty string")
)

func index2human(index int) string {

	// Reduce value to last digit, with the exception for 11th,12th and 13th.
	// 22 => 2 => 22nd
	selector := index
	if index >= 14 {
		selector = index % 10
	}

	switch selector {
	case 1:
		return fmt.Sprintf("%dst", index)
	case 2:
		return fmt.Sprintf("%dnd", index)
	case 3:
		return fmt.Sprintf("%drd", index)
	default:
		return fmt.Sprintf("%dth", index)
	}
}

type DuplicateCredentialsKeyError struct {
	message string
	index   int
	first   int
}

func (e DuplicateCredentialsKeyError) Error() string {
	return fmt.Sprintf("%s credential has duplicate key '%s' (already defined by %s credential)", index2human(e.index), e.message, index2human(e.first))
}

func NewDefaultConfig() *Config {
	return &Config{
		MetricsPath:          "/metrics",
		ProbePath:            "/probe",
		Credentials:          make([]Credentials, 0),
		LegacyMode:           true,
		MustConnectOnStartup: true,
	}
}

func (c *Config) ReadFromFile(path string) error {
	var err error
	var data []byte
	if path == "" {
		return ErrorNoConfigFileGiven
	}
	// Turn off legacyMode
	c.LegacyMode = false

	data, err = os.ReadFile(path)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(data, c)
	if err != nil {
		return err
	}
	var credErr CredentialsErrorInterface
	keyCount := map[string]int{}
	for i, credential := range c.Credentials {
		if credErr = credential.Validate(); credErr != nil {
			credErr.SetIndex(i + 1)
			return credErr
		}
		if first, ok := keyCount[credential.GetKey()]; !ok {
			keyCount[credential.GetKey()] = i
		} else {
			return &DuplicateCredentialsKeyError{credential.GetKey(), i + 1, first + 1}
		}
	}

	return nil
}

type Config struct {
	MetricsPath          string        `yaml:"metrics_path"`
	ProbePath            string        `yaml:"probe_path"`
	Credentials          []Credentials `yaml:"credentials"`
	LegacyMode           bool          `yaml:"legacy_mode"`
	DSN                  string        `yaml:"dsn"`
	PidFile              string        `yaml:"pid_file"`
	MustConnectOnStartup bool          `yaml:"must_connect_on_startup"`
}

func (c *Config) GetCredentials(key string) (Credentials, error) {
	for _, cred := range c.Credentials {
		if cred.GetKey() == key {
			return cred, nil
		}
	}

	return Credentials{}, fmt.Errorf("credential %s not found", key)

}
