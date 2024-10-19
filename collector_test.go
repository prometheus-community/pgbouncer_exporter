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
	"github.com/blang/semver/v4"
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
	logger := slog.Default()

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
	logger := slog.Default()

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

func TestQueryVersion(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Error opening a stub db connection: %s", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{"version"}).
		AddRow("PgBouncer 1.23.1")

	mock.ExpectQuery("SHOW VERSION;").WillReturnRows(rows)

	ch := make(chan prometheus.Metric)
	go func() {
		defer close(ch)
		err := queryVersion(ch, db)
		if err != nil {
			t.Errorf("Error running queryShowConfig: %s", err)
		}
	}()

	expected := []MetricResult{
		{labels: labelMap{"version": "PgBouncer 1.23.1"}, metricType: dto.MetricType_GAUGE, value: 1},
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

func TestBadQueryVersion(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Error opening a stub db connection: %s", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{"version"}).
		AddRow("PgBouncer x.x.x")

	mock.ExpectQuery("SHOW VERSION;").WillReturnRows(rows)

	ch := make(chan prometheus.Metric)
	go func() {
		defer close(ch)
		err := queryVersion(ch, db)
		if err != nil {
			t.Errorf("Error running queryShowConfig: %s", err)
		}
	}()

	expected := []MetricResult{
		{labels: labelMap{"version": "PgBouncer x.x.x"}, metricType: dto.MetricType_GAUGE, value: 1},
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

func TestMakeDescMap(t *testing.T) {
	currentVersion := semver.MustParse("1.20.1")
	metricMap := map[string]ColumnMapping{
		"name":                {LABEL, "N/A", 1, "N/A", semver.Version{}},
		"host":                {LABEL, "N/A", 1, "N/A", semver.MustParse("1.21.0")},
		"port":                {LABEL, "N/A", 1, "N/A", semver.MustParse("1.9.0")},
		"pool_size":           {GAUGE, "pool_size", 1, "Maximum number of server connections", semver.MustParse("1.22.0")},
		"reserve_pool":        {GAUGE, "reserve_pool", 1, "Maximum number of additional connections for this database", semver.Version{}},
		"current_connections": {GAUGE, "current_connections", 1e-6, "Current number of connections for this database", semver.MustParse("1.7.0")},
		"total_query_count":   {COUNTER, "queries_pooled_total", 1, "Total number of SQL queries pooled", semver.Version{}},
	}
	metricMaps := map[string]map[string]ColumnMapping{
		"database": metricMap,
	}
	logger := slog.Default()

	convey.Convey("Test makeDescMap", t, func() {
		descMap := makeDescMap(metricMaps, "foo", logger, currentVersion)

		convey.So(descMap, convey.ShouldContainKey, "database")
		convey.So(descMap, convey.ShouldHaveLength, 1)

		convey.So(descMap["database"].labels, convey.ShouldHaveLength, 2)
		convey.So(descMap["database"].labels, convey.ShouldContain, "name")
		convey.So(descMap["database"].labels, convey.ShouldContain, "port")

		convey.So(descMap["database"].columnMappings, convey.ShouldHaveLength, 3)
		convey.So(descMap["database"].columnMappings, convey.ShouldContainKey, "reserve_pool")
		convey.So(descMap["database"].columnMappings["reserve_pool"].vtype, convey.ShouldEqual, prometheus.GaugeValue)
		convey.So(descMap["database"].columnMappings, convey.ShouldContainKey, "current_connections")
		convey.So(descMap["database"].columnMappings["current_connections"].vtype, convey.ShouldEqual, prometheus.GaugeValue)
		convey.So(descMap["database"].columnMappings, convey.ShouldContainKey, "total_query_count")
		convey.So(descMap["database"].columnMappings["total_query_count"].vtype, convey.ShouldEqual, prometheus.CounterValue)
	})
}
