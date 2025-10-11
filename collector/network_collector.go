package collector

import (
	"log"

	"github.com/prometheus/client_golang/prometheus"
	"libvirt.org/go/libvirt"
)

// NetworkCollector collects network I/O statistics
type NetworkCollector struct {
	vmNetworkRxBytes *prometheus.Desc
	vmNetworkTxBytes *prometheus.Desc
	vmNetworkRxPkts  *prometheus.Desc
	vmNetworkTxPkts  *prometheus.Desc
	metricsCollector MetricsCollector
}

// NewNetworkCollector creates a new NetworkCollector
func NewNetworkCollector() *NetworkCollector {
	return &NetworkCollector{
		vmNetworkRxBytes: prometheus.NewDesc(
			"libvirt_vm_network_rx_bytes_total",
			"Total network bytes received by the virtual machine",
			[]string{"domain", "uuid", "interface"},
			nil,
		),
		vmNetworkTxBytes: prometheus.NewDesc(
			"libvirt_vm_network_tx_bytes_total",
			"Total network bytes transmitted by the virtual machine",
			[]string{"domain", "uuid", "interface"},
			nil,
		),
		vmNetworkRxPkts: prometheus.NewDesc(
			"libvirt_vm_network_rx_packets_total",
			"Total network packets received by the virtual machine",
			[]string{"domain", "uuid", "interface"},
			nil,
		),
		vmNetworkTxPkts: prometheus.NewDesc(
			"libvirt_vm_network_tx_packets_total",
			"Total network packets transmitted by the virtual machine",
			[]string{"domain", "uuid", "interface"},
			nil,
		),
		metricsCollector: NewLibvirtMetricsCollector(),
	}
}

// Describe implements the prometheus.Collector interface for NetworkCollector
func (c *NetworkCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.vmNetworkRxBytes
	ch <- c.vmNetworkTxBytes
	ch <- c.vmNetworkRxPkts
	ch <- c.vmNetworkTxPkts
}

// Collect implements the Collector interface for NetworkCollector
func (c *NetworkCollector) Collect(
	ch chan<- prometheus.Metric,
	conn *libvirt.Connect,
	domain *libvirt.Domain,
) {
	metricsList, err := c.metricsCollector.CollectNetworkStats(conn, domain)
	if err != nil {
		log.Printf("Failed to collect network metrics: %v", err)
		return
	}

	for _, metrics := range metricsList {
		ch <- prometheus.MustNewConstMetric(
			c.vmNetworkRxBytes,
			prometheus.CounterValue,
			float64(metrics.RxBytes),
			metrics.Name,
			metrics.UUID,
			metrics.Interface,
		)

		ch <- prometheus.MustNewConstMetric(
			c.vmNetworkTxBytes,
			prometheus.CounterValue,
			float64(metrics.TxBytes),
			metrics.Name,
			metrics.UUID,
			metrics.Interface,
		)

		ch <- prometheus.MustNewConstMetric(
			c.vmNetworkRxPkts,
			prometheus.CounterValue,
			float64(metrics.RxPackets),
			metrics.Name,
			metrics.UUID,
			metrics.Interface,
		)

		ch <- prometheus.MustNewConstMetric(
			c.vmNetworkTxPkts,
			prometheus.CounterValue,
			float64(metrics.TxPackets),
			metrics.Name,
			metrics.UUID,
			metrics.Interface,
		)
	}
}
