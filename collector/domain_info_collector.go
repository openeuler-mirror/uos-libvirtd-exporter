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
	vmAutostart      *prometheus.Desc
	vmPersistent     *prometheus.Desc
	vmManagedSave    *prometheus.Desc
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
		vmAutostart: prometheus.NewDesc(
			"libvirt_vm_autostart_enabled",
			"Whether the virtual machine is set to autostart",
			[]string{"domain", "uuid"},
			nil,
		),
		vmPersistent: prometheus.NewDesc(
			"libvirt_vm_persistent",
			"Whether the virtual machine is persistent",
			[]string{"domain", "uuid"},
			nil,
		),
		vmManagedSave: prometheus.NewDesc(
			"libvirt_vm_managed_save",
			"Whether the virtual machine has a managed save image",
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
	ch <- c.vmAutostart
	ch <- c.vmPersistent
	ch <- c.vmManagedSave
}

// Collect implements the Collector interface for DomainInfoCollector
func (c *DomainInfoCollector) Collect(
	ch chan<- prometheus.Metric,
	conn *libvirt.Connect,
	domain *libvirt.Domain,
) {
	metrics, err := c.metricsCollector.CollectDomainInfo(conn, domain)
	if err != nil {
		log.Printf("Failed to collect domain info metrics: %v", err)
		return
	}

	// VM status metric
	ch <- prometheus.MustNewConstMetric(
		c.vmStatus,
		prometheus.GaugeValue,
		metrics.Status,
		metrics.Name,
		metrics.UUID,
	)

	// CPU time metric
	ch <- prometheus.MustNewConstMetric(
		c.vmCPUTime,
		prometheus.CounterValue,
		metrics.CPUTime,
		metrics.Name,
		metrics.UUID,
	)

	// Memory metrics
	ch <- prometheus.MustNewConstMetric(
		c.vmMemoryCurrent,
		prometheus.GaugeValue,
		metrics.MemoryCurrent,
		metrics.Name,
		metrics.UUID,
	)

	ch <- prometheus.MustNewConstMetric(
		c.vmMemoryMax,
		prometheus.GaugeValue,
		metrics.MemoryMax,
		metrics.Name,
		metrics.UUID,
	)

	// Autostart metric
	var autostartValue float64
	if metrics.Autostart {
		autostartValue = 1.0
	}
	ch <- prometheus.MustNewConstMetric(
		c.vmAutostart,
		prometheus.GaugeValue,
		autostartValue,
		metrics.Name,
		metrics.UUID,
	)

	// Persistent metric
	var persistentValue float64
	if metrics.Persistent {
		persistentValue = 1.0
	}
	ch <- prometheus.MustNewConstMetric(
		c.vmPersistent,
		prometheus.GaugeValue,
		persistentValue,
		metrics.Name,
		metrics.UUID,
	)

	// Managed save metric
	var managedSaveValue float64
	if metrics.ManagedSave {
		managedSaveValue = 1.0
	}
	ch <- prometheus.MustNewConstMetric(
		c.vmManagedSave,
		prometheus.GaugeValue,
		managedSaveValue,
		metrics.Name,
		metrics.UUID,
	)

	// Only collect uptime for running domains
	if metrics.HasUptime {
		ch <- prometheus.MustNewConstMetric(
			c.vmUptime,
			prometheus.GaugeValue,
			metrics.Uptime,
			metrics.Name,
			metrics.UUID,
		)
	}
}

// Reset implements the Collector interface
func (c *DomainInfoCollector) Reset() {
	// No internal state to reset
}
