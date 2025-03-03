package main

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// MetricCache holds cached metrics and related metadata
type MetricCache struct {
	metrics     []prometheus.Metric
	lastUpdated time.Time
	updating    bool
	mu          sync.RWMutex
}

// CachedExporter wraps the original Exporter with caching capability
type CachedExporter struct {
	*Exporter
	cache         *MetricCache
	cacheInterval time.Duration
	ctx           context.Context
	cancel        context.CancelFunc
}

// NewCachedExporter creates a new exporter with caching capabilities
func NewCachedExporter(connectionString string, namespace string, logger *slog.Logger, filterEmptyPools bool, cacheInterval time.Duration) *CachedExporter {
	ctx, cancel := context.WithCancel(context.Background())

	cached := &CachedExporter{
		Exporter: NewExporter(connectionString, namespace, logger, filterEmptyPools),
		cache: &MetricCache{
			metrics: make([]prometheus.Metric, 0),
		},
		cacheInterval: cacheInterval,
		ctx:           ctx,
		cancel:        cancel,
	}

	// Start the background cache updater
	go cached.updateMetricsLoop()

	// Perform initial cache population
	cached.updateCache()

	return cached
}

// updateMetricsLoop runs a periodic update of the cached metrics
func (ce *CachedExporter) updateMetricsLoop() {
	ticker := time.NewTicker(ce.cacheInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ce.ctx.Done():
			return
		case <-ticker.C:
			ce.updateCache()
		}
	}
}

// updateCache performs the actual metrics collection and updates the cache
func (ce *CachedExporter) updateCache() {
	ce.cache.mu.Lock()
	if ce.cache.updating {
		ce.cache.mu.Unlock()
		return
	}
	ce.cache.updating = true
	ce.cache.mu.Unlock()

	// Create a channel to collect metrics
	ch := make(chan prometheus.Metric)
	done := make(chan struct{})

	// Collect metrics in a separate goroutine
	collected := make([]prometheus.Metric, 0)
	go func() {
		for metric := range ch {
			collected = append(collected, metric)
		}
		close(done)
	}()

	// Perform collection using parent's Collect
	ce.Exporter.Collect(ch)
	close(ch)
	<-done

	// Update the cache with new metrics
	ce.cache.mu.Lock()
	ce.cache.metrics = collected
	ce.cache.lastUpdated = time.Now()
	ce.cache.updating = false
	ce.cache.mu.Unlock()
}

// Collect implements prometheus.Collector interface using cached metrics
func (ce *CachedExporter) Collect(ch chan<- prometheus.Metric) {
	ce.cache.mu.RLock()
	defer ce.cache.mu.RUnlock()

	// Send cached metrics
	for _, m := range ce.cache.metrics {
		ch <- m
	}

	// Add a metric for cache age
	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "cache_age_seconds"),
			"Number of seconds since the metrics cache was last updated",
			nil, nil,
		),
		prometheus.GaugeValue,
		time.Since(ce.cache.lastUpdated).Seconds(),
	)
}

// Describe implements prometheus.Collector interface
func (ce *CachedExporter) Describe(ch chan<- *prometheus.Desc) {
	ce.Exporter.Describe(ch)
}

// Close stops the background updater
func (ce *CachedExporter) Close() {
	ce.cancel()
}
