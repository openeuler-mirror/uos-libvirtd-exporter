package collector

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"libvirt.org/go/libvirt"
)

// LibvirtCollector implements the prometheus.Collector interface
type LibvirtCollector struct {
	uri   string
	conn  *libvirt.Connect
	mutex sync.RWMutex

	// VM status metrics
	vmStatus *prometheus.Desc

	// CPU metrics
	vmCPUTime *prometheus.Desc

	// Memory metrics
	vmMemoryCurrent *prometheus.Desc
	vmMemoryMax     *prometheus.Desc

	// Disk I/O metrics
	vmDiskReadBytes  *prometheus.Desc
	vmDiskWriteBytes *prometheus.Desc
	vmDiskReadOps    *prometheus.Desc
	vmDiskWriteOps   *prometheus.Desc

	// Network I/O metrics
	vmNetworkRxBytes *prometheus.Desc
	vmNetworkTxBytes *prometheus.Desc
	vmNetworkRxPkts  *prometheus.Desc
	vmNetworkTxPkts  *prometheus.Desc

	// Uptime metrics
	vmUptime *prometheus.Desc
}

// NewLibvirtCollector creates a new LibvirtCollector
func NewLibvirtCollector(uri string) (*LibvirtCollector, error) {
	log.Printf("Connecting to libvirt at '%s'", uri)
	conn, err := libvirt.NewConnect(uri)
	if err != nil {
		return nil, err
	}

	alive, err := conn.IsAlive()
	if err != nil || !alive {
		return nil, fmt.Errorf("connection is not alive")
	}

	log.Println("Successfully connected to libvirt")

	return &LibvirtCollector{
		uri:  uri,
		conn: conn,

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

		vmUptime: prometheus.NewDesc(
			"libvirt_vm_uptime_seconds",
			"Virtual machine uptime in seconds",
			[]string{"domain", "uuid"},
			nil,
		),
	}, nil
}

// Describe implements the prometheus.Collector interface
func (c *LibvirtCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.vmStatus
	ch <- c.vmCPUTime
	ch <- c.vmMemoryCurrent
	ch <- c.vmMemoryMax
	ch <- c.vmDiskReadBytes
	ch <- c.vmDiskWriteBytes
	ch <- c.vmDiskReadOps
	ch <- c.vmDiskWriteOps
	ch <- c.vmNetworkRxBytes
	ch <- c.vmNetworkTxBytes
	ch <- c.vmNetworkRxPkts
	ch <- c.vmNetworkTxPkts
	ch <- c.vmUptime
}

// Collect implements the prometheus.Collector interface
func (c *LibvirtCollector) Collect(ch chan<- prometheus.Metric) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Check connection health
	alive, err := c.conn.IsAlive()
	if err != nil || !alive {
		log.Printf("Warning: Connection to libvirt lost, reconnecting...")
		c.conn.Close()

		conn, err := libvirt.NewConnect(c.uri)
		if err != nil {
			log.Printf("Error: Failed to reconnect to libvirt: %v", err)
			return
		}
		c.conn = conn
		log.Println("Successfully reconnected to libvirt")
	}

	// Get all domains
	domains, err := c.conn.ListAllDomains(libvirt.CONNECT_LIST_DOMAINS_ACTIVE | libvirt.CONNECT_LIST_DOMAINS_INACTIVE)
	if err != nil {
		log.Printf("Error: Failed to list domains: %v", err)
		return
	}
	defer func() {
		for _, domain := range domains {
			domain.Free()
		}
	}()

	for _, domain := range domains {
		c.collectDomainMetrics(ch, &domain)
	}
}

