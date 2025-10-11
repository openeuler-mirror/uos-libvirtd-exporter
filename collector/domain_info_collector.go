package collector

import (
	"log"

	"github.com/prometheus/client_golang/prometheus"
	"libvirt.org/go/libvirt"
)

// DomainInfoCollector collects basic domain information
type DomainInfoCollector struct {
	vmStatus         *prometheus.Desc
	vmCPUTime        *prometheus.Desc
	vmMemoryCurrent  *prometheus.Desc
	vmMemoryMax      *prometheus.Desc
	vmUptime         *prometheus.Desc
	metricsCollector MetricsCollector
}

// NewDomainInfoCollector creates a new DomainInfoCollector
func NewDomainInfoCollector() *DomainInfoCollector {
	return &DomainInfoCollector{
		vmStatus: prometheus.NewDesc(
			"libvirt_vm_status",
			"Status of the virtual machine (1=running, 0=other)",
			[]string{"domain", "uuid"},
			nil,
		),
		vmCPUTime: prometheus.NewDesc(
			"libvirt_vm_cpu_time_seconds_total",
			"Total CPU time used by the virtual machine in seconds",
			[]string{"domain", "uuid"},
			nil,
		),
		vmMemoryCurrent: prometheus.NewDesc(
			"libvirt_vm_memory_current_bytes",
			"Current memory usage of the virtual machine in bytes",
			[]string{"domain", "uuid"},
			nil,
		),
		vmMemoryMax: prometheus.NewDesc(
			"libvirt_vm_memory_max_bytes",
			"Maximum memory allowed for the virtual machine in bytes",
			[]string{"domain", "uuid"},
			nil,
		),
		vmUptime: prometheus.NewDesc(
			"libvirt_vm_uptime_seconds",
			"Virtual machine uptime in seconds",
			[]string{"domain", "uuid"},
			nil,
		),
		metricsCollector: NewLibvirtMetricsCollector(),
	}
}

// Describe implements the prometheus.Collector interface for DomainInfoCollector
func (c *DomainInfoCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.vmStatus
	ch <- c.vmCPUTime
	ch <- c.vmMemoryCurrent
	ch <- c.vmMemoryMax
	ch <- c.vmUptime
}

// Collect implements the Collector interface for DomainInfoCollector
func (c *DomainInfoCollector) Collect(ch chan<- prometheus.Metric, conn *libvirt.Connect, domain *libvirt.Domain) {
	metrics, err := c.metricsCollector.CollectDomainInfo(conn, domain)
	if err != nil {
		log.Printf("Failed to collect domain info metrics: %v", err)
		return
	}

	// VM status metric
	ch <- prometheus.MustNewConstMetric(c.vmStatus, prometheus.GaugeValue, metrics.Status, metrics.Name, metrics.UUID)

	// CPU time metric
	ch <- prometheus.MustNewConstMetric(c.vmCPUTime, prometheus.CounterValue, metrics.CPUTime, metrics.Name, metrics.UUID)

	// Memory metrics
	ch <- prometheus.MustNewConstMetric(c.vmMemoryCurrent, prometheus.GaugeValue, metrics.MemoryCurrent, metrics.Name, metrics.UUID)
	ch <- prometheus.MustNewConstMetric(c.vmMemoryMax, prometheus.GaugeValue, metrics.MemoryMax, metrics.Name, metrics.UUID)

	// Only collect uptime for running domains
	if metrics.HasUptime {
		ch <- prometheus.MustNewConstMetric(c.vmUptime, prometheus.GaugeValue, metrics.Uptime, metrics.Name, metrics.UUID)
	}
}
