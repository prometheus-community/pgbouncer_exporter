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
	"testing"

	"log/slog"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/smartystreets/goconvey/convey"
)

type labelMap map[string]string

type MetricResult struct {
	labels     labelMap
	value      float64
	metricType dto.MetricType
}

func readMetric(m prometheus.Metric) MetricResult {
	pb := &dto.Metric{}
	m.Write(pb)
	labels := make(labelMap, len(pb.Label))
	for _, v := range pb.Label {
		labels[v.GetName()] = v.GetValue()
	}
	if pb.Gauge != nil {
		return MetricResult{labels: labels, value: pb.GetGauge().GetValue(), metricType: dto.MetricType_GAUGE}
	}
	if pb.Counter != nil {
		return MetricResult{labels: labels, value: pb.GetCounter().GetValue(), metricType: dto.MetricType_COUNTER}
	}
	if pb.Untyped != nil {
		return MetricResult{labels: labels, value: pb.GetUntyped().GetValue(), metricType: dto.MetricType_UNTYPED}
	}
	panic("Unsupported metric type")
}

func TestQueryShowList(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Error opening a stub db connection: %s", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{"key", "value"}).
		AddRow("dns_queries", -1).
		AddRow("databases", 1).
		AddRow("pools", 0).
		AddRow("users", 2)

	mock.ExpectQuery("SHOW LISTS;").WillReturnRows(rows)
	logger := &slog.Logger{}

	ch := make(chan prometheus.Metric)
	go func() {
		defer close(ch)
		if err := queryShowLists(ch, db, logger); err != nil {
			t.Errorf("Error running queryShowList: %s", err)
		}
	}()

	expected := []MetricResult{
		{labels: labelMap{}, metricType: dto.MetricType_GAUGE, value: -1},
		{labels: labelMap{}, metricType: dto.MetricType_GAUGE, value: 1},
		{labels: labelMap{}, metricType: dto.MetricType_GAUGE, value: 0},
		{labels: labelMap{}, metricType: dto.MetricType_GAUGE, value: 2},
	}

	convey.Convey("Metrics comparison", t, func() {
		for _, expect := range expected {
			m := readMetric(<-ch)
			convey.So(expect, convey.ShouldResemble, m)
		}
	})
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled exceptions: %s", err)
	}
}

func TestQueryShowConfig(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Error opening a stub db connection: %s", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{"key", "value", "default", "changeable"}).
		AddRow("max_client_conn", 1900, 100, true).
		AddRow("max_user_connections", 100, 100, true).
		AddRow("auth_type", "md5", "md5", true).
		AddRow("client_tls_ciphers", "default", "default", "yes")

	mock.ExpectQuery("SHOW CONFIG;").WillReturnRows(rows)
	logger := &slog.Logger{}

	ch := make(chan prometheus.Metric)
	go func() {
		defer close(ch)
		if err := queryShowConfig(ch, db, logger); err != nil {
			t.Errorf("Error running queryShowConfig: %s", err)
		}
	}()

	expected := []MetricResult{
		{labels: labelMap{}, metricType: dto.MetricType_GAUGE, value: 1900},
		{labels: labelMap{}, metricType: dto.MetricType_GAUGE, value: 100},
		{labels: labelMap{"method": "md5"}, metricType: dto.MetricType_GAUGE, value: 1},
	}
	convey.Convey("Metrics comparison", t, func() {
		for _, expect := range expected {
			m := readMetric(<-ch)
			convey.So(expect, convey.ShouldResemble, m)
		}
	})
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled exceptions: %s", err)
	}
}
