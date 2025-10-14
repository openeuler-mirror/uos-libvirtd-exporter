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
	vmNetworkRxErrs  *prometheus.Desc
	vmNetworkTxErrs  *prometheus.Desc
	vmNetworkRxDrop  *prometheus.Desc
	vmNetworkTxDrop  *prometheus.Desc
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
		vmNetworkRxErrs: prometheus.NewDesc(
			"libvirt_vm_network_rx_errors_total",
			"Total network receive errors by the virtual machine",
			[]string{"domain", "uuid", "interface"},
			nil,
		),
		vmNetworkTxErrs: prometheus.NewDesc(
			"libvirt_vm_network_tx_errors_total",
			"Total network transmit errors by the virtual machine",
			[]string{"domain", "uuid", "interface"},
			nil,
		),
		vmNetworkRxDrop: prometheus.NewDesc(
			"libvirt_vm_network_rx_dropped_total",
			"Total network receive packets dropped by the virtual machine",
			[]string{"domain", "uuid", "interface"},
			nil,
		),
		vmNetworkTxDrop: prometheus.NewDesc(
			"libvirt_vm_network_tx_dropped_total",
			"Total network transmit packets dropped by the virtual machine",
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
	ch <- c.vmNetworkRxErrs
	ch <- c.vmNetworkTxErrs
	ch <- c.vmNetworkRxDrop
	ch <- c.vmNetworkTxDrop
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

		ch <- prometheus.MustNewConstMetric(
			c.vmNetworkRxErrs,
			prometheus.CounterValue,
			float64(metrics.RxErrors),
			metrics.Name,
			metrics.UUID,
			metrics.Interface,
		)

		ch <- prometheus.MustNewConstMetric(
			c.vmNetworkTxErrs,
			prometheus.CounterValue,
			float64(metrics.TxErrors),
			metrics.Name,
			metrics.UUID,
			metrics.Interface,
		)

		ch <- prometheus.MustNewConstMetric(
			c.vmNetworkRxDrop,
			prometheus.CounterValue,
			float64(metrics.RxDrops),
			metrics.Name,
			metrics.UUID,
			metrics.Interface,
		)

		ch <- prometheus.MustNewConstMetric(
			c.vmNetworkTxDrop,
			prometheus.CounterValue,
			float64(metrics.TxDrops),
			metrics.Name,
			metrics.UUID,
			metrics.Interface,
		)
	}
}
