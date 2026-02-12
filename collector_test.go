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

func TestQueryShowDatabases(t *testing.T) {
	rows := sqlmock.NewRows([]string{"name", "host", "port", "database", "pool_size"}).
		AddRow("pg0_db", "10.10.10.1", "5432", "pg0", 20)

	expected := []MetricResult{
		{labels: labelMap{"name": "pg0_db", "host": "10.10.10.1", "port": "5432", "database": "pg0", "force_user": "", "pool_mode": ""}, metricType: dto.MetricType_GAUGE, value: 20},
	}

	testQueryNamespaceMapping(t, "databases", rows, expected)
}

func TestQueryShowStats(t *testing.T) {
	// columns are listed in the order PgBouncers exposes them, a value of -1 means pgbouncer_exporter does not expose this value as a metric
	rows := sqlmock.NewRows([]string{"database",
		"server_assignment_count",
		"xact_count", "query_count", "bytes_received", "bytes_sent",
		"xact_time", "query_time", "wait_time", "client_parse_count", "server_parse_count", "bind_count"}).
		AddRow("pg0", -1, 10, 40, 220, 460, 6, 8, 9, 5, 55, 555)

	// expected metrics are returned in the same order as the colums
	expected := []MetricResult{
		{labels: labelMap{"database": "pg0"}, metricType: dto.MetricType_COUNTER, value: -1},   // server_assignment_count
		{labels: labelMap{"database": "pg0"}, metricType: dto.MetricType_COUNTER, value: 10},   // xact_count
		{labels: labelMap{"database": "pg0"}, metricType: dto.MetricType_COUNTER, value: 40},   // query_count
		{labels: labelMap{"database": "pg0"}, metricType: dto.MetricType_COUNTER, value: 220},  // bytes_received
		{labels: labelMap{"database": "pg0"}, metricType: dto.MetricType_COUNTER, value: 460},  // bytes_sent
		{labels: labelMap{"database": "pg0"}, metricType: dto.MetricType_COUNTER, value: 6e-6}, // xact_time
		{labels: labelMap{"database": "pg0"}, metricType: dto.MetricType_COUNTER, value: 8e-6}, // query_time
		{labels: labelMap{"database": "pg0"}, metricType: dto.MetricType_COUNTER, value: 9e-6}, // wait_time
		{labels: labelMap{"database": "pg0"}, metricType: dto.MetricType_COUNTER, value: 5},    // client_parse_count
		{labels: labelMap{"database": "pg0"}, metricType: dto.MetricType_COUNTER, value: 55},   // server_parse_count
		{labels: labelMap{"database": "pg0"}, metricType: dto.MetricType_COUNTER, value: 555},  // bind_count
	}

	testQueryNamespaceMapping(t, "stats_totals", rows, expected)
}

func TestQueryShowPools(t *testing.T) {
	rows := sqlmock.NewRows([]string{"database", "user", "cl_active"}).
		AddRow("pg0", "postgres", 2)

	expected := []MetricResult{
		{labels: labelMap{"database": "pg0", "user": "postgres"}, metricType: dto.MetricType_GAUGE, value: 2},
	}

	testQueryNamespaceMapping(t, "pools", rows, expected)
}

func testQueryNamespaceMapping(t *testing.T, namespaceMapping string, rows *sqlmock.Rows, expected []MetricResult) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Error opening a stub db connection: %s", err)
	}
	defer db.Close()

	mock.ExpectQuery("SHOW " + namespaceMapping + ";").WillReturnRows(rows)

	logger := slog.Default()

	metricMap := makeDescMap(metricMaps, namespace, logger)

	ch := make(chan prometheus.Metric)
	go func() {
		defer close(ch)
		if _, err := queryNamespaceMapping(ch, db, namespaceMapping, metricMap[namespaceMapping], logger); err != nil {
			t.Errorf("Error running queryNamespaceMapping: %s", err)
		}
	}()

	convey.Convey("Metrics comparison", t, func() {
		for _, expect := range expected {
			m := readMetric(<-ch)
			convey.So(m, convey.ShouldResemble, expect)
		}
	})
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled exceptions: %s", err)
	}
}
