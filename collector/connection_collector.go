package collector

import (
	"log"
	"sync/atomic"

	"github.com/prometheus/client_golang/prometheus"
	"libvirt.org/go/libvirt"
)

// ConnectionCollector collects connection and host level metrics
type ConnectionCollector struct {
	// Connection metrics
	connectionAlive          *prometheus.Desc
	activeDomains            *prometheus.Desc
	inactiveDomains          *prometheus.Desc

	// Host metrics
	hostCPUCount             *prometheus.Desc
	hostMemoryTotal          *prometheus.Desc
	hostMemoryFree           *prometheus.Desc
	hostLibvirtVersion       *prometheus.Desc
	hostHypervisorVersion    *prometheus.Desc

	metricsCollector MetricsCollector
	
	// Used to ensure we only collect connection metrics once per scrape
	collected uint32 // atomic flag
}

// NewConnectionCollector creates a new ConnectionCollector
func NewConnectionCollector() *ConnectionCollector {
	return &ConnectionCollector{
		// Connection metrics
		connectionAlive: prometheus.NewDesc(
			"libvirt_connection_alive",
			"Whether the connection to libvirt is alive (1=alive, 0=dead)",
			nil, // No labels needed as there's only one connection
		),
		activeDomains: prometheus.NewDesc(
			"libvirt_active_domains",
			"Number of active domains",
			nil,
		),
		inactiveDomains: prometheus.NewDesc(
			"libvirt_inactive_domains",
			"Number of inactive domains",
			nil,
		),

		// Host metrics
		hostCPUCount: prometheus.NewDesc(
			"libvirt_host_cpu_count",
			"Number of CPU cores on the host",
			nil,
		),
		hostMemoryTotal: prometheus.NewDesc(
			"libvirt_host_memory_total_bytes",
			"Total memory on the host in bytes",
			nil,
		),
		hostMemoryFree: prometheus.NewDesc(
			"libvirt_host_memory_free_bytes",
			"Free memory on the host in bytes",
			nil,
		),
		hostLibvirtVersion: prometheus.NewDesc(
			"libvirt_host_libvirt_version",
			"Version of libvirt",
			nil,
		),
		hostHypervisorVersion: prometheus.NewDesc(
			"libvirt_host_hypervisor_version",
			"Version of the hypervisor",
			nil,
		),

		metricsCollector: NewLibvirtMetricsCollector(),
	}
}

// Describe implements the prometheus.Collector interface for ConnectionCollector
func (c *ConnectionCollector) Describe(ch chan<- *prometheus.Desc) {
	// Connection metrics
	ch <- c.connectionAlive
	ch <- c.activeDomains
	ch <- c.inactiveDomains

	// Host metrics
	ch <- c.hostCPUCount
	ch <- c.hostMemoryTotal
	ch <- c.hostMemoryFree
	ch <- c.hostLibvirtVersion
	ch <- c.hostHypervisorVersion
}

// Reset implements the Collector interface for ConnectionCollector
func (c *ConnectionCollector) Reset() {
	atomic.StoreUint32(&c.collected, 0)
}

// Collect implements the Collector interface for ConnectionCollector
func (c *ConnectionCollector) Collect(
	ch chan<- prometheus.Metric,
	conn *libvirt.Connect,
	domain *libvirt.Domain,
) {
	// Use atomic operation to ensure we only collect connection metrics once per scrape
	if atomic.CompareAndSwapUint32(&c.collected, 0, 1) {
		c.collectConnectionMetrics(ch, conn)
		c.collectHostMetrics(ch, conn)
	}
}

// collectConnectionMetrics collects connection-level metrics
func (c *ConnectionCollector) collectConnectionMetrics(
	ch chan<- prometheus.Metric,
	conn *libvirt.Connect,
) {
	metrics, err := c.metricsCollector.CollectConnectionStats(conn)
	if err != nil {
		log.Printf("Warning: Failed to collect connection metrics: %v", err)
		return
	}

	// Connection metrics
	var aliveValue float64
	if metrics.IsAlive {
		aliveValue = 1.0
	}

	ch <- prometheus.MustNewConstMetric(
		c.connectionAlive,
		prometheus.GaugeValue,
		aliveValue,
	)

	ch <- prometheus.MustNewConstMetric(
		c.activeDomains,
		prometheus.GaugeValue,
		float64(metrics.ActiveDomainCount),
	)

	ch <- prometheus.MustNewConstMetric(
		c.inactiveDomains,
		prometheus.GaugeValue,
		float64(metrics.InactiveDomainCount),
	)
}

// collectHostMetrics collects host-level metrics
func (c *ConnectionCollector) collectHostMetrics(
	ch chan<- prometheus.Metric,
	conn *libvirt.Connect,
) {
	metrics, err := c.metricsCollector.CollectHostStats(conn)
	if err != nil {
		log.Printf("Warning: Failed to collect host metrics: %v", err)
		return
	}

	// Host metrics
	ch <- prometheus.MustNewConstMetric(
		c.hostCPUCount,
		prometheus.GaugeValue,
		float64(metrics.TotalCPUCount),
	)

	ch <- prometheus.MustNewConstMetric(
		c.hostMemoryTotal,
		prometheus.GaugeValue,
		float64(metrics.TotalMemoryBytes),
	)

	ch <- prometheus.MustNewConstMetric(
		c.hostMemoryFree,
		prometheus.GaugeValue,
		float64(metrics.FreeMemoryBytes),
	)

	ch <- prometheus.MustNewConstMetric(
		c.hostLibvirtVersion,
		prometheus.GaugeValue,
		float64(metrics.LibvirtVersion),
	)

	ch <- prometheus.MustNewConstMetric(
		c.hostHypervisorVersion,
		prometheus.GaugeValue,
		float64(metrics.HypervisorVersion),
	)
}