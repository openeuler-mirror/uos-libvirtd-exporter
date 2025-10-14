package collector

import (
	"log"

	"github.com/prometheus/client_golang/prometheus"
	"libvirt.org/go/libvirt"
)

// DiskCollector collects disk I/O statistics
type DiskCollector struct {
	vmDiskReadBytes  *prometheus.Desc
	vmDiskWriteBytes *prometheus.Desc
	vmDiskReadOps    *prometheus.Desc
	vmDiskWriteOps   *prometheus.Desc
	vmDiskReadTime   *prometheus.Desc
	vmDiskWriteTime  *prometheus.Desc
	metricsCollector MetricsCollector
}

// NewDiskCollector creates a new DiskCollector
func NewDiskCollector() *DiskCollector {
	return &DiskCollector{
		vmDiskReadBytes: prometheus.NewDesc(
			"libvirt_vm_disk_read_bytes_total",
			"Total bytes read from disk by the virtual machine",
			[]string{"domain", "uuid", "device"},
			nil,
		),
		vmDiskWriteBytes: prometheus.NewDesc(
			"libvirt_vm_disk_write_bytes_total",
			"Total bytes written to disk by the virtual machine",
			[]string{"domain", "uuid", "device"},
			nil,
		),
		vmDiskReadOps: prometheus.NewDesc(
			"libvirt_vm_disk_read_ops_total",
			"Total disk read operations by the virtual machine",
			[]string{"domain", "uuid", "device"},
			nil,
		),
		vmDiskWriteOps: prometheus.NewDesc(
			"libvirt_vm_disk_write_ops_total",
			"Total disk write operations by the virtual machine",
			[]string{"domain", "uuid", "device"},
			nil,
		),
		vmDiskReadTime: prometheus.NewDesc(
			"libvirt_vm_disk_read_time_seconds_total",
			"Total time spent reading from disk by the virtual machine",
			[]string{"domain", "uuid", "device"},
			nil,
		),
		vmDiskWriteTime: prometheus.NewDesc(
			"libvirt_vm_disk_write_time_seconds_total",
			"Total time spent writing to disk by the virtual machine",
			[]string{"domain", "uuid", "device"},
			nil,
		),
		metricsCollector: NewLibvirtMetricsCollector(),
	}
}

// Describe implements the prometheus.Collector interface for DiskCollector
func (c *DiskCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.vmDiskReadBytes
	ch <- c.vmDiskWriteBytes
	ch <- c.vmDiskReadOps
	ch <- c.vmDiskWriteOps
	ch <- c.vmDiskReadTime
	ch <- c.vmDiskWriteTime
}

// Collect implements the Collector interface for DiskCollector
func (c *DiskCollector) Collect(
	ch chan<- prometheus.Metric,
	conn *libvirt.Connect,
	domain *libvirt.Domain,
) {
	// Get domain info first to check if it's running
	domainInfo, err := domain.GetInfo()
	if err != nil {
		log.Printf("Warning: Failed to get domain info for disk metrics: %v", err)
		return
	}

	// Only collect disk metrics for running domains
	if domainInfo.State != libvirt.DOMAIN_RUNNING {
		// Silently skip non-running domains - this is expected behavior
		return
	}

	metricsList, err := c.metricsCollector.CollectDiskStats(conn, domain)
	if err != nil {
		// Check if this is because domain is not running (expected for some operations)
		if lverr, ok := err.(libvirt.Error); ok && lverr.Code == libvirt.ERR_OPERATION_INVALID {
			// Domain stopped running between our check and metric collection - silently skip
			return
		}
		// For other errors, log with more context
		domainName, _ := domain.GetName()
		log.Printf("Warning: Failed to collect disk metrics for domain '%s': %v", domainName, err)
		return
	}

	for _, metrics := range metricsList {
		ch <- prometheus.MustNewConstMetric(
			c.vmDiskReadBytes,
			prometheus.CounterValue,
			float64(metrics.ReadBytes),
			metrics.Name,
			metrics.UUID,
			metrics.Device,
		)

		ch <- prometheus.MustNewConstMetric(
			c.vmDiskWriteBytes,
			prometheus.CounterValue,
			float64(metrics.WriteBytes),
			metrics.Name,
			metrics.UUID,
			metrics.Device,
		)

		ch <- prometheus.MustNewConstMetric(
			c.vmDiskReadOps,
			prometheus.CounterValue,
			float64(metrics.ReadOps),
			metrics.Name,
			metrics.UUID,
			metrics.Device,
		)

		ch <- prometheus.MustNewConstMetric(
			c.vmDiskWriteOps,
			prometheus.CounterValue,
			float64(metrics.WriteOps),
			metrics.Name,
			metrics.UUID,
			metrics.Device,
		)

		// Only expose time metrics if they are available
		if metrics.ReadTimeNs > 0 || metrics.WriteTimeNs > 0 {
			ch <- prometheus.MustNewConstMetric(
				c.vmDiskReadTime,
				prometheus.CounterValue,
				float64(metrics.ReadTimeNs)/1e9,
				metrics.Name,
				metrics.UUID,
				metrics.Device,
			)

			ch <- prometheus.MustNewConstMetric(
				c.vmDiskWriteTime,
				prometheus.CounterValue,
				float64(metrics.WriteTimeNs)/1e9,
				metrics.Name,
				metrics.UUID,
				metrics.Device,
			)
		}
	}
}
