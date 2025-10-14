package collector

import (
	"log"

	"github.com/prometheus/client_golang/prometheus"
	"libvirt.org/go/libvirt"
)

// DeviceCollector collects device statistics
type DeviceCollector struct {
	vmHasTPM         *prometheus.Desc
	vmHasRNG         *prometheus.Desc
	vmSnapshotCount  *prometheus.Desc
	metricsCollector MetricsCollector
}

// NewDeviceCollector creates a new DeviceCollector
func NewDeviceCollector() *DeviceCollector {
	return &DeviceCollector{
		vmHasTPM: prometheus.NewDesc(
			"libvirt_vm_has_tpm",
			"Whether the virtual machine has a TPM device",
			[]string{"domain", "uuid"},
			nil,
		),
		vmHasRNG: prometheus.NewDesc(
			"libvirt_vm_has_rng",
			"Whether the virtual machine has an RNG device",
			[]string{"domain", "uuid"},
			nil,
		),
		vmSnapshotCount: prometheus.NewDesc(
			"libvirt_vm_snapshot_count",
			"Number of snapshots for the virtual machine",
			[]string{"domain", "uuid"},
			nil,
		),
		metricsCollector: NewLibvirtMetricsCollector(),
	}
}

// Describe implements the prometheus.Collector interface for DeviceCollector
func (c *DeviceCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.vmHasTPM
	ch <- c.vmHasRNG
	ch <- c.vmSnapshotCount
}

// Collect implements the Collector interface for DeviceCollector
func (c *DeviceCollector) Collect(
	ch chan<- prometheus.Metric,
	conn *libvirt.Connect,
	domain *libvirt.Domain,
) {
	// Collect device stats
	deviceMetrics, err := c.metricsCollector.CollectDeviceStats(conn, domain)
	if err != nil {
		log.Printf("Failed to collect device metrics: %v", err)
	} else {
		var tpmValue float64
		if deviceMetrics.HasTPM {
			tpmValue = 1.0
		}

		var rngValue float64
		if deviceMetrics.HasRNG {
			rngValue = 1.0
		}

		ch <- prometheus.MustNewConstMetric(
			c.vmHasTPM,
			prometheus.GaugeValue,
			tpmValue,
			deviceMetrics.Name,
			deviceMetrics.UUID,
		)

		ch <- prometheus.MustNewConstMetric(
			c.vmHasRNG,
			prometheus.GaugeValue,
			rngValue,
			deviceMetrics.Name,
			deviceMetrics.UUID,
		)
	}

	// Collect snapshot stats
	snapshotMetrics, err := c.metricsCollector.CollectSnapshotStats(conn, domain)
	if err != nil {
		log.Printf("Failed to collect snapshot metrics: %v", err)
	} else {
		ch <- prometheus.MustNewConstMetric(
			c.vmSnapshotCount,
			prometheus.GaugeValue,
			float64(snapshotMetrics.Count),
			snapshotMetrics.Name,
			snapshotMetrics.UUID,
		)
	}
}
