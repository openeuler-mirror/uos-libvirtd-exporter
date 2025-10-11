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
		metricsCollector: NewLibvirtMetricsCollector(),
	}
}

// Describe implements the prometheus.Collector interface for DiskCollector
func (c *DiskCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.vmDiskReadBytes
	ch <- c.vmDiskWriteBytes
	ch <- c.vmDiskReadOps
	ch <- c.vmDiskWriteOps
}

// Collect implements the Collector interface for DiskCollector
func (c *DiskCollector) Collect(ch chan<- prometheus.Metric, conn *libvirt.Connect, domain *libvirt.Domain) {
	metricsList, err := c.metricsCollector.CollectDiskStats(conn, domain)
	if err != nil {
		log.Printf("Failed to collect disk metrics: %v", err)
		return
	}

	for _, metrics := range metricsList {
		ch <- prometheus.MustNewConstMetric(c.vmDiskReadBytes, prometheus.CounterValue, float64(metrics.ReadBytes), metrics.Name, metrics.UUID, metrics.Device)
		ch <- prometheus.MustNewConstMetric(c.vmDiskWriteBytes, prometheus.CounterValue, float64(metrics.WriteBytes), metrics.Name, metrics.UUID, metrics.Device)
		ch <- prometheus.MustNewConstMetric(c.vmDiskReadOps, prometheus.CounterValue, float64(metrics.ReadOps), metrics.Name, metrics.UUID, metrics.Device)
		ch <- prometheus.MustNewConstMetric(c.vmDiskWriteOps, prometheus.CounterValue, float64(metrics.WriteOps), metrics.Name, metrics.UUID, metrics.Device)
	}
}
