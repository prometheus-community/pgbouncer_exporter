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
	"fmt"
	"github.com/google/go-cmp/cmp"
	"io/fs"
	"strings"
	"testing"
)

func TestDefaultConfig(t *testing.T) {

	config := NewDefaultConfig()

	MetricsPathWant := "/metrics"
	if config.MetricsPath != MetricsPathWant {
		t.Errorf("MetricsPath does not match. Want: %v, Got: %v", MetricsPathWant, config.MetricsPath)
	}

	ProbePathWant := "/probe"
	if config.ProbePath != ProbePathWant {
		t.Errorf("ProbePath does not match. Want: %v, Got: %v", ProbePathWant, config.ProbePath)
	}

}

func TestUnHappyFileConfig(t *testing.T) {

	config := NewDefaultConfig()
	var err error

	err = config.ReadFromFile("")
	if errors.Is(err, ErrorNoConfigFileGiven) == false {
		t.Errorf("config.ReadFromFile should return ErrorNoConfigFileGiven error. Got: %v", err)
	}

	err = config.ReadFromFile("./testdata/i-do-not-exist.yaml")
	if errors.Is(err, fs.ErrNotExist) == false {
		t.Errorf("config.ReadFromFile should return fs.ErrNotExist error. Got: %v", err)
	}

	err = config.ReadFromFile("./testdata/parse_error.yaml")
	if err != nil && strings.Contains(err.Error(), "yaml: line") == false {
		t.Errorf("config.ReadFromFile should return yaml parse error. Got: %v", err)
	}

	err = config.ReadFromFile("./testdata/duplicate_creds.yaml")
	var dcke *DuplicateCredentialsKeyError
	if errors.As(err, &dcke) == false {
		t.Errorf("config.ReadFromFile should return DuplicateCredentialsKeyError error. Got: %v", err)
	}

	err = config.ReadFromFile("./testdata/invalid_creds.yaml")
	var ce *CredentialsError
	if errors.As(err, &ce) == false {
		t.Errorf("config.ReadFromFile should return CredentialsError error. Got: %v", err)
	} else if err.(*CredentialsError).field != "ssl.key" {
		t.Errorf("config.ReadFromFile should return CredentialsError for field key. Got: %v", err)
	}

	for i, v := range map[int]string{2: "2nd", 3: "3rd", 4: "4th", 15: "15th", 20: "20th", 30: "30th"} {
		err = DuplicateCredentialsKeyError{message: "test", index: i, first: 1}
		want := fmt.Sprintf("%s credential has duplicate key 'test' (already defined by 1st credential)", v)
		if err.Error() != want {
			t.Errorf("DuplicateCredentialsKeyError did not return expected string. Want %v, Got: %v", want, err.Error())
		}
	}

}

func TestFileConfig(t *testing.T) {

	config := NewDefaultConfig()
	var err error

	err = config.ReadFromFile("./testdata/config.yaml")
	if err != nil {
		t.Errorf("config.ReadFromFile() should not throw an error: %v", err)
	}

	MetricsPathWant := "/prom"
	if config.MetricsPath != MetricsPathWant {
		t.Errorf("MetricsPath does not match. Want: %v, Got: %v", MetricsPathWant, config.MetricsPath)
	}

	ProbePathWant := "/data"
	if config.ProbePath != ProbePathWant {
		t.Errorf("ProbePath does not match. Want: %v, Got: %v", ProbePathWant, config.ProbePath)
	}

	CredKeyWant := "cred_c"
	cred, err := config.GetCredentials(CredKeyWant)
	if err != nil {
		t.Errorf("config.GetCredentials() should not throw an error: %v", err)
	}
	if cred.GetKey() != "cred_c" {
		t.Errorf("Key of retreived credential does not match. Want: %v, Got: %v", CredKeyWant, cred.GetKey())
	}

	_, err = config.GetCredentials("cred_d")
	if err == nil {
		t.Errorf("config.GetCredentials should return error. Got: %v", err)
	}

	credWants := []Credentials{
		{
			Key:      "cred_a",
			Username: "user",
			Password: "pass",
		},
		{
			Key:      "",
			Username: "cred_b",
			Password: "pass",
		},
	}

	for i := range credWants {
		if cmp.Equal(config.Credentials[i], credWants[i]) == false {
			t.Errorf("Credentials config %d does not match. Want: %v, Got: %v", i, credWants[i], config.Credentials[i])
		}
	}

}