func (c *LibvirtCollector) collectDomainMetrics(ch chan<- prometheus.Metric, domain *libvirt.Domain) {
	domainInfo, err := domain.GetInfo()
	if err != nil {
		log.Printf("Failed to get domain info: %v", err)
		return
	}

	domainName, err := domain.GetName()
	if err != nil {
		log.Printf("Failed to get domain name: %v", err)
		return
	}

	domainUUID, err := domain.GetUUIDString()
	if err != nil {
		log.Printf("Failed to get domain UUID: %v", err)
		return
	}

	// VM status metric
	status := 0.0
	if domainInfo.State == libvirt.DOMAIN_RUNNING {
		status = 1.0
	}
	ch <- prometheus.MustNewConstMetric(c.vmStatus, prometheus.GaugeValue, status, domainName, domainUUID)

	// CPU time metric (convert from nanoseconds to seconds)
	ch <- prometheus.MustNewConstMetric(c.vmCPUTime, prometheus.CounterValue, float64(domainInfo.CpuTime)/1e9, domainName, domainUUID)

	// Memory metrics
	ch <- prometheus.MustNewConstMetric(c.vmMemoryCurrent, prometheus.GaugeValue, float64(domainInfo.Memory)*1024, domainName, domainUUID)
	ch <- prometheus.MustNewConstMetric(c.vmMemoryMax, prometheus.GaugeValue, float64(domainInfo.MaxMem)*1024, domainName, domainUUID)

	// Only collect additional metrics for running domains
	if domainInfo.State == libvirt.DOMAIN_RUNNING {
		c.collectRunningDomainMetrics(ch, domain, domainName, domainUUID)
	}
}

func (c *LibvirtCollector) collectRunningDomainMetrics(ch chan<- prometheus.Metric, domain *libvirt.Domain, domainName, domainUUID string) {
	// Collect disk I/O statistics for common block devices
	blockDevices := []string{"vda", "vdb", "hda", "hdb", "sda", "sdb"}
	for _, device := range blockDevices {
		stats, err := domain.BlockStats(device)
		if err == nil {
			ch <- prometheus.MustNewConstMetric(c.vmDiskReadBytes, prometheus.CounterValue, float64(stats.RdBytes), domainName, domainUUID, device)
			ch <- prometheus.MustNewConstMetric(c.vmDiskWriteBytes, prometheus.CounterValue, float64(stats.WrBytes), domainName, domainUUID, device)
			ch <- prometheus.MustNewConstMetric(c.vmDiskReadOps, prometheus.CounterValue, float64(stats.RdReq), domainName, domainUUID, device)
			ch <- prometheus.MustNewConstMetric(c.vmDiskWriteOps, prometheus.CounterValue, float64(stats.WrReq), domainName, domainUUID, device)
		}
	}

	// Collect network I/O statistics for common network interfaces
	netInterfaces := []string{"vnet0", "vnet1", "eth0", "eth1"}
	for _, iface := range netInterfaces {
		stats, err := domain.InterfaceStats(iface)
		if err == nil {
			ch <- prometheus.MustNewConstMetric(c.vmNetworkRxBytes, prometheus.CounterValue, float64(stats.RxBytes), domainName, domainUUID, iface)
			ch <- prometheus.MustNewConstMetric(c.vmNetworkTxBytes, prometheus.CounterValue, float64(stats.TxBytes), domainName, domainUUID, iface)
			ch <- prometheus.MustNewConstMetric(c.vmNetworkRxPkts, prometheus.CounterValue, float64(stats.RxPackets), domainName, domainUUID, iface)
			ch <- prometheus.MustNewConstMetric(c.vmNetworkTxPkts, prometheus.CounterValue, float64(stats.TxPackets), domainName, domainUUID, iface)
		}
	}

	// Collect uptime (simplified - using current time minus start time)
	domainTime, _, err := domain.GetTime(0)
	if err == nil {
		uptime := time.Since(time.Unix(int64(domainTime/1000), 0)).Seconds()
		ch <- prometheus.MustNewConstMetric(c.vmUptime, prometheus.GaugeValue, uptime, domainName, domainUUID)
	}
}

// Close closes the libvirt connection
func (c *LibvirtCollector) Close() {
	if c.conn != nil {
		log.Println("Closing libvirt connection...")
		c.conn.Close()
		log.Println("Libvirt connection closed")
	}
}