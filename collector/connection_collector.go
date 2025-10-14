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
	hostname                 *prometheus.Desc
	libvirtVersion           *prometheus.Desc
	hypervisorVersion        *prometheus.Desc
	driverType               *prometheus.Desc

	// Host resource metrics
	hostCPUCount             *prometheus.Desc
	hostCPUPercent           *prometheus.Desc
	hostMemoryTotal          *prometheus.Desc
	hostMemoryFree           *prometheus.Desc

	// Storage pool metrics
	storagePoolInfo          *prometheus.Desc
	storagePoolCapacity      *prometheus.Desc
	storagePoolAllocation    *prometheus.Desc
	storagePoolAvailable     *prometheus.Desc
	storagePoolVolumes       *prometheus.Desc

	// Network pool metrics
	networkPoolInfo          *prometheus.Desc
	networkPoolBridge        *prometheus.Desc

	// Host interface metrics
	hostInterfaceRxBytes     *prometheus.Desc
	hostInterfaceTxBytes     *prometheus.Desc
	hostInterfaceRxPackets   *prometheus.Desc
	hostInterfaceTxPackets   *prometheus.Desc

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
			[]string{},
			nil,
		),
		activeDomains: prometheus.NewDesc(
			"libvirt_active_domains",
			"Number of active domains",
			[]string{},
			nil,
		),
		inactiveDomains: prometheus.NewDesc(
			"libvirt_inactive_domains",
			"Number of inactive domains",
			[]string{},
			nil,
		),
		hostname: prometheus.NewDesc(
			"libvirt_host_name",
			"Hostname of the libvirt host",
			[]string{"hostname"},
			nil,
		),
		libvirtVersion: prometheus.NewDesc(
			"libvirt_host_libvirt_version",
			"Version of libvirt",
			[]string{},
			nil,
		),
		hypervisorVersion: prometheus.NewDesc(
			"libvirt_host_hypervisor_version",
			"Version of the hypervisor",
			[]string{},
			nil,
		),
		driverType: prometheus.NewDesc(
			"libvirt_host_driver_type",
			"Type of hypervisor driver",
			[]string{"driver"},
			nil,
		),

		// Host resource metrics
		hostCPUCount: prometheus.NewDesc(
			"libvirt_host_cpu_count",
			"Number of CPU cores on the host",
			[]string{},
			nil,
		),
		hostCPUPercent: prometheus.NewDesc(
			"libvirt_host_cpu_usage_percent",
			"Host CPU usage percentage",
			[]string{},
			nil,
		),
		hostMemoryTotal: prometheus.NewDesc(
			"libvirt_host_memory_total_bytes",
			"Total memory on the host in bytes",
			[]string{},
			nil,
		),
		hostMemoryFree: prometheus.NewDesc(
			"libvirt_host_memory_free_bytes",
			"Free memory on the host in bytes",
			[]string{},
			nil,
		),

		// Storage pool metrics
		storagePoolInfo: prometheus.NewDesc(
			"libvirt_storage_pool_info",
			"Storage pool information",
			[]string{"name", "type", "state"},
			nil,
		),
		storagePoolCapacity: prometheus.NewDesc(
			"libvirt_storage_pool_capacity_bytes",
			"Storage pool capacity in bytes",
			[]string{"name"},
			nil,
		),
		storagePoolAllocation: prometheus.NewDesc(
			"libvirt_storage_pool_allocation_bytes",
			"Storage pool allocated bytes",
			[]string{"name"},
			nil,
		),
		storagePoolAvailable: prometheus.NewDesc(
			"libvirt_storage_pool_available_bytes",
			"Storage pool available bytes",
			[]string{"name"},
			nil,
		),
		storagePoolVolumes: prometheus.NewDesc(
			"libvirt_storage_pool_volumes",
			"Number of volumes in storage pool",
			[]string{"name"},
			nil,
		),

		// Network pool metrics
		networkPoolInfo: prometheus.NewDesc(
			"libvirt_network_pool_info",
			"Virtual network information",
			[]string{"name", "bridge"},
			nil,
		),
		networkPoolBridge: prometheus.NewDesc(
			"libvirt_network_pool_bridge",
			"Bridge interface for virtual network",
			[]string{"name", "bridge"},
			nil,
		),

		// Host interface metrics
		hostInterfaceRxBytes: prometheus.NewDesc(
			"libvirt_host_interface_rx_bytes",
			"Host interface received bytes",
			[]string{"interface"},
			nil,
		),
		hostInterfaceTxBytes: prometheus.NewDesc(
			"libvirt_host_interface_tx_bytes",
			"Host interface transmitted bytes",
			[]string{"interface"},
			nil,
		),
		hostInterfaceRxPackets: prometheus.NewDesc(
			"libvirt_host_interface_rx_packets",
			"Host interface received packets",
			[]string{"interface"},
			nil,
		),
		hostInterfaceTxPackets: prometheus.NewDesc(
			"libvirt_host_interface_tx_packets",
			"Host interface transmitted packets",
			[]string{"interface"},
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
	ch <- c.hostname
	ch <- c.libvirtVersion
	ch <- c.hypervisorVersion
	ch <- c.driverType

	// Host resource metrics
	ch <- c.hostCPUCount
	ch <- c.hostCPUPercent
	ch <- c.hostMemoryTotal
	ch <- c.hostMemoryFree

	// Storage pool metrics
	ch <- c.storagePoolInfo
	ch <- c.storagePoolCapacity
	ch <- c.storagePoolAllocation
	ch <- c.storagePoolAvailable
	ch <- c.storagePoolVolumes

	// Network pool metrics
	ch <- c.networkPoolInfo
	ch <- c.networkPoolBridge

	// Host interface metrics
	ch <- c.hostInterfaceRxBytes
	ch <- c.hostInterfaceTxBytes
	ch <- c.hostInterfaceRxPackets
	ch <- c.hostInterfaceTxPackets
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
		c.collectStoragePoolMetrics(ch, conn)
		c.collectNetworkPoolMetrics(ch, conn)
		c.collectHostInterfaceMetrics(ch, conn)
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
		float64(metrics.ActiveDomains),
	)

	ch <- prometheus.MustNewConstMetric(
		c.inactiveDomains,
		prometheus.GaugeValue,
		float64(metrics.DefinedDomains-metrics.ActiveDomains),
	)

	ch <- prometheus.MustNewConstMetric(
		c.hostname,
		prometheus.GaugeValue,
		1.0,
		metrics.Hostname,
	)

	ch <- prometheus.MustNewConstMetric(
		c.libvirtVersion,
		prometheus.GaugeValue,
		float64(metrics.LibvirtVersion),
	)

	ch <- prometheus.MustNewConstMetric(
		c.hypervisorVersion,
		prometheus.GaugeValue,
		float64(metrics.HypervisorVersion),
	)

	ch <- prometheus.MustNewConstMetric(
		c.driverType,
		prometheus.GaugeValue,
		1.0,
		metrics.DriverType,
	)
}

