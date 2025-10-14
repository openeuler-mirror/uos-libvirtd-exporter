package collector

import (
	"log"

	"github.com/prometheus/client_golang/prometheus"
	"libvirt.org/go/libvirt"
)

// CPUCollector collects CPU statistics
type CPUCollector struct {
	vmVcpuMax        *prometheus.Desc
	vmVcpuCurrent    *prometheus.Desc
	vmCPUTimeTotal   *prometheus.Desc
	vmUserTime       *prometheus.Desc
	vmSystemTime     *prometheus.Desc
	vmStealTime      *prometheus.Desc
	metricsCollector MetricsCollector
}

// NewCPUCollector creates a new CPUCollector
func NewCPUCollector() *CPUCollector {
	return &CPUCollector{
		vmVcpuMax: prometheus.NewDesc(
			"libvirt_vm_vcpu_max",
			"Maximum vCPU count for the virtual machine",
			[]string{"domain", "uuid"},
			nil,
		),
		vmVcpuCurrent: prometheus.NewDesc(
			"libvirt_vm_vcpu_current",
			"Current vCPU count for the virtual machine",
			[]string{"domain", "uuid"},
			nil,
		),
		vmCPUTimeTotal: prometheus.NewDesc(
			"libvirt_vm_cpu_time_total_nanoseconds",
			"Total CPU time used by the virtual machine in nanoseconds",
			[]string{"domain", "uuid"},
			nil,
		),
		vmUserTime: prometheus.NewDesc(
			"libvirt_vm_cpu_user_time_nanoseconds",
			"Guest user CPU time in nanoseconds",
			[]string{"domain", "uuid"},
			nil,
		),
		vmSystemTime: prometheus.NewDesc(
			"libvirt_vm_cpu_system_time_nanoseconds",
			"Guest system CPU time in nanoseconds",
			[]string{"domain", "uuid"},
			nil,
		),
		vmStealTime: prometheus.NewDesc(
			"libvirt_vm_cpu_steal_time_nanoseconds",
			"vCPU steal time in nanoseconds",
			[]string{"domain", "uuid"},
			nil,
		),
		metricsCollector: NewLibvirtMetricsCollector(),
	}
}

// Describe implements the prometheus.Collector interface for CPUCollector
func (c *CPUCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.vmVcpuMax
	ch <- c.vmVcpuCurrent
	ch <- c.vmCPUTimeTotal
	ch <- c.vmUserTime
	ch <- c.vmSystemTime
	ch <- c.vmStealTime
}

// Collect implements the Collector interface for CPUCollector
func (c *CPUCollector) Collect(
	ch chan<- prometheus.Metric,
	conn *libvirt.Connect,
	domain *libvirt.Domain,
) {
	// Get domain info first to check if it's running
	domainInfo, err := domain.GetInfo()
	if err != nil {
		log.Printf("Warning: Failed to get domain info for CPU metrics: %v", err)
		return
	}

	// Only collect CPU metrics for running domains
	if domainInfo.State != libvirt.DOMAIN_RUNNING {
		// Silently skip non-running domains - this is expected behavior
		return
	}

	metrics, err := c.metricsCollector.CollectCPUStats(conn, domain)
	if err != nil {
		// Check if this is because domain is not running (expected for some operations)
		if lverr, ok := err.(libvirt.Error); ok && lverr.Code == libvirt.ERR_OPERATION_INVALID {
			// Domain stopped running between our check and metric collection - silently skip
			return
		}
		// For other errors, log with more context
		domainName, _ := domain.GetName()
		log.Printf("Warning: Failed to collect CPU metrics for domain '%s': %v", domainName, err)
		return
	}

	ch <- prometheus.MustNewConstMetric(
		c.vmVcpuMax,
		prometheus.GaugeValue,
		float64(metrics.VCPUsMax),
		metrics.Name,
		metrics.UUID,
	)

	ch <- prometheus.MustNewConstMetric(
		c.vmVcpuCurrent,
		prometheus.GaugeValue,
		float64(metrics.VCPUsCurrent),
		metrics.Name,
		metrics.UUID,
	)

	ch <- prometheus.MustNewConstMetric(
		c.vmCPUTimeTotal,
		prometheus.CounterValue,
		float64(metrics.CPUTime),
		metrics.Name,
		metrics.UUID,
	)

	// Only expose extended metrics if they are available
	if metrics.UserTime > 0 {
		ch <- prometheus.MustNewConstMetric(
			c.vmUserTime,
			prometheus.CounterValue,
			float64(metrics.UserTime),
			metrics.Name,
			metrics.UUID,
		)
	}

	if metrics.SystemTime > 0 {
		ch <- prometheus.MustNewConstMetric(
			c.vmSystemTime,
			prometheus.CounterValue,
			float64(metrics.SystemTime),
			metrics.Name,
			metrics.UUID,
		)
	}

	if metrics.StealTime > 0 {
		ch <- prometheus.MustNewConstMetric(
			c.vmStealTime,
			prometheus.CounterValue,
			float64(metrics.StealTime),
			metrics.Name,
			metrics.UUID,
		)
	}
}

// Reset implements the Collector interface
func (c *CPUCollector) Reset() {
	// No internal state to reset
}
