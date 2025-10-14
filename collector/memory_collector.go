package collector

import (
	"log"

	"github.com/prometheus/client_golang/prometheus"
	"libvirt.org/go/libvirt"
)

// MemoryCollector collects memory statistics
type MemoryCollector struct {
	vmMemoryBalloon     *prometheus.Desc
	vmMemoryUnused      *prometheus.Desc
	vmMemoryAvailable   *prometheus.Desc
	vmMemoryRSS         *prometheus.Desc
	vmMemorySwapIn      *prometheus.Desc
	vmMemorySwapOut     *prometheus.Desc
	vmMemoryMajorFaults *prometheus.Desc
	vmMemoryMinorFaults *prometheus.Desc
	vmMemoryTotal       *prometheus.Desc
	metricsCollector    MetricsCollector
}

// NewMemoryCollector creates a new MemoryCollector
func NewMemoryCollector() *MemoryCollector {
	return &MemoryCollector{
		vmMemoryBalloon: prometheus.NewDesc(
			"libvirt_vm_memory_balloon_bytes",
			"Current balloon size in bytes",
			[]string{"domain", "uuid"},
			nil,
		),
		vmMemoryUnused: prometheus.NewDesc(
			"libvirt_vm_memory_unused_bytes",
			"Guest unused memory in bytes",
			[]string{"domain", "uuid"},
			nil,
		),
		vmMemoryAvailable: prometheus.NewDesc(
			"libvirt_vm_memory_available_bytes",
			"Guest available memory in bytes",
			[]string{"domain", "uuid"},
			nil,
		),
		vmMemoryRSS: prometheus.NewDesc(
			"libvirt_vm_memory_rss_bytes",
			"Resident set size in bytes",
			[]string{"domain", "uuid"},
			nil,
		),
		vmMemorySwapIn: prometheus.NewDesc(
			"libvirt_vm_memory_swap_in_bytes",
			"Memory swapped in bytes",
			[]string{"domain", "uuid"},
			nil,
		),
		vmMemorySwapOut: prometheus.NewDesc(
			"libvirt_vm_memory_swap_out_bytes",
			"Memory swapped out bytes",
			[]string{"domain", "uuid"},
			nil,
		),
		vmMemoryMajorFaults: prometheus.NewDesc(
			"libvirt_vm_memory_major_faults_total",
			"Major page faults",
			[]string{"domain", "uuid"},
			nil,
		),
		vmMemoryMinorFaults: prometheus.NewDesc(
			"libvirt_vm_memory_minor_faults_total",
			"Minor page faults",
			[]string{"domain", "uuid"},
			nil,
		),
		vmMemoryTotal: prometheus.NewDesc(
			"libvirt_vm_memory_total_bytes",
			"Total assigned memory in bytes",
			[]string{"domain", "uuid"},
			nil,
		),
		metricsCollector: NewLibvirtMetricsCollector(),
	}
}

// Describe implements the prometheus.Collector interface for MemoryCollector
func (c *MemoryCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.vmMemoryBalloon
	ch <- c.vmMemoryUnused
	ch <- c.vmMemoryAvailable
	ch <- c.vmMemoryRSS
	ch <- c.vmMemorySwapIn
	ch <- c.vmMemorySwapOut
	ch <- c.vmMemoryMajorFaults
	ch <- c.vmMemoryMinorFaults
	ch <- c.vmMemoryTotal
}

// Collect implements the Collector interface for MemoryCollector
func (c *MemoryCollector) Collect(
	ch chan<- prometheus.Metric,
	conn *libvirt.Connect,
	domain *libvirt.Domain,
) {
	// Get domain info first to check if it's running
	domainInfo, err := domain.GetInfo()
	if err != nil {
		log.Printf("Warning: Failed to get domain info for memory metrics: %v", err)
		return
	}

	// Only collect memory metrics for running domains
	if domainInfo.State != libvirt.DOMAIN_RUNNING {
		// Silently skip non-running domains - this is expected behavior
		return
	}

	metrics, err := c.metricsCollector.CollectMemoryStats(conn, domain)
	if err != nil {
		// Check if this is because domain is not running (expected for some operations)
		if lverr, ok := err.(libvirt.Error); ok && lverr.Code == libvirt.ERR_OPERATION_INVALID {
			// Domain stopped running between our check and metric collection - silently skip
			return
		}
		// For other errors, log with more context
		domainName, _ := domain.GetName()
		log.Printf("Warning: Failed to collect memory metrics for domain '%s': %v", domainName, err)
		return
	}

	// Convert KB to bytes (multiply by 1024)
	ch <- prometheus.MustNewConstMetric(
		c.vmMemoryBalloon,
		prometheus.GaugeValue,
		float64(metrics.BalloonSize*1024),
		metrics.Name,
		metrics.UUID,
	)

	ch <- prometheus.MustNewConstMetric(
		c.vmMemoryUnused,
		prometheus.GaugeValue,
		float64(metrics.Unused*1024),
		metrics.Name,
		metrics.UUID,
	)

	ch <- prometheus.MustNewConstMetric(
		c.vmMemoryAvailable,
		prometheus.GaugeValue,
		float64(metrics.Available*1024),
		metrics.Name,
		metrics.UUID,
	)

	ch <- prometheus.MustNewConstMetric(
		c.vmMemoryRSS,
		prometheus.GaugeValue,
		float64(metrics.RSS*1024),
		metrics.Name,
		metrics.UUID,
	)

	ch <- prometheus.MustNewConstMetric(
		c.vmMemorySwapIn,
		prometheus.CounterValue,
		float64(metrics.SwapIn*1024),
		metrics.Name,
		metrics.UUID,
	)

	ch <- prometheus.MustNewConstMetric(
		c.vmMemorySwapOut,
		prometheus.CounterValue,
		float64(metrics.SwapOut*1024),
		metrics.Name,
		metrics.UUID,
	)

	ch <- prometheus.MustNewConstMetric(
		c.vmMemoryMajorFaults,
		prometheus.CounterValue,
		float64(metrics.MajorFaults),
		metrics.Name,
		metrics.UUID,
	)

	ch <- prometheus.MustNewConstMetric(
		c.vmMemoryMinorFaults,
		prometheus.CounterValue,
		float64(metrics.MinorFaults),
		metrics.Name,
		metrics.UUID,
	)

	ch <- prometheus.MustNewConstMetric(
		c.vmMemoryTotal,
		prometheus.GaugeValue,
		float64(metrics.Total*1024),
		metrics.Name,
		metrics.UUID,
	)
}