// collectHostMetrics collects host-level metrics
func (c *ConnectionCollector) collectHostMetrics(
	ch chan<- prometheus.Metric,
	conn *libvirt.Connect,
) {
	metrics, err := c.metricsCollector.CollectConnectionStats(conn)
	if err != nil {
		log.Printf("Warning: Failed to collect host metrics: %v", err)
		return
	}

	// Host resource metrics
	ch <- prometheus.MustNewConstMetric(
		c.hostCPUCount,
		prometheus.GaugeValue,
		float64(metrics.TotalCPUs),
	)

	ch <- prometheus.MustNewConstMetric(
		c.hostCPUPercent,
		prometheus.GaugeValue,
		metrics.HostCPUUsagePercent,
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
}

// collectStoragePoolMetrics collects storage pool metrics
func (c *ConnectionCollector) collectStoragePoolMetrics(
	ch chan<- prometheus.Metric,
	conn *libvirt.Connect,
) {
	metrics, err := c.metricsCollector.CollectConnectionStats(conn)
	if err != nil {
		log.Printf("Warning: Failed to collect storage pool metrics: %v", err)
		return
	}

	for _, pool := range metrics.StoragePools {
		ch <- prometheus.MustNewConstMetric(
			c.storagePoolInfo,
			prometheus.GaugeValue,
			1.0,
			pool.Name, pool.Type, pool.State,
		)

		ch <- prometheus.MustNewConstMetric(
			c.storagePoolCapacity,
			prometheus.GaugeValue,
			float64(pool.Capacity),
			pool.Name,
		)

		ch <- prometheus.MustNewConstMetric(
			c.storagePoolAllocation,
			prometheus.GaugeValue,
			float64(pool.Allocation),
			pool.Name,
		)

		ch <- prometheus.MustNewConstMetric(
			c.storagePoolAvailable,
			prometheus.GaugeValue,
			float64(pool.Available),
			pool.Name,
		)

		ch <- prometheus.MustNewConstMetric(
			c.storagePoolVolumes,
			prometheus.GaugeValue,
			float64(pool.Volumes),
			pool.Name,
		)
	}
}

// collectNetworkPoolMetrics collects virtual network pool metrics
func (c *ConnectionCollector) collectNetworkPoolMetrics(
	ch chan<- prometheus.Metric,
	conn *libvirt.Connect,
) {
	metrics, err := c.metricsCollector.CollectConnectionStats(conn)
	if err != nil {
		log.Printf("Warning: Failed to collect network pool metrics: %v", err)
		return
	}

	for _, network := range metrics.Networks {
		ch <- prometheus.MustNewConstMetric(
			c.networkPoolInfo,
			prometheus.GaugeValue,
			1.0,
			network.Name, network.Bridge,
		)

		var activeValue float64
		if network.Active {
			activeValue = 1.0
		}

		ch <- prometheus.MustNewConstMetric(
			c.networkPoolBridge,
			prometheus.GaugeValue,
			activeValue,
			network.Name, network.Bridge,
		)
	}
}

// collectHostInterfaceMetrics collects host interface metrics
func (c *ConnectionCollector) collectHostInterfaceMetrics(
	ch chan<- prometheus.Metric,
	conn *libvirt.Connect,
) {
	metrics, err := c.metricsCollector.CollectConnectionStats(conn)
	if err != nil {
		log.Printf("Warning: Failed to collect host interface metrics: %v", err)
		return
	}

	for _, iface := range metrics.Interfaces {
		ch <- prometheus.MustNewConstMetric(
			c.hostInterfaceRxBytes,
			prometheus.CounterValue,
			float64(iface.RxBytes),
			iface.Name,
		)

		ch <- prometheus.MustNewConstMetric(
			c.hostInterfaceTxBytes,
			prometheus.CounterValue,
			float64(iface.TxBytes),
			iface.Name,
		)

		ch <- prometheus.MustNewConstMetric(
			c.hostInterfaceRxPackets,
			prometheus.CounterValue,
			float64(iface.RxPackets),
			iface.Name,
		)

		ch <- prometheus.MustNewConstMetric(
			c.hostInterfaceTxPackets,
			prometheus.CounterValue,
			float64(iface.TxPackets),
			iface.Name,
		)
	}
}