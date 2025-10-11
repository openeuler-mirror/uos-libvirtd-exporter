package collector

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"libvirt.org/go/libvirt"
)

// Collector defines the interface for collecting metrics
type Collector interface {
	Describe(ch chan<- *prometheus.Desc)
	Collect(ch chan<- prometheus.Metric, conn *libvirt.Connect, domain *libvirt.Domain)
}

// DomainInfoMetrics represents raw domain information metrics
type DomainInfoMetrics struct {
	Name      string
	UUID      string
	Status    float64
	CPUTime   float64
	MemoryCurrent float64
	MemoryMax float64
	Uptime    float64
	HasUptime bool
}

// DiskMetrics represents raw disk I/O statistics
type DiskMetrics struct {
	Name      string
	UUID      string
	Device    string
	ReadBytes  uint64
	WriteBytes uint64
	ReadOps    uint64
	WriteOps   uint64
}

// NetworkMetrics represents raw network I/O statistics
type NetworkMetrics struct {
	Name     string
	UUID     string
	Interface string
	RxBytes  uint64
	TxBytes  uint64
	RxPackets uint64
	TxPackets uint64
}

// MetricsCollector defines interface for collecting raw metrics from libvirt
type MetricsCollector interface {
	CollectDomainInfo(conn *libvirt.Connect, domain *libvirt.Domain) (*DomainInfoMetrics, error)
	CollectDiskStats(conn *libvirt.Connect, domain *libvirt.Domain) ([]DiskMetrics, error)
	CollectNetworkStats(conn *libvirt.Connect, domain *libvirt.Domain) ([]NetworkMetrics, error)
}

// LibvirtMetricsCollector implements MetricsCollector to fetch raw metrics from libvirt
type LibvirtMetricsCollector struct{}

// NewLibvirtMetricsCollector creates a new LibvirtMetricsCollector
func NewLibvirtMetricsCollector() *LibvirtMetricsCollector {
	return &LibvirtMetricsCollector{}
}

// CollectDomainInfo collects basic domain information from libvirt
func (mc *LibvirtMetricsCollector) CollectDomainInfo(conn *libvirt.Connect, domain *libvirt.Domain) (*DomainInfoMetrics, error) {
	domainInfo, err := domain.GetInfo()
	if err != nil {
		return nil, err
	}

	domainName, err := domain.GetName()
	if err != nil {
		return nil, err
	}

	domainUUID, err := domain.GetUUIDString()
	if err != nil {
		return nil, err
	}

	metrics := &DomainInfoMetrics{
		Name:          domainName,
		UUID:          domainUUID,
		MemoryCurrent: float64(domainInfo.Memory) * 1024,
		MemoryMax:     float64(domainInfo.MaxMem) * 1024,
		CPUTime:       float64(domainInfo.CpuTime) / 1e9,
	}

	// VM status metric
	if domainInfo.State == libvirt.DOMAIN_RUNNING {
		metrics.Status = 1.0
	} else {
		metrics.Status = 0.0
	}

	// Only collect uptime for running domains
	if domainInfo.State == libvirt.DOMAIN_RUNNING {
		domainTime, _, err := domain.GetTime(0)
		if err == nil {
			metrics.Uptime = time.Since(time.Unix(int64(domainTime/1000), 0)).Seconds()
			metrics.HasUptime = true
		}
	}

	return metrics, nil
}

// CollectDiskStats collects disk I/O statistics from libvirt
func (mc *LibvirtMetricsCollector) CollectDiskStats(conn *libvirt.Connect, domain *libvirt.Domain) ([]DiskMetrics, error) {
	domainInfo, err := domain.GetInfo()
	if err != nil {
		return nil, err
	}

	// Only collect metrics for running domains
	if domainInfo.State != libvirt.DOMAIN_RUNNING {
		return []DiskMetrics{}, nil
	}

	domainName, err := domain.GetName()
	if err != nil {
		return nil, err
	}

	domainUUID, err := domain.GetUUIDString()
	if err != nil {
		return nil, err
	}

	var metrics []DiskMetrics

	// Use the approach from the original code to collect disk stats for common block devices
	blockDevices := []string{"vda", "vdb", "hda", "hdb", "sda", "sdb"}
	for _, device := range blockDevices {
		stats, err := domain.BlockStats(device)
		if err == nil {
			m := DiskMetrics{
				Name:       domainName,
				UUID:       domainUUID,
				Device:     device,
				ReadBytes:  uint64(stats.RdBytes),
				WriteBytes: uint64(stats.WrBytes),
				ReadOps:    uint64(stats.RdReq),
				WriteOps:   uint64(stats.WrReq),
			}
			metrics = append(metrics, m)
		}
	}

	return metrics, nil
}

