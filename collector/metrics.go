package collector

import (
	"libvirt.org/go/libvirt"
)

// DomainInfoMetrics represents raw domain information metrics
type DomainInfoMetrics struct {
	Name          string
	UUID          string
	Status        float64
	CPUTime       float64
	MemoryCurrent float64
	MemoryMax     float64
	Uptime        float64
	HasUptime     bool
}

// DiskMetrics represents raw disk I/O statistics
type DiskMetrics struct {
	Name       string
	UUID       string
	Device     string
	ReadBytes  uint64
	WriteBytes uint64
	ReadOps    uint64
	WriteOps   uint64
}

// NetworkMetrics represents raw network I/O statistics
type NetworkMetrics struct {
	Name      string
	UUID      string
	Interface string
	RxBytes   uint64
	TxBytes   uint64
	RxPackets uint64
	TxPackets uint64
}

// MetricsCollector defines interface for collecting raw metrics from libvirt
type MetricsCollector interface {
	CollectDomainInfo(
		conn *libvirt.Connect,
		domain *libvirt.Domain,
	) (*DomainInfoMetrics, error)
	CollectDiskStats(
		conn *libvirt.Connect,
		domain *libvirt.Domain,
	) ([]DiskMetrics, error)
	CollectNetworkStats(
		conn *libvirt.Connect,
		domain *libvirt.Domain,
	) ([]NetworkMetrics, error)
}
