package collector

import (
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"libvirt.org/go/libvirt"
)

// ExporterCollector collects exporter self-monitoring metrics
type ExporterCollector struct {
	up                *prometheus.Desc
	lastScrapeTime    *prometheus.Desc
	scrapeDuration    *prometheus.Desc
	scrapeErrors      *prometheus.Desc
	domainsDiscovered *prometheus.Desc
	cacheHits         *prometheus.Desc
	cacheMisses       *prometheus.Desc
	buildVersion      *prometheus.Desc
	buildCommit       *prometheus.Desc

	// Internal state
	startTime         time.Time
	lastScrape        time.Time
	scrapeErrorsTotal uint64
	cacheHitsTotal    uint64
	cacheMissesTotal  uint64
	domainsFound      int

	collected uint32 // atomic flag
}

// NewExporterCollector creates a new ExporterCollector
func NewExporterCollector() *ExporterCollector {
	return &ExporterCollector{
		up: prometheus.NewDesc(
			"libvirt_exporter_up",
			"Whether the exporter is up and running (1=up, 0=down)",
			[]string{},
			nil,
		),
		lastScrapeTime: prometheus.NewDesc(
			"libvirt_exporter_last_scrape_timestamp_seconds",
			"Unix timestamp of the last successful scrape",
			[]string{},
			nil,
		),
		scrapeDuration: prometheus.NewDesc(
			"libvirt_exporter_scrape_duration_seconds",
			"Duration of the last scrape in seconds",
			[]string{},
			nil,
		),
		scrapeErrors: prometheus.NewDesc(
			"libvirt_exporter_scrape_errors_total",
			"Total number of scrape errors",
			[]string{},
			nil,
		),
		domainsDiscovered: prometheus.NewDesc(
			"libvirt_exporter_domains_discovered",
			"Number of domains discovered during the last scrape",
			[]string{},
			nil,
		),
		cacheHits: prometheus.NewDesc(
			"libvirt_exporter_cache_hits_total",
			"Total number of cache hits",
			[]string{},
			nil,
		),
		cacheMisses: prometheus.NewDesc(
			"libvirt_exporter_cache_misses_total",
			"Total number of cache misses",
			[]string{},
			nil,
		),
		buildVersion: prometheus.NewDesc(
			"libvirt_exporter_build_version",
			"Exporter build version",
			[]string{"version"},
			nil,
		),
		buildCommit: prometheus.NewDesc(
			"libvirt_exporter_build_commit",
			"Exporter build commit hash",
			[]string{"commit"},
			nil,
		),
		startTime: time.Now(),
	}
}

// Describe implements the prometheus.Collector interface for ExporterCollector
func (c *ExporterCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.up
	ch <- c.lastScrapeTime
	ch <- c.scrapeDuration
	ch <- c.scrapeErrors
	ch <- c.domainsDiscovered
	ch <- c.cacheHits
	ch <- c.cacheMisses
	ch <- c.buildVersion
	ch <- c.buildCommit
}

// Reset implements the Collector interface for ExporterCollector
func (c *ExporterCollector) Reset() {
	atomic.StoreUint32(&c.collected, 0)
}

// Collect implements the Collector interface for ExporterCollector
func (c *ExporterCollector) Collect(
	ch chan<- prometheus.Metric,
	conn *libvirt.Connect,
	domain *libvirt.Domain,
) {
	// Use atomic operation to ensure we only collect exporter metrics once per scrape
	if atomic.CompareAndSwapUint32(&c.collected, 0, 1) {
		c.collectExporterMetrics(ch, conn)
	}
}

// collectExporterMetrics collects exporter self-monitoring metrics
func (c *ExporterCollector) collectExporterMetrics(
	ch chan<- prometheus.Metric,
	conn *libvirt.Connect,
) {
	start := time.Now()

	// Check if connection is alive
	alive := false
	if conn != nil {
		var err error
		alive, err = conn.IsAlive()
		if err != nil {
			alive = false
		}
	}

	// Get current metrics
	scrapeErrors := atomic.LoadUint64(&c.scrapeErrorsTotal)
	cacheHits := atomic.LoadUint64(&c.cacheHitsTotal)
	cacheMisses := atomic.LoadUint64(&c.cacheMissesTotal)
	domainsFound := c.domainsFound

	// Calculate uptime (not used in metrics, but kept for future use)
	_ = time.Since(c.startTime).Seconds()

	// Set metrics
	var upValue float64
	if alive {
		upValue = 1.0
	}

	ch <- prometheus.MustNewConstMetric(
		c.up,
		prometheus.GaugeValue,
		upValue,
	)

	ch <- prometheus.MustNewConstMetric(
		c.lastScrapeTime,
		prometheus.GaugeValue,
		float64(c.lastScrape.Unix()),
	)

	ch <- prometheus.MustNewConstMetric(
		c.scrapeDuration,
		prometheus.GaugeValue,
		float64(time.Since(start).Seconds()),
	)

	ch <- prometheus.MustNewConstMetric(
		c.scrapeErrors,
		prometheus.CounterValue,
		float64(scrapeErrors),
	)

	ch <- prometheus.MustNewConstMetric(
		c.domainsDiscovered,
		prometheus.GaugeValue,
		float64(domainsFound),
	)

	ch <- prometheus.MustNewConstMetric(
		c.cacheHits,
		prometheus.CounterValue,
		float64(cacheHits),
	)

	ch <- prometheus.MustNewConstMetric(
		c.cacheMisses,
		prometheus.CounterValue,
		float64(cacheMisses),
	)

	// Build info (these would typically come from build-time variables)
	buildVersion := "unknown"
	buildCommit := "unknown"

	ch <- prometheus.MustNewConstMetric(
		c.buildVersion,
		prometheus.GaugeValue,
		1.0,
		buildVersion,
	)

	ch <- prometheus.MustNewConstMetric(
		c.buildCommit,
		prometheus.GaugeValue,
		1.0,
		buildCommit,
	)

	// Update last scrape time
	c.lastScrape = time.Now()
}

// RecordScrapeError records a scrape error
func (c *ExporterCollector) RecordScrapeError() {
	atomic.AddUint64(&c.scrapeErrorsTotal, 1)
}

// RecordCacheHit records a cache hit
func (c *ExporterCollector) RecordCacheHit() {
	atomic.AddUint64(&c.cacheHitsTotal, 1)
}

// RecordCacheMiss records a cache miss
func (c *ExporterCollector) RecordCacheMiss() {
	atomic.AddUint64(&c.cacheMissesTotal, 1)
}

// SetDomainsFound sets the number of domains found
func (c *ExporterCollector) SetDomainsFound(count int) {
	c.domainsFound = count
}