// CollectNetworkStats collects network I/O statistics from libvirt
func (mc *LibvirtMetricsCollector) CollectNetworkStats(conn *libvirt.Connect, domain *libvirt.Domain) ([]NetworkMetrics, error) {
	domainInfo, err := domain.GetInfo()
	if err != nil {
		return nil, err
	}

	// Only collect metrics for running domains
	if domainInfo.State != libvirt.DOMAIN_RUNNING {
		return []NetworkMetrics{}, nil
	}

	domainName, err := domain.GetName()
	if err != nil {
		return nil, err
	}

	domainUUID, err := domain.GetUUIDString()
	if err != nil {
		return nil, err
	}

	var metrics []NetworkMetrics

	// Use the approach from the original code to collect network stats for common interfaces
	netInterfaces := []string{"vnet0", "vnet1", "eth0", "eth1"}
	for _, iface := range netInterfaces {
		stats, err := domain.InterfaceStats(iface)
		if err == nil {
			m := NetworkMetrics{
				Name:      domainName,
				UUID:      domainUUID,
				Interface: iface,
				RxBytes:   uint64(stats.RxBytes),
				TxBytes:   uint64(stats.TxBytes),
				RxPackets: uint64(stats.RxPackets),
				TxPackets: uint64(stats.TxPackets),
			}
			metrics = append(metrics, m)
		}
	}

	return metrics, nil
}

// DomainInfoCollector collects basic domain information
type DomainInfoCollector struct {
	vmStatus        *prometheus.Desc
	vmCPUTime       *prometheus.Desc
	vmMemoryCurrent *prometheus.Desc
	vmMemoryMax     *prometheus.Desc
	vmUptime        *prometheus.Desc
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
func (c *NetworkCollector) Collect(ch chan<- prometheus.Metric, conn *libvirt.Connect, domain *libvirt.Domain) {
	metricsList, err := c.metricsCollector.CollectNetworkStats(conn, domain)
	if err != nil {
		log.Printf("Failed to collect network metrics: %v", err)
		return
	}

	for _, metrics := range metricsList {
		ch <- prometheus.MustNewConstMetric(c.vmNetworkRxBytes, prometheus.CounterValue, float64(metrics.RxBytes), metrics.Name, metrics.UUID, metrics.Interface)
		ch <- prometheus.MustNewConstMetric(c.vmNetworkTxBytes, prometheus.CounterValue, float64(metrics.TxBytes), metrics.Name, metrics.UUID, metrics.Interface)
		ch <- prometheus.MustNewConstMetric(c.vmNetworkRxPkts, prometheus.CounterValue, float64(metrics.RxPackets), metrics.Name, metrics.UUID, metrics.Interface)
		ch <- prometheus.MustNewConstMetric(c.vmNetworkTxPkts, prometheus.CounterValue, float64(metrics.TxPackets), metrics.Name, metrics.UUID, metrics.Interface)
	}
}

// LibvirtCollector implements the prometheus.Collector interface
type LibvirtCollector struct {
	uri          string
	conn         *libvirt.Connect
	mutex        sync.RWMutex
	collectors   []Collector
	reconnectErr chan error
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

	collector := &LibvirtCollector{
		uri:          uri,
		conn:         conn,
		reconnectErr: make(chan error),
	}

	// Initialize individual collectors
	collector.collectors = append(collector.collectors, NewDomainInfoCollector())
	collector.collectors = append(collector.collectors, NewDiskCollector())
	collector.collectors = append(collector.collectors, NewNetworkCollector())

	return collector, nil
}

// Describe implements the prometheus.Collector interface
func (c *LibvirtCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, collector := range c.collectors {
		collector.Describe(ch)
	}
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
		// Use individual collectors to gather metrics
		for _, collector := range c.collectors {
			collector.Collect(ch, c.conn, &domain)
		}
